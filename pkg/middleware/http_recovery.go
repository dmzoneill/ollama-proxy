package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"go.uber.org/zap"
)

// HTTPRecovery is HTTP middleware that recovers from panics
func HTTPRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				requestID := ""
				if ctx := r.(*http.Request).Context(); ctx != nil {
					requestID = GetRequestID(ctx)
				}

				if logging.Logger != nil {
					logging.Logger.Error("HTTP panic recovered",
						zap.Any("panic", r),
						zap.String("stack", string(debug.Stack())),
						zap.String("request_id", requestID),
						zap.String("method", r.(*http.Request).Method),
						zap.String("path", r.(*http.Request).URL.Path),
					)
				}

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RecoveryHandlerFunc wraps an http.HandlerFunc with panic recovery
func RecoveryHandlerFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := ""
				if ctx := r.Context(); ctx != nil {
					requestID = GetRequestID(ctx)
				}

				if logging.Logger != nil {
					logging.Logger.Error("HTTP panic recovered",
						zap.Any("panic", rec),
						zap.String("stack", string(debug.Stack())),
						zap.String("request_id", requestID),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
					)
				}

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			}
		}()

		handler(w, r)
	}
}
