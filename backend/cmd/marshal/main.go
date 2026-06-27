package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/api"
	"github.com/Ksalgotra1/Marshal/internal/assigner"
	"github.com/Ksalgotra1/Marshal/internal/config"
	"github.com/Ksalgotra1/Marshal/internal/grouper"
	"github.com/Ksalgotra1/Marshal/internal/handlers"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/sse"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/telegram"
	"github.com/Ksalgotra1/Marshal/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
)

func main() {
	cfg := config.Load()
	config.InitLogger(cfg.LogFormat)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := config.ConnectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := config.RunMigrations(ctx, pool, "migrations"); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations OK")

	// Realtime hub manages rooms and fans events out to transport adapters.
	realtimeHub := realtime.NewHub(cfg.AllowedOrigin)
	go realtimeHub.Run()

	var bot *telegram.Bot
	if cfg.TelegramBotToken != "" {
		gs := &store.GroupStore{DB: pool}
		ds := &store.DriverStore{DB: pool}
		cs := &store.ChatStore{DB: pool}
		bot = telegram.New(cfg.TelegramBotToken, mustParseInt64(cfg.TelegramDriverGroup), gs, ds, cs, realtimeHub)
	}

	mux := http.NewServeMux()

	if bot != nil {
		if cfg.TelegramWebhookURL != "" {
			if err := bot.RegisterWebhook(ctx, cfg.TelegramWebhookURL, cfg.TelegramWebhookSecret); err != nil {
				slog.Error("failed to register webhook", "error", err)
				os.Exit(1)
			}
			mux.Handle("POST /telegram/webhook", telegram.NewWebhookHandler(bot, cfg.TelegramWebhookSecret))
		} else {
			if err := bot.DeleteWebhook(ctx); err != nil {
				slog.Warn("deleteWebhook failed (may not have been set)", "error", err)
			}
			go bot.StartPolling(ctx)
		}
	}

	// Start background workers
	go worker.Run(ctx, worker.Config{
		Name:     "grouper",
		JobType:  "group_pending",
		Channel:  "grouper_wakeup",
		Interval: 30 * time.Second,
		Pool:     pool,
		Process:  grouperProcess(pool, realtimeHub),
	})

	ttl, _ := strconv.Atoi(cfg.DriverPresenceTTL)
	go worker.Run(ctx, worker.Config{
		Name:     "assigner",
		JobType:  "assign_group",
		Channel:  "assigner_wakeup",
		Interval: 30 * time.Second,
		Pool:     pool,
		Process:  assigner.NewProcess(realtimeHub, bot, ttl),
	})

	h := &handlers.Handlers{Pool: pool, ServerCtx: ctx, Events: realtimeHub, WebSocket: realtimeHub}
	if bot != nil {
		h.MessageSender = bot
	}

	limiter := api.NewIPRateLimiter(rate.Every(3*time.Second), 10)
	go limiter.Cleanup(ctx, 10*time.Minute, 5*time.Minute)

	// Health
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		api.WriteJSON(w, http.StatusOK, api.JSON{
			"status":      "ok",
			"connections": realtimeHub.ConnectionCount(),
		})
	})

	// WebSocket
	mux.HandleFunc("GET /ws", h.HandleWebSocket)
	mux.Handle("GET /events", &sse.Handler{Streams: realtimeHub})

	// Ride requests
	mux.Handle("GET /api/requests", jsonTimeout(http.HandlerFunc(h.HandleListRequests)))
	mux.Handle("POST /api/requests", jsonTimeout(limiter.Middleware(http.HandlerFunc(h.HandleCreateRequest))))
	mux.Handle("GET /api/requests/{id}", jsonTimeout(http.HandlerFunc(h.HandleGetRequest)))

	// Groups
	mux.Handle("GET /api/groups", jsonTimeout(http.HandlerFunc(h.HandleListGroups)))
	mux.Handle("GET /api/groups/open", jsonTimeout(http.HandlerFunc(h.HandleListOpenGroups)))
	mux.Handle("GET /api/groups/{id}", jsonTimeout(http.HandlerFunc(h.HandleGetGroup)))
	mux.Handle("POST /api/groups/{id}/join", jsonTimeout(limiter.Middleware(http.HandlerFunc(h.HandleJoinGroup))))
	mux.Handle("POST /api/groups/{id}/claim", jsonTimeout(limiter.Middleware(http.HandlerFunc(h.HandleClaimGroup))))
	mux.Handle("GET /api/groups/{id}/messages", jsonTimeout(http.HandlerFunc(h.HandleListMessages)))
	mux.Handle("POST /api/groups/{id}/messages", jsonTimeout(limiter.Middleware(http.HandlerFunc(h.HandleCreateMessage))))

	// Drivers
	mux.Handle("POST /api/drivers", jsonTimeout(limiter.Middleware(api.RequireAdminKey(cfg.AdminAPIKey)(http.HandlerFunc(h.HandleRegisterDriver)))))
	mux.Handle("GET /api/drivers", jsonTimeout(http.HandlerFunc(h.HandleListDrivers)))

	// Apply middleware stack
	stack := api.RequestIDMiddleware(api.CORSMiddleware(api.SecurityHeadersMiddleware(mux)))

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           stack,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
	}()

	go func() {
		time.Sleep(3 * time.Second)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		pingURL := "http://localhost:" + cfg.Port + "/healthz"
		client := &http.Client{Timeout: 5 * time.Second}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				resp, err := client.Get(pingURL)
				if err != nil {
					slog.Warn("keepalive ping failed", "error", err)
					continue
				}
				resp.Body.Close()
			}
		}
	}()

	slog.Info("marshal starting", "port", cfg.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func mustParseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		slog.Error("invalid int64 config value", "value", s)
		os.Exit(1)
	}
	return n
}

func jsonTimeout(next http.Handler) http.Handler {
	return http.TimeoutHandler(next, 10*time.Second, `{"error":"request timed out"}`)
}

// grouperProcess runs the H3 bucketing engine, then wakes the assigner.
func grouperProcess(pool *pgxpool.Pool, events grouper.EventPublisher) worker.ProcessFunc {
	return func(ctx context.Context, p *pgxpool.Pool, payload []byte) error {
		engine := &grouper.Engine{Pool: p, Events: events}
		engine.Run(ctx)
		// Enqueue an immediate check, and a delayed check for the 2-minute pooling window
		js := &store.JobStore{DB: pool}
		js.Enqueue(ctx, "assign_group", struct{}{}, models.PriorityNormal, time.Now())
		js.Enqueue(ctx, "assign_group", struct{}{}, models.PriorityNormal, time.Now().Add(2*time.Minute))
		worker.Notify(ctx, pool, "assigner_wakeup")
		return nil
	}
}
