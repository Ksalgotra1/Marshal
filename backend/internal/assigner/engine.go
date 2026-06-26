package assigner

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/dispatch"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventPublisher interface {
	BroadcastMulti([]string, realtime.Event)
}

type Dispatcher interface {
	SendDispatch(ctx context.Context, groupID, msg string) (int, error)
}

// Run processes all assignable groups in one cycle:
// 1. Cancel groups that exceeded max retries (3 attempts)
// 2. Revert timed-out dispatching groups back to 'grouped'
// 3. Pop highest-score group from priority queue
// 4. Mark it as 'dispatching' — drivers can now claim it
func Run(ctx context.Context, pool *pgxpool.Pool, events EventPublisher, bot Dispatcher, ttlMinutes int) {
	gs := &store.GroupStore{DB: pool}

	// Step 1: Cancel groups with 3+ failed attempts
	cancelled, err := gs.CancelMaxRetries(ctx)
	if err != nil {
		slog.Error("assigner: cancel max retries failed", "error", err)
	} else if cancelled > 0 {
		slog.Warn("assigner: cancelled groups with max retries", "count", cancelled)
	}

	// Step 2: Revert timed-out dispatching groups
	reverted, err := gs.RevertTimedOut(ctx)
	if err != nil {
		slog.Error("assigner: revert timed out failed", "error", err)
	} else if len(reverted) > 0 {
		slog.Info("assigner: reverted timed-out groups", "count", len(reverted), "ids", reverted)
	}

	ds := &store.DriverStore{DB: pool}
	if n, err := ds.MarkStaleOffline(ctx, ttlMinutes); err != nil {
		slog.Error("assigner: mark stale offline failed", "error", err)
	} else if n > 0 {
		slog.Info("assigner: marked drivers offline (stale)", "count", n)
	}

	count, err := ds.CountOnline(ctx, ttlMinutes)
	if err != nil {
		slog.Error("assigner: failed to count online drivers", "error", err)
		return
	}
	if count == 0 {
		slog.Info("assigner: no online drivers, skipping cycle")
		return
	}

	// Step 3: Pop and dispatch all available groups
	for {
		if ctx.Err() != nil {
			return
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			slog.Error("assigner: begin tx failed", "error", err)
			return
		}

		txGS := &store.GroupStore{DB: tx}
		group, err := txGS.PopHighestScore(ctx)
		if err != nil {
			tx.Rollback(ctx)
			slog.Error("assigner: pop failed", "error", err)
			return
		}
		if group == nil {
			tx.Rollback(ctx)
			return // no more groups to dispatch
		}

		// Mark as dispatching
		if err := txGS.MarkDispatching(ctx, group.ID); err != nil {
			tx.Rollback(ctx)
			slog.Error("assigner: mark dispatching failed", "group_id", group.ID, "error", err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("assigner: commit failed", "group_id", group.ID, "error", err)
			return
		}

		timeout := DynamicTimeout(group.ArriveBy)

		gs2 := &store.GroupStore{DB: pool}
		_ = gs2.SetDispatchTimeoutAt(ctx, group.ID, time.Now().Add(timeout))

		detail, err := gs2.GetByIDWithMembers(ctx, group.ID)
		if err != nil {
			slog.Error("assigner: failed to fetch group members", "error", err)
		} else {
			_, _, msg, err := dispatch.GenerateMessage(detail.Members)
			if err == nil {
				if bot != nil {
					msgID, err := bot.SendDispatch(ctx, group.ID, msg)
					if err != nil {
						slog.Error("assigner: telegram send failed", "error", err)
					} else {
						_ = gs2.SetTelegramMsgID(ctx, group.ID, msgID)
					}
				}
			}
		}

		slog.Info("assigner: group dispatching",
			"group_id", group.ID,
			"score", group.RouteScore,
			"attempt", group.DispatchAttempts+1,
			"timeout", timeout,
		)

		if events != nil {
			event := realtime.GroupDispatching(group.ID, group.DispatchAttempts+1, group.RouteScore)
			events.BroadcastMulti([]string{"global", group.ID}, event)
		}
	}
}

// DynamicTimeout computes the dispatch timeout based on urgency.
// Closer arrive_by = shorter timeout = more urgency.
//
//	timeout = clamp(2min, 10min, (arrive_by - now) × 0.1)
//
// Examples:
//
//	8h away  → 10min (capped)
//	1h away  → 6min
//	30m away → 3min
//	20m away → 2min (floored)
func DynamicTimeout(arriveBy time.Time) time.Duration {
	remaining := time.Until(arriveBy).Minutes()
	timeoutMin := remaining * 0.1
	timeoutMin = math.Max(2.0, math.Min(10.0, timeoutMin))
	return time.Duration(math.Ceil(timeoutMin)) * time.Minute
}

// NewProcess creates a worker.ProcessFunc with event publishing injected.
func NewProcess(events EventPublisher, bot Dispatcher, ttlMinutes int) worker.ProcessFunc {
	return func(ctx context.Context, pool *pgxpool.Pool, payload []byte) error {
		Run(ctx, pool, events, bot, ttlMinutes)
		worker.Notify(ctx, pool, "assigner_wakeup")
		return nil
	}
}
