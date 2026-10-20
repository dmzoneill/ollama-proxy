package virtual

import (
	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

func warnErr(msg string, err error) {
	if err == nil {
		return
	}
	logging.Warn(msg, zap.Error(err))
}
