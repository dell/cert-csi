package utils

import (
	"context"
	"github.com/sirupsen/logrus"
)

const LoggerContextKey = "logger"

func GetLoggerFromContext(ctx context.Context) *logrus.Entry {
	log, ok := ctx.Value(LoggerContextKey).(*logrus.Entry)
	if !ok {
		return logrus.NewEntry(logrus.StandardLogger()) // Just return standard logger to not panic
	}
	return log
}
