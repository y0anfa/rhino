package logger

import "go.uber.org/zap"

var (
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
)

func init() {
	Logger, _ = zap.NewProduction()
	defer Logger.Sync()
	Sugar = Logger.Sugar()
}

func Debug(fields ...interface{}) {
	Sugar.Debug(fields...)
}

func Info(fields ...interface{}) {
	Sugar.Info(fields...)
}

func Warn(fields ...interface{}) {
	Sugar.Warn(fields...)
}

func Error(fields ...interface{}) {
	Sugar.Error(fields...)
}

func Fatal(fields ...interface{}) {
	Sugar.Fatal(fields...)
}
