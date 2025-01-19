package logger

import (
	"log"
	"strings"
)

type Logger interface {
	Infof(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

func NewLogger(level string) Logger {
	return &stdLogger{
		level: parseLevel(level),
	}
}

type stdLogger struct {
	level int
}

const (
	levelDebug = 0
	levelInfo  = 1
	levelError = 2
)

func parseLevel(l string) int {
	switch strings.ToLower(l) {
	case "debug":
		return levelDebug
	case "error":
		return levelError
	default:
		return levelInfo
	}
}

func (l *stdLogger) Infof(format string, v ...interface{}) {
	if l.level <= levelInfo {
		log.Printf("[INFO] "+format, v...)
	}
}

func (l *stdLogger) Errorf(format string, v ...interface{}) {
	if l.level <= levelError {
		log.Printf("[ERROR] "+format, v...)
	}
}

func (l *stdLogger) Fatalf(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}
