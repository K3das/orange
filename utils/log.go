package utils

import (
	"context"

	"go.uber.org/zap"
)

type logKeyType struct{}

func LogContext(ctx context.Context, fields ...zap.Field) context.Context {
	old := GetLogContextFields(ctx)
	fields = append(old, fields...)
	return context.WithValue(ctx, logKeyType{}, fields)
}

func GetLogContextFields(ctx context.Context) []zap.Field {
	fields, ok := ctx.Value(logKeyType{}).([]zap.Field)
	if !ok {
		return nil
	}
	return fields
}

func GetLogFromContext(ctx context.Context, parentLog *zap.Logger) *zap.Logger {
	return parentLog.With(GetLogContextFields(ctx)...)
}

func LogContextWith(ctx context.Context, parentLog *zap.Logger, fields ...zap.Field) (context.Context, *zap.Logger) {
	ctx = LogContext(ctx, fields...)
	parentLog = parentLog.With(fields...)
	return ctx, parentLog
}
