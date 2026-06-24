package grouper

import (
	"context"
	"log/slog"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uber/h3-go/v4"
)

type EventPublisher interface {
	BroadcastMulti([]string, realtime.Event)
}

// Engine holds the grouper's dependencies.
type Engine struct {
	Pool   *pgxpool.Pool
	Events EventPublisher
}

// Run processes all pending ride requests in one cycle:
// 1. Fetch pending requests inside a locked transaction (FOR UPDATE SKIP LOCKED)
// 2. Bucket by H3 pickup cell
// 3. Filter buckets by arrive_by compatibility (2-hour window)
// 4. 3-pass spatial matching: exact cell → k-ring=1 → k-ring=2
// 5. Score each candidate group and persist the best ones
func (e *Engine) Run(ctx context.Context) {
	tx, err := e.Pool.Begin(ctx)
	if err != nil {
		slog.Error("grouper: failed to begin tx", "error", err)
		return
	}
	defer tx.Rollback(ctx)

	reqStore := &store.RequestStore{DB: tx}
	grpStore := &store.GroupStore{DB: tx}

	pending, err := reqStore.FetchPendingLocked(ctx)
	if err != nil {
		slog.Error("grouper: failed to fetch pending", "error", err)
		return
	}
	fastTrack, normal := splitByUrgency(pending)
	var formedEvents []realtime.Event

	if len(fastTrack) >= 2 {
		// skip rerank hook
		groups := e.processPool(ctx, fastTrack, 2, 4)
		formedEvents = append(formedEvents, e.persistGroups(ctx, grpStore, groups, models.PriorityHigh)...)
	}
	if len(normal) >= 2 {
		groups := e.processPool(ctx, normal, 2, 4)
		groups = batchReRank(groups)
		formedEvents = append(formedEvents, e.persistGroups(ctx, grpStore, groups, models.PriorityNormal)...)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("grouper: failed to commit", "error", err)
		return
	}

	if e.Events != nil {
		for _, event := range formedEvents {
			e.Events.BroadcastMulti([]string{"global", event.GroupID}, event)
		}
	}

	slog.Info("grouper: cycle complete", "total_pending", len(pending), "events", len(formedEvents))
}

func (e *Engine) persistGroups(ctx context.Context, gs *store.GroupStore, groups []FormedGroup, priority string) []realtime.Event {
	var events []realtime.Event
	for _, fg := range groups {
		var reqs []models.RideRequest
		for _, m := range fg.Members {
			reqs = append(reqs, models.RideRequest{
				ID:       m.StudentID,
				ArriveBy: m.ArriveBy,
			})
		}
		groupID, err := gs.Create(ctx, reqs, fg.Score, priority)
		if err != nil {
			slog.Error("grouper: failed to create group", "error", err)
			continue
		}
		slog.Info("grouper: group formed", "group_id", groupID, "members", len(fg.Members), "score", fg.Score, "priority", priority)
		events = append(events, realtime.GroupFormed(groupID, len(fg.Members), fg.Score))
	}
	return events
}

func (e *Engine) processPool(ctx context.Context, pending []models.RideRequest, minSize, maxSize int) []FormedGroup {
	buckets := make(map[int64][]models.RideRequest)
	now := time.Now()
	for _, r := range pending {
		if now.After(r.ArriveBy) {
			continue
		}
		if r.PickupH3 != nil {
			buckets[*r.PickupH3] = append(buckets[*r.PickupH3], r)
		}
	}

	assigned := make(map[string]bool)
	var groups []FormedGroup

	for _, bucket := range buckets {
		groups = append(groups, greedyFormation(bucket, assigned, minSize, maxSize)...)
	}

	for cell, bucket := range buckets {
		neighbors, err := h3.GridDisk(h3.Cell(cell), 1)
		if err != nil {
			continue
		}
		var pool []models.RideRequest
		for _, n := range neighbors {
			for _, r := range buckets[int64(n)] {
				if !assigned[r.ID] {
					pool = append(pool, r)
				}
			}
		}
		_ = bucket
		groups = append(groups, greedyFormation(pool, assigned, minSize, maxSize)...)
	}

	for cell, bucket := range buckets {
		neighbors, err := h3.GridDisk(h3.Cell(cell), 2)
		if err != nil {
			continue
		}
		var pool []models.RideRequest
		for _, n := range neighbors {
			for _, r := range buckets[int64(n)] {
				if !assigned[r.ID] {
					pool = append(pool, r)
				}
			}
		}
		_ = bucket
		groups = append(groups, greedyFormation(pool, assigned, minSize, maxSize)...)
	}

	return groups
}

