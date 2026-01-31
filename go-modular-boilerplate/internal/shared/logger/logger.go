package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zap.Logger to provide structured logging
type Logger struct {
	*zap.Logger
}

// New creates a new logger instance based on the environment
func New(environment string) *Logger {
	var config zap.Config

	if environment == "production" {
		// Production config (structured JSON logs)
		config = zap.NewProductionConfig()
	} else {
		// Development config (human-readable colored logs)
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Create log directory if it doesn't exist
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		os.Stderr.WriteString("Failed to create log directory: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Create a JSON encoder config for files (without colors)
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // No colors for files

	// Create a console encoder config for development (with colors)
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// Configure lumberjack for log rotation
	infoWriter := &lumberjack.Logger{
		Filename:   logDir + "/info.log",
		MaxSize:    100, // megabytes
		MaxBackups: 30,  // number of backups
		MaxAge:     30,  // days
		Compress:   true,
	}

	errorWriter := &lumberjack.Logger{
		Filename:   logDir + "/error.log",
		MaxSize:    100, // megabytes
		MaxBackups: 30,  // number of backups
		MaxAge:     30,  // days
		Compress:   true,
	}

	warnWriter := &lumberjack.Logger{
		Filename:   logDir + "/warn.log",
		MaxSize:    100, // megabytes
		MaxBackups: 30,  // number of backups
		MaxAge:     30,  // days
		Compress:   true,
	}

	debugWriter := &lumberjack.Logger{
		Filename:   logDir + "/debug.log",
		MaxSize:    100, // megabytes
		MaxBackups: 30,  // number of backups
		MaxAge:     30,  // days
		Compress:   true,
	}

	// Create a custom core that routes logs to different files based on level
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(fileEncoderConfig),
			zapcore.AddSync(infoWriter),
			zapcore.InfoLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(fileEncoderConfig),
			zapcore.AddSync(errorWriter),
			zapcore.ErrorLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(fileEncoderConfig),
			zapcore.AddSync(warnWriter),
			zapcore.WarnLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(fileEncoderConfig),
			zapcore.AddSync(debugWriter),
			zapcore.DebugLevel,
		),
	)

	// Also keep stdout for development
	if environment != "production" {
		core = zapcore.NewTee(
			core,
			zapcore.NewCore(
				zapcore.NewConsoleEncoder(consoleEncoderConfig),
				zapcore.AddSync(os.Stdout),
				zapcore.DebugLevel,
			),
		)
	}

	// Build the logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		Logger: logger,
	}
}

// Named returns a named logger
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		Logger: l.Logger.Named(name),
	}
}

// With creates a child logger with the given fields
func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
	}
}

// Sugar returns a sugared logger
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.Logger.Sugar()
}
