package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
)

// JSON is a convenience alias for response payloads.
type JSON map[string]any

// WriteJSON serialises data as JSON and sets the Content-Type header.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("api: failed to encode JSON response", "error", err, "status", status)
	}
}

// WriteError writes a standard error response.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, JSON{"error": msg})
}

// RequestID returns the request correlation id attached by RequestIDMiddleware.
func RequestID(r *http.Request) string {
	id, _ := r.Context().Value(RequestIDKey).(string)
	return id
}

// WriteRequestError sends a safe client error and logs internal details with a request id.
func WriteRequestError(w http.ResponseWriter, r *http.Request, status int, publicMsg string, internalErr error, attrs ...any) {
	requestID := RequestID(r)
	logAttrs := []any{
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"public_error", publicMsg,
	}
	if internalErr != nil {
		logAttrs = append(logAttrs, "error", internalErr)
	}
	logAttrs = append(logAttrs, attrs...)

	if status >= http.StatusInternalServerError {
		slog.Error("api request failed", logAttrs...)
	} else {
		slog.Warn("api request rejected", logAttrs...)
	}

	WriteJSON(w, status, JSON{
		"error":      publicMsg,
		"request_id": requestID,
	})
}

// CORSMiddleware sets permissive CORS headers for the configured origin.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("ALLOWED_ORIGIN")
		if origin == "" {
			origin = "http://localhost:5173"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware injects a unique X-Request-ID into every request.
type ctxKey string

const RequestIDKey ctxKey = "request_id"

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
