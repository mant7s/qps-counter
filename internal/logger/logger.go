package logger

import (
	"fmt"
	"github.com/mant7s/qps-counter/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

var (
	globalLogger *zap.Logger
	atomicLevel  zap.AtomicLevel
)

func Init(cfg config.LoggerConfig) {
	atomicLevel = zap.NewAtomicLevel()

	switch cfg.Level {
	case "debug":
		atomicLevel.SetLevel(zap.DebugLevel)
	case "info":
		atomicLevel.SetLevel(zap.InfoLevel)
	case "warn":
		atomicLevel.SetLevel(zap.WarnLevel)
	case "error":
		atomicLevel.SetLevel(zap.ErrorLevel)
	default:
		atomicLevel.SetLevel(zap.InfoLevel)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "timestamp"

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var cores []zapcore.Core

	if cfg.FilePath != "" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   true,
		})
		fileCore := zapcore.NewCore(encoder, fileWriter, atomicLevel)
		cores = append(cores, fileCore)
	}

	consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), atomicLevel)
	cores = append(cores, consoleCore)

	globalLogger = zap.New(zapcore.NewTee(cores...), zap.AddCaller())

	zap.RedirectStdLog(globalLogger)
}

func Sync() error {
	return globalLogger.Sync()
}

func GetLogger() *zap.Logger {
	return globalLogger
}

func Debug(msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	globalLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	globalLogger.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	globalLogger.Fatal(msg, fields...)
}

func ErrorWrap(err error, msg string, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	globalLogger.Error(fmt.Sprintf("%s: %v", msg, err), fields...)
}
