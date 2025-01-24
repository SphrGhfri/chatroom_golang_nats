package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

var loggerKey = ctxKey{}
var logFile string // Add this line to store logFile name globally

type Logger interface {
	WithContext(ctx context.Context) Logger
	WithModule(name string) Logger
	WithFields(fields map[string]interface{}) Logger
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

type zapLogger struct {
	*zap.SugaredLogger
	module  string
	traceID string
}

// NewLogger initializes a Zap logger with enhanced features
func NewLogger(level, logFileName string) Logger {
	logFile = logFileName // Store the logFile name globally
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, os.ModePerm); err != nil {
		fmt.Printf("Could not create logs directory: %v\n", err)
		os.Exit(1)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			parseLevel(level),
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(createLogFile(logsDir, logFile)),
			parseLevel(level),
		),
	)

	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	).Sugar()

	return &zapLogger{
		SugaredLogger: logger,
		module:        "main",
		traceID:       generateTraceID(),
	}
}

func (l *zapLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	// Try to get existing trace ID from context
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return &zapLogger{
			SugaredLogger: l.SugaredLogger.With(
				"trace_id", traceID,
				"module", l.module,
			),
			module:  l.module,
			traceID: traceID,
		}
	}

	// Generate new trace ID if none exists
	traceID := generateTraceID()
	return &zapLogger{
		SugaredLogger: l.SugaredLogger.With(
			"trace_id", traceID,
			"module", l.module,
		),
		module:  l.module,
		traceID: traceID,
	}
}

func (l *zapLogger) WithModule(name string) Logger {
	return &zapLogger{
		SugaredLogger: l.SugaredLogger.With("module", name),
		module:        name,
		traceID:       l.traceID,
	}
}

func (l *zapLogger) WithFields(fields map[string]interface{}) Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &zapLogger{
		SugaredLogger: l.SugaredLogger.With(args...),
		module:        l.module,
		traceID:       l.traceID,
	}
}

// Helper functions
func generateTraceID() string {
	return uuid.New().String()
}

func createLogFile(logsDir string, logsFile string) *os.File {
	logFile, err := os.OpenFile(
		filepath.Join(logsDir, logsFile),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		fmt.Printf("Could not open log file: %v\n", err)
		os.Exit(1)
	}
	return logFile
}

// Context helpers
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerKey).(Logger); ok {
		return l
	}
	return NewLogger("info", logFile)
}

func NewContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "error":
		return zapcore.ErrorLevel
	case "warn":
		return zapcore.WarnLevel
	default:
		return zapcore.InfoLevel
	}
}

// Logger methods for different levels
func (l *zapLogger) Infof(format string, v ...interface{})  { l.SugaredLogger.Infof(format, v...) }
func (l *zapLogger) Warnf(format string, v ...interface{})  { l.SugaredLogger.Warnf(format, v...) }
func (l *zapLogger) Errorf(format string, v ...interface{}) { l.SugaredLogger.Errorf(format, v...) }
func (l *zapLogger) Fatalf(format string, v ...interface{}) { l.SugaredLogger.Fatalf(format, v...) }
