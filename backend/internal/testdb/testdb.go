package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// truncateMu serialises TRUNCATE calls across parallel test packages
// that share the same CI database to avoid deadlocks.
var truncateMu sync.Mutex

const (
	dbUser     = "marshal_test"
	dbPassword = "marshal_test_secret"
	dbName     = "marshal_test"
)

type Instance struct {
	Pool        *pgxpool.Pool
	DatabaseURL string
	containerID string
}

func Start(t testing.TB) *Instance {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)

	dsn := os.Getenv("MARSHAL_TEST_DATABASE_URL")
	if dsn != "" {
		pool := connect(ctx, t, dsn)
		applySchema(ctx, t, pool)
		t.Cleanup(pool.Close)
		return &Instance{Pool: pool, DatabaseURL: dsn}
	}

	containerID := dockerRun(ctx, t)
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = exec.CommandContext(stopCtx, "docker", "rm", "-f", containerID).Run()
	})

	port := dockerPort(ctx, t, containerID)
	dsn = fmt.Sprintf("postgres://%s:%s@127.0.0.1:%s/%s?sslmode=disable", dbUser, dbPassword, port, dbName)
	pool := connect(ctx, t, dsn)
	applySchema(ctx, t, pool)
	t.Cleanup(pool.Close)

	return &Instance{Pool: pool, DatabaseURL: dsn, containerID: containerID}
}

func Truncate(ctx context.Context, t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	truncateMu.Lock()
	defer truncateMu.Unlock()
	_, err := pool.Exec(ctx, `
		TRUNCATE chat_messages, group_members, ride_groups, ride_requests, drivers, jobs
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate test db: %v", err)
	}
}

func dockerRun(ctx context.Context, t testing.TB) string {
	t.Helper()
	name := fmt.Sprintf("marshal_test_%d", time.Now().UnixNano())
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-d",
		"--name", name,
		"-p", "127.0.0.1::5432",
		"-e", "POSTGRES_USER="+dbUser,
		"-e", "POSTGRES_PASSWORD="+dbPassword,
		"-e", "POSTGRES_DB="+dbName,
		"postgres:16",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "permission denied") || strings.Contains(msg, "Cannot connect to the Docker daemon") {
			t.Skipf("Docker is unavailable for DB integration tests: %v: %s", err, msg)
		}
		t.Fatalf("start postgres test container: %v: %s", err, msg)
	}
	return strings.TrimSpace(string(out))
}

func dockerPort(ctx context.Context, t testing.TB, containerID string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "docker", "port", containerID, "5432/tcp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect postgres test container port: %v: %s", err, strings.TrimSpace(string(out)))
	}

	mapping := strings.TrimSpace(string(out))
	if strings.Contains(mapping, "\n") {
		mapping = strings.Split(mapping, "\n")[0]
	}
	u, err := url.Parse("tcp://" + mapping)
	if err == nil && u.Port() != "" {
		return u.Port()
	}
	parts := strings.Split(mapping, ":")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		t.Fatalf("unexpected docker port mapping: %q", mapping)
	}
	return parts[len(parts)-1]
}

func connect(ctx context.Context, t testing.TB, dsn string) *pgxpool.Pool {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		pool, err := pgxpool.New(ctx, dsn)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return pool
			} else {
				lastErr = pingErr
			}
			pool.Close()
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("connect to postgres test db: %v", lastErr)
	return nil
}

func applySchema(ctx context.Context, t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("create pgcrypto extension: %v", err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate testdb helper")
	}
	migrationsDir := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	
	var upFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	for _, f := range upFiles {
		migration, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			t.Fatalf("read migration %s: %v", f, err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "duplicate column name") {
				t.Fatalf("apply migration %s: %v", f, err)
			}
		}
	}
	Truncate(ctx, t, pool)
}
