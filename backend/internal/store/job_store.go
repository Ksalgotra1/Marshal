package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

// JobStore manages the Postgres-backed job queue.
type JobStore struct{ DB DBTX }

// Enqueue inserts a new job into the queue.
// runAfter controls when the job becomes visible to workers (use time.Now() for immediate).
func (s *JobStore) Enqueue(ctx context.Context, jobType string, payload any, runAfter time.Time) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("JobStore.Enqueue marshal: %w", err)
	}

	var id string
	err = s.DB.QueryRow(ctx, `
		INSERT INTO jobs (job_type, payload, run_after)
		VALUES ($1, $2, $3)
		RETURNING id
	`, jobType, data, runAfter).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("JobStore.Enqueue: %w", err)
	}
	return id, nil
}

// Dequeue atomically grabs the next available job using FOR UPDATE SKIP LOCKED.
// Returns nil if no jobs are ready. The caller MUST call MarkDone or MarkFailed.
func (s *JobStore) Dequeue(ctx context.Context, jobType string) (*models.Job, error) {
	var j models.Job
	err := s.DB.QueryRow(ctx, `
		UPDATE jobs
		SET status = 'processing', attempts = attempts + 1, updated_at = NOW()
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = 'queued' AND job_type = $1 AND run_after <= NOW()
			ORDER BY run_after ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, job_type, payload, status, attempts, max_attempts, run_after, created_at, updated_at
	`, jobType).Scan(
		&j.ID, &j.JobType, &j.Payload, &j.Status, &j.Attempts,
		&j.MaxAttempts, &j.RunAfter, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // no jobs available
		}
		return nil, fmt.Errorf("JobStore.Dequeue: %w", err)
	}
	return &j, nil
}

// MarkDone marks a job as successfully completed.
func (s *JobStore) MarkDone(ctx context.Context, jobID string) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE jobs SET status = 'done', updated_at = NOW() WHERE id = $1`, jobID,
	)
	return err
}

// MarkFailed marks a job as failed. If attempts < max_attempts, re-queues it with a backoff.
func (s *JobStore) MarkFailed(ctx context.Context, jobID string) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE jobs SET
			status = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'queued' END,
			run_after = CASE WHEN attempts >= max_attempts THEN run_after ELSE NOW() + (attempts * interval '10 seconds') END,
			updated_at = NOW()
		WHERE id = $1
	`, jobID)
	return err
}
