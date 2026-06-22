//go:build integration

package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/testdb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type WorkerIntegrationSuite struct {
	suite.Suite
	db  *testdb.Instance
	ctx context.Context
}

func TestWorkerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(WorkerIntegrationSuite))
}

func (s *WorkerIntegrationSuite) SetupSuite() {
	s.ctx = context.Background()
	s.db = testdb.Start(s.T())
}

func (s *WorkerIntegrationSuite) SetupTest() {
	testdb.Truncate(s.ctx, s.T(), s.db.Pool)
}

func (s *WorkerIntegrationSuite) TestWorker_DrainSuccess() {
	js := &store.JobStore{DB: s.db.Pool}
	
	_, err := js.Enqueue(s.ctx, "test_job", map[string]string{"key": "value"}, time.Now().Add(-24*time.Hour))
	s.Require().NoError(err)

	called := false
	mockProcess := func(ctx context.Context, pool *pgxpool.Pool, payload []byte) error {
		called = true
		s.Contains(string(payload), "value")
		return nil
	}

	cfg := Config{
		Name:    "test_worker",
		JobType: "test_job",
		Pool:    s.db.Pool,
		Process: mockProcess,
	}

	// Drain runs until no jobs remain
	drain(s.ctx, cfg)

	s.True(called, "ProcessFunc should have been called")

	// Verify job is marked 'done'
	var status string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT status FROM jobs WHERE job_type = 'test_job' LIMIT 1`).Scan(&status))
	s.Equal("done", status)
}

func (s *WorkerIntegrationSuite) TestWorker_DrainError() {
	js := &store.JobStore{DB: s.db.Pool}
	
	_, err := js.Enqueue(s.ctx, "test_job_fail", map[string]string{"fail": "true"}, time.Now().Add(-24*time.Hour))
	s.Require().NoError(err)

	mockProcess := func(ctx context.Context, pool *pgxpool.Pool, payload []byte) error {
		return errors.New("simulated error")
	}

	cfg := Config{
		Name:    "test_worker_fail",
		JobType: "test_job_fail",
		Pool:    s.db.Pool,
		Process: mockProcess,
	}

	drain(s.ctx, cfg)

	// Verify job is marked 'queued' (retry)
	var status string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT status FROM jobs WHERE job_type = 'test_job_fail' LIMIT 1`).Scan(&status))
	s.Equal("queued", status)
}

func (s *WorkerIntegrationSuite) TestWorker_NotifyPropagation() {
	notifyChan := make(chan struct{}, 1)
	
	// Start listener in background
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	
	go listen(ctx, s.db.Pool, "test_channel", notifyChan)

	// Give it a split second to execute LISTEN
	time.Sleep(100 * time.Millisecond)

	// Trigger Notify
	Notify(s.ctx, s.db.Pool, "test_channel")

	select {
	case <-notifyChan:
		// Success
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for LISTEN/NOTIFY propagation")
	}
}
