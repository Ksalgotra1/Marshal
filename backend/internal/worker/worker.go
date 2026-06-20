package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProcessFunc is called when a job is dequeued. Return an error to mark the job as failed.
type ProcessFunc func(ctx context.Context, pool *pgxpool.Pool, payload []byte) error

// Config holds worker configuration.
type Config struct {
	Name     string        // e.g. "grouper", "assigner"
	JobType  string        // e.g. "group_pending", "assign_group"
	Interval time.Duration // safety-net poll interval (fallback if NOTIFY is missed)
	Pool     *pgxpool.Pool
	Process  ProcessFunc
	Channel  string // Postgres LISTEN channel name (e.g. "grouper_wakeup")
}

// Run starts a blocking worker loop that listens for Postgres NOTIFY events.
// Primary wakeup: LISTEN/NOTIFY (instant, zero polling overhead).
// Fallback wakeup: ticker at Config.Interval (safety net if a notification is missed).
// Exits cleanly when ctx is cancelled (server shutdown).
func Run(ctx context.Context, cfg Config) {
	slog.Info("worker started", "name", cfg.Name, "channel", cfg.Channel, "fallback_interval", cfg.Interval)

	// Spawn a dedicated listener goroutine that feeds into a notification channel
	notify := make(chan struct{}, 1)
	go listen(ctx, cfg.Pool, cfg.Channel, notify)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopping", "name", cfg.Name)
			return
		case <-notify:
			drain(ctx, cfg)
		case <-ticker.C:
			drain(ctx, cfg)
		}
	}
}

// listen acquires a dedicated Postgres connection and blocks on LISTEN.
// Each notification pushes a non-blocking signal into the channel.
// If the connection drops, it reconnects with backoff.
func listen(ctx context.Context, pool *pgxpool.Pool, channel string, notify chan struct{}) {
	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := pool.Acquire(ctx)
		if err != nil {
			slog.Error("worker listener: failed to acquire connection", "channel", channel, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		_, err = conn.Exec(ctx, "LISTEN "+channel)
		if err != nil {
			slog.Error("worker listener: LISTEN failed", "channel", channel, "error", err)
			conn.Release()
			continue
		}

		slog.Info("worker listener: subscribed", "channel", channel)

		// Block until a notification arrives or context is cancelled
		for {
			_, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					conn.Release()
					return
				}
				slog.Warn("worker listener: connection lost, reconnecting", "channel", channel, "error", err)
				conn.Release()
				break // break inner loop to reconnect
			}

			// Non-blocking push — if worker is already awake, signal is absorbed
			select {
			case notify <- struct{}{}:
			default:
			}
		}
	}
}

// drain processes all available jobs of this type until none remain.
func drain(ctx context.Context, cfg Config) {
	js := &store.JobStore{DB: cfg.Pool}
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := js.Dequeue(ctx, cfg.JobType)
		if err != nil {
			slog.Error("worker dequeue error", "name", cfg.Name, "error", err)
			return
		}
		if job == nil {
			return // no more jobs
		}

		slog.Info("worker processing job", "name", cfg.Name, "job_id", job.ID)
		if err := cfg.Process(ctx, cfg.Pool, job.Payload); err != nil {
			slog.Error("worker job failed", "name", cfg.Name, "job_id", job.ID, "error", err)
			js.MarkFailed(ctx, job.ID)
		} else {
			js.MarkDone(ctx, job.ID)
			slog.Info("worker job done", "name", cfg.Name, "job_id", job.ID)
		}
	}
}

// Notify sends a Postgres NOTIFY on the given channel.
// Call this from HTTP handlers after enqueuing a job.
func Notify(ctx context.Context, pool *pgxpool.Pool, channel string) {
	_, err := pool.Exec(ctx, "SELECT pg_notify($1, '')", channel)
	if err != nil {
		slog.Error("pg_notify failed", "channel", channel, "error", err)
	}
}
