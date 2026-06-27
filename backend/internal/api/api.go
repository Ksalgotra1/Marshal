package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"crypto/subtle"

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
        allowed := os.Getenv("ALLOWED_ORIGIN")
        if allowed == "" {
            allowed = "http://localhost:5173"
        }
        requestOrigin := r.Header.Get("Origin")
        matched := ""
        for _, o := range strings.Split(allowed, ",") {
            if strings.TrimSpace(o) == requestOrigin {
                matched = requestOrigin
                break
            }
        }
        if matched == "" && allowed == "*" {
            matched = "*"
        }
        if matched != "" {
            w.Header().Set("Access-Control-Allow-Origin", matched)
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        }
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

// RequireAdminKey gates a handler behind a shared secret passed in the
// X-Admin-Key header. Fails closed: if adminKey is unset, the route is
// disabled rather than left open.
func RequireAdminKey(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if adminKey == "" {
				WriteRequestError(w, r, http.StatusServiceUnavailable,
					"this endpoint is not configured", nil)
				return
			}
			provided := r.Header.Get("X-Admin-Key")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(adminKey)) != 1 {
				WriteRequestError(w, r, http.StatusUnauthorized,
					"invalid or missing admin key", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
