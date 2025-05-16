package utils

import (
	"runtime/debug"

	"go.uber.org/zap"
)

func PanicRecovery(log *zap.Logger) {
	if r := recover(); r != nil {
		log.With(zap.String("stack", string(debug.Stack()))).Error("recovered panic")
	}
}