// greedyFormation finds the best compatible groups from the pool without persisting.
func greedyFormation(pool []models.RideRequest, assigned map[string]bool, minSize, maxSize int) []FormedGroup {
	var formed []FormedGroup
	for {
		group := bestGroup(pool, assigned, minSize, maxSize)
		if len(group) < minSize {
			break
		}
		score := computeRouteScoreForGroup(group)
		if score <= 0 {
			break
		}
		res := CheckCorridor(group[0], group[len(group)-1])

		var members []RequestMember
		for _, r := range group {
			members = append(members, RequestMember{
				StudentID:  r.ID,
				PickupLat:  r.PickupLat,
				PickupLng:  r.PickupLng,
				DropoffLat: r.DropoffLat,
				DropoffLng: r.DropoffLng,
				ArriveBy:   r.ArriveBy,
			})
			assigned[r.ID] = true
		}

		formed = append(formed, FormedGroup{
			Members:   members,
			GroupType: res.Type,
			Score:     score,
		})
	}
	return formed
}

// bestGroup picks the best compatible group of size between minSize and maxSize from the current pool.
func bestGroup(pool []models.RideRequest, assigned map[string]bool, minSize, maxSize int) []models.RideRequest {
	var best []models.RideRequest
	var bestScore float64 = -9999

	for i := 0; i < len(pool); i++ {
		if assigned[pool[i].ID] {
			continue
		}
		for j := i + 1; j < len(pool); j++ {
			if assigned[pool[j].ID] {
				continue
			}
			candidate2 := []models.RideRequest{pool[i], pool[j]}
			if !groupCompatible(candidate2) {
				continue
			}
			if minSize <= 2 && 2 <= maxSize {
				if s := computeRouteScoreForGroup(candidate2); s > bestScore {
					bestScore = s
					best = candidate2
				}
			}

			if maxSize < 3 {
				continue
			}
			for k := j + 1; k < len(pool); k++ {
				if assigned[pool[k].ID] {
					continue
				}
				candidate3 := []models.RideRequest{pool[i], pool[j], pool[k]}
				if !groupCompatible(candidate3) {
					continue
				}
				if minSize <= 3 && 3 <= maxSize {
					if s := computeRouteScoreForGroup(candidate3); s > bestScore {
						bestScore = s
						best = candidate3
					}
				}

				if maxSize < 4 {
					continue
				}
				for l := k + 1; l < len(pool); l++ {
					if assigned[pool[l].ID] {
						continue
					}
					candidate4 := []models.RideRequest{pool[i], pool[j], pool[k], pool[l]}
					if !groupCompatible(candidate4) {
						continue
					}
					if minSize <= 4 && 4 <= maxSize {
						if s := computeRouteScoreForGroup(candidate4); s > bestScore {
							bestScore = s
							best = candidate4
						}
					}
				}
			}
		}
	}

	return best
}

func computeRouteScoreForGroup(group []models.RideRequest) float64 {
	if len(group) < 2 {
		return 0
	}
	res := CheckCorridor(group[0], group[len(group)-1])
	var members []RequestMember
	for _, r := range group {
		members = append(members, RequestMember{
			StudentID:  r.ID,
			PickupLat:  r.PickupLat,
			PickupLng:  r.PickupLng,
			DropoffLat: r.DropoffLat,
			DropoffLng: r.DropoffLng,
			ArriveBy:   r.ArriveBy,
		})
	}
	in := RouteScoreInput{
		Members:   members,
		GroupType: res.Type,
	}
	return ComputeRouteScore(in)
}

func groupCompatible(group []models.RideRequest) bool {
	for i := 0; i < len(group); i++ {
		for j := i + 1; j < len(group); j++ {
			if !timeCompatible(group[i], group[j]) {
				return false
			}
			res := CheckCorridor(group[i], group[j])
			if !res.Compatible {
				return false
			}
		}
	}
	return true
}

// timeCompatible returns true if two requests have arrive_by within a 2-hour window.
func timeCompatible(r1, r2 models.RideRequest) bool {
	diff := r1.ArriveBy.Sub(r2.ArriveBy).Abs()
	return diff <= 2*time.Hour
}
