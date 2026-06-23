package grouper

import (
	"context"
	"log/slog"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/math"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uber/h3-go/v4"
)

type EventPublisher interface {
	BroadcastMulti([]string, realtime.Event)
}

const targetGroupSize = 3

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
	if len(pending) < targetGroupSize {
		return
	}

	// Bucket requests by H3 pickup cell
	buckets := make(map[int64][]models.RideRequest)
	now := time.Now()
	for _, r := range pending {
		// Ignore stale requests that are already past their arrive_by time
		if now.After(r.ArriveBy) {
			continue
		}
		if r.PickupH3 != nil {
			buckets[*r.PickupH3] = append(buckets[*r.PickupH3], r)
		}
	}

	assigned := make(map[string]bool)
	var formedEvents []realtime.Event

	// PASS 1 — Exact H3 cell match
	for _, bucket := range buckets {
		formedEvents = append(formedEvents, e.matchAndCreate(ctx, grpStore, bucket, assigned, "exact")...)
	}

	// PASS 2 — k-ring=1 expansion (~870m)
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
		formedEvents = append(formedEvents, e.matchAndCreate(ctx, grpStore, pool, assigned, "k1")...)
	}

	// PASS 3 — k-ring=2 expansion (~1.7km)
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
		formedEvents = append(formedEvents, e.matchAndCreate(ctx, grpStore, pool, assigned, "k2")...)
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

	slog.Info("grouper: cycle complete", "total_pending", len(pending), "members_grouped", len(assigned))
}

// matchAndCreate finds the best compatible group from the pool and persists it.
func (e *Engine) matchAndCreate(ctx context.Context, gs *store.GroupStore, pool []models.RideRequest, assigned map[string]bool, pass string) []realtime.Event {
	var events []realtime.Event
	for {
		group := bestGroup(pool, assigned)
		if len(group) < targetGroupSize {
			break
		}
		score := computeRouteScoreForGroup(group)
		groupID, err := gs.Create(ctx, group, score)
		if err != nil {
			slog.Error("grouper: failed to create group", "pass", pass, "error", err)
			break
		}
		for _, r := range group {
			assigned[r.ID] = true
		}
		slog.Info("grouper: group formed", "group_id", groupID, "members", len(group), "score", score, "pass", pass)

		events = append(events, realtime.GroupFormed(groupID, len(group), score))
	}
	return events
}

// bestGroup picks the best compatible 3-request group from the current pool.
func bestGroup(pool []models.RideRequest, assigned map[string]bool) []models.RideRequest {
	var best []models.RideRequest
	var bestScore float64 = -9999

	for i, r1 := range pool {
		if assigned[r1.ID] {
			continue
		}
		for j := i + 1; j < len(pool); j++ {
			r2 := pool[j]
			if assigned[r2.ID] {
				continue
			}
			for k := j + 1; k < len(pool); k++ {
				r3 := pool[k]
				if assigned[r3.ID] {
					continue
				}
				candidate := []models.RideRequest{r1, r2, r3}
				if !groupCompatible(candidate) {
					continue
				}
				if s := computeRouteScoreForGroup(candidate); s > bestScore {
					bestScore = s
					best = candidate
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
			if !dropoffCompatible(group[i], group[j]) || !timeCompatible(group[i], group[j]) {
				return false
			}
		}
	}
	return true
}

// dropoffCompatible returns true if two requests share a roughly similar dropoff.
// Uses H3 grid distance if available, falls back to Haversine.
func dropoffCompatible(r1, r2 models.RideRequest) bool {
	if r1.DropoffH3 != nil && r2.DropoffH3 != nil {
		dist, err := h3.GridDistance(h3.Cell(*r1.DropoffH3), h3.Cell(*r2.DropoffH3))
		if err == nil && dist <= 5 { // ~870m at resolution 9
			return true
		}
	}
	return math.Distance(r1.DropoffLat, r1.DropoffLng, r2.DropoffLat, r2.DropoffLng) <= 1.5
}

// timeCompatible returns true if two requests have arrive_by within a 2-hour window.
func timeCompatible(r1, r2 models.RideRequest) bool {
	diff := r1.ArriveBy.Sub(r2.ArriveBy).Abs()
	return diff <= 2*time.Hour
}
