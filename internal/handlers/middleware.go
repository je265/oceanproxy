package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/je265/oceanproxy/internal/pkg/errors"
)

// AuthMiddleware provides bearer token authentication
func NewAuthMiddleware(bearerToken string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health checks and public endpoints
			if isPublicEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn("Missing Authorization header",
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr))

				respondWithError(w, http.StatusUnauthorized, "Authorization header required", nil)
				return
			}

			// Check Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				logger.Warn("Invalid Authorization header format",
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr))

				respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header format", nil)
				return
			}

			token := parts[1]
			if token != bearerToken {
				logger.Warn("Invalid bearer token",
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr))

				respondWithError(w, http.StatusUnauthorized, "Invalid bearer token", nil)
				return
			}

			// Add user context (for future use)
			ctx := context.WithValue(r.Context(), "authenticated", true)
			ctx = context.WithValue(ctx, "auth_method", "bearer")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimitMiddleware provides basic rate limiting
func NewRateLimitMiddleware(requestsPerMinute int, logger *zap.Logger) func(http.Handler) http.Handler {
	// Simple in-memory rate limiter (for production, use Redis or similar)
	type clientData struct {
		requests  int
		resetTime time.Time
	}

	clients := make(map[string]*clientData)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			now := time.Now()

			// Clean up old entries periodically
			if len(clients) > 1000 {
				for ip, data := range clients {
					if now.After(data.resetTime) {
						delete(clients, ip)
					}
				}
			}

			client, exists := clients[clientIP]
			if !exists || now.After(client.resetTime) {
				clients[clientIP] = &clientData{
					requests:  1,
					resetTime: now.Add(time.Minute),
				}
				next.ServeHTTP(w, r)
				return
			}

			if client.requests >= requestsPerMinute {
				logger.Warn("Rate limit exceeded",
					zap.String("client_ip", clientIP),
					zap.Int("requests", client.requests),
					zap.String("path", r.URL.Path))

				w.Header().Set("X-RateLimit-Limit", string(rune(requestsPerMinute)))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", string(rune(client.resetTime.Unix())))

				respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded", nil)
				return
			}

			client.requests++
			w.Header().Set("X-RateLimit-Limit", string(rune(requestsPerMinute)))
			w.Header().Set("X-RateLimit-Remaining", string(rune(requestsPerMinute-client.requests)))
			w.Header().Set("X-RateLimit-Reset", string(rune(client.resetTime.Unix())))

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware provides request logging
func NewLoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", getClientIP(r)),
				zap.Int("status_code", wrapped.statusCode),
				zap.Duration("duration", duration),
				zap.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// API versioning header
		w.Header().Set("X-API-Version", "v1")

		next.ServeHTTP(w, r)
	})
}

// ContentTypeMiddleware ensures JSON content type for API responses
func ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set default content type for API endpoints
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics and returns 500 error
func NewRecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("remote_addr", getClientIP(r)),
						zap.Any("error", err))

					respondWithError(w, http.StatusInternalServerError, "Internal server error", nil)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Helper types and functions

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}

	return ip
}

func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/ready",
		"/ping",
		"/metrics",
		"/docs",
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}

	return false
}

func respondWithError(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := errors.NewErrorResponse(message, err)

	// Don't log JSON encoding errors to avoid infinite loops
	json.NewEncoder(w).Encode(errorResponse)
}
