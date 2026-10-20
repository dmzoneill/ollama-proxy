// Package middleware provides HTTP middleware used by the proxy.
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
			rec := recover()
			if rec == nil {
				return
			}
			requestID := GetRequestID(r.Context())
			logging.Error("HTTP panic recovered",
				zap.Any("panic", rec),
				zap.String("stack", string(debug.Stack())),
				zap.String("request_id", requestID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)
			w.WriteHeader(http.StatusInternalServerError)
			writeServerError(w)
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
				writeServerError(w)
			}
		}()

		handler(w, r)
	}
}

func writeServerError(w http.ResponseWriter) {
	if _, err := w.Write([]byte("Internal Server Error")); err != nil {
		logging.Warn("failed to write panic response", zap.Error(err))
	}
}
