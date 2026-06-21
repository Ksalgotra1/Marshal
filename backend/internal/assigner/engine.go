package assigner

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/worker"
	"github.com/Ksalgotra1/Marshal/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run processes all assignable groups in one cycle:
// 1. Cancel groups that exceeded max retries (3 attempts)
// 2. Revert timed-out dispatching groups back to 'grouped'
// 3. Pop highest-score group from priority queue
// 4. Mark it as 'dispatching' — drivers can now claim it
func Run(ctx context.Context, pool *pgxpool.Pool, hub *ws.Hub) {
	gs := &store.GroupStore{DB: pool}

	// Step 1: Cancel groups with 3+ failed attempts
	cancelled, err := gs.CancelMaxRetries(ctx)
	if err != nil {
		slog.Error("assigner: cancel max retries failed", "error", err)
	} else if cancelled > 0 {
		slog.Warn("assigner: cancelled groups with max retries", "count", cancelled)
	}

	// Step 2: Revert timed-out dispatching groups
	// Use a fixed 5-min timeout for now; dynamic timeout per-group comes with Telegram
	reverted, err := gs.RevertTimedOut(ctx, 5*time.Minute)
	if err != nil {
		slog.Error("assigner: revert timed out failed", "error", err)
	} else if len(reverted) > 0 {
		slog.Info("assigner: reverted timed-out groups", "count", len(reverted), "ids", reverted)
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
		slog.Info("assigner: group dispatching",
			"group_id", group.ID,
			"score", group.ConfidenceScore,
			"attempt", group.DispatchAttempts+1,
			"timeout", timeout,
		)

		// Broadcast dispatching event
		if hub != nil {
			event := ws.GroupDispatching(group.ID, group.DispatchAttempts+1, group.ConfidenceScore)
			hub.BroadcastMulti([]string{"global", group.ID}, event)
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
	return time.Duration(timeoutMin) * time.Minute
}

// NewProcess creates a worker.ProcessFunc with the hub injected.
func NewProcess(hub *ws.Hub) worker.ProcessFunc {
	return func(ctx context.Context, pool *pgxpool.Pool, payload []byte) error {
		Run(ctx, pool, hub)
		worker.Notify(ctx, pool, "assigner_wakeup")
		return nil
	}
}
