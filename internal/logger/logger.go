package logger

import (
	"fmt"
	"time"
)

// Color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
)

type LogLevel string

var (
	GlobalLogLevel LogLevel = "INFO"
)

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type Log struct {
	level LogLevel
	err   error
}

func New() *Log {
	return &Log{
		level: GlobalLogLevel,
	}
}

func (l *Log) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Log) WithError(err error) *Log {
	return &Log{err: err}
}

func (l *Log) timestamp() string {
	return time.Now().Format("15:04:05")
}

func (l *Log) Debug(msg string) {
	if l.level > LogLevelDebug {
		return
	}
	if l.err != nil {
		fmt.Printf("%s[%s]%s ℹ️  %s: %v%s\n", ColorCyan, l.timestamp(), ColorReset, msg, l.err, ColorReset)
		return
	}
	fmt.Printf("%s[%s]%s ℹ️  %s%s\n", ColorBlue, l.timestamp(), ColorReset, msg, ColorReset)
}

func (l *Log) Info(msg string) {
	if l.level > LogLevelInfo {
		return
	}

	fmt.Printf("%s[%s]%s ℹ️  %s%s\n", ColorBlue, l.timestamp(), ColorReset, msg, ColorReset)
}

func (l *Log) Character(character, msg string) {
	if l.level > LogLevelInfo {
		return
	}

	fmt.Printf("%s[%s]%s [%s]ℹ%s  %s", ColorBlue, l.timestamp(), ColorBold, character, msg, ColorReset)
}

func (l *Log) Warn(msg string) {
	if l.level > LogLevelWarn {
		return
	}

	if l.err != nil {
		fmt.Printf("%s[%s]%s ⚠️  %s: %v%s\n", ColorYellow, l.timestamp(), ColorReset, msg, l.err, ColorReset)
		return
	}
	fmt.Printf("%s[%s]%s ⚠️  %s%s\n", ColorYellow, l.timestamp(), ColorReset, msg, ColorReset)
}

func (l *Log) Error(msg string) {
	if l.err != nil {
		fmt.Printf("%s[%s]%s ❌ %s: %v%s\n", ColorRed, l.timestamp(), ColorReset, msg, l.err, ColorReset)
		return
	}
	fmt.Printf("%s[%s]%s ❌ %s%s\n", ColorRed, l.timestamp(), ColorReset, msg, ColorReset)
}
