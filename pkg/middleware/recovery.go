package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/daoneill/ollama-proxy/pkg/logging"
	"go.uber.org/zap"
)

// RecoverPanic recovers from panics in handler functions
func RecoverPanic(ctx context.Context, handler func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			requestID := GetRequestID(ctx)
			logging.Error("Panic recovered",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())),
				zap.String("request_id", requestID),
			)
			err = fmt.Errorf("internal server error: panic recovered")
		}
	}()

	return handler()
}
