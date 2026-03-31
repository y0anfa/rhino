package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func init() {
	Logger = buildLogger()
}

func buildLogger() *zap.Logger {
	level := zapcore.InfoLevel
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToLower(envLevel) {
		case "debug":
			level = zapcore.DebugLevel
		case "info":
			level = zapcore.InfoLevel
		case "warn":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		}
	}

	format := "json"
	if envFormat := os.Getenv("LOG_FORMAT"); envFormat != "" {
		format = strings.ToLower(envFormat)
	}

	var cfg zap.Config
	if format == "text" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	cfg.Level.SetLevel(level)

	l, err := cfg.Build()
	if err != nil {
		l, _ = zap.NewProduction()
	}
	return l
}

func Sync() {
	Logger.Sync()
}

func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}
