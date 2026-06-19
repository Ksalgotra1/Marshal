package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/api"
	"github.com/Ksalgotra1/Marshal/internal/config"
	"github.com/Ksalgotra1/Marshal/internal/handlers"
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

	h := &handlers.Handlers{Pool: pool, ServerCtx: ctx}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		api.WriteJSON(w, http.StatusOK, api.JSON{"status": "ok"})
	})

	// Ride requests
	mux.HandleFunc("POST /api/requests", h.HandleCreateRequest)
	mux.HandleFunc("GET /api/requests/{id}", h.HandleGetRequest)

	// Groups
	mux.HandleFunc("GET /api/groups", h.HandleListGroups)
	mux.HandleFunc("GET /api/groups/open", h.HandleListOpenGroups)
	mux.HandleFunc("POST /api/groups/{id}/join", h.HandleJoinGroup)

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
