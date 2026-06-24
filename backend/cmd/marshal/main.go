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

	// Realtime hub manages rooms and fans events out to transport adapters.
	realtimeHub := realtime.NewHub()
	go realtimeHub.Run()

	var bot *telegram.Bot
	if cfg.TelegramBotToken != "" {
		gs := &store.GroupStore{DB: pool}
		ds := &store.DriverStore{DB: pool}
		bot = telegram.New(cfg.TelegramBotToken, mustParseInt64(cfg.TelegramDriverGroup), gs, ds)
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

	go worker.Run(ctx, worker.Config{
		Name:     "assigner",
		JobType:  "assign_group",
		Channel:  "assigner_wakeup",
		Interval: 30 * time.Second,
		Pool:     pool,
		Process:  assigner.NewProcess(realtimeHub, bot),
	})

	h := &handlers.Handlers{Pool: pool, ServerCtx: ctx, Events: realtimeHub, WebSocket: realtimeHub}

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
	mux.HandleFunc("GET /api/requests", h.HandleListRequests)
	mux.HandleFunc("POST /api/requests", h.HandleCreateRequest)
	mux.HandleFunc("GET /api/requests/{id}", h.HandleGetRequest)

	// Groups
	mux.HandleFunc("GET /api/groups", h.HandleListGroups)
	mux.HandleFunc("GET /api/groups/open", h.HandleListOpenGroups)
	mux.HandleFunc("GET /api/groups/{id}", h.HandleGetGroup)
	mux.HandleFunc("POST /api/groups/{id}/join", h.HandleJoinGroup)
	mux.HandleFunc("POST /api/groups/{id}/claim", h.HandleClaimGroup)

	// Drivers
	mux.HandleFunc("POST /api/drivers", h.HandleRegisterDriver)
	mux.HandleFunc("GET /api/drivers", h.HandleListDrivers)

	// Apply middleware stack
	stack := api.RequestIDMiddleware(api.CORSMiddleware(mux))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      stack,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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

// grouperProcess runs the H3 bucketing engine, then wakes the assigner.
func grouperProcess(pool *pgxpool.Pool, events grouper.EventPublisher) worker.ProcessFunc {
	return func(ctx context.Context, p *pgxpool.Pool, payload []byte) error {
		engine := &grouper.Engine{Pool: p, Events: events}
		engine.Run(ctx)
		// Enqueue an assign_group job and wake assigner via NOTIFY
		js := &store.JobStore{DB: pool}
		js.Enqueue(ctx, "assign_group", struct{}{}, models.PriorityNormal, time.Now())
		worker.Notify(ctx, pool, "assigner_wakeup")
		return nil
	}
}
