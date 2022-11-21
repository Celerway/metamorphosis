// Package log provides logging services. All logging goes through this layer so that we can
// easily change the logging implementation.
package log

import (
	"fmt"
	"io"
	golog "log"
	"os"
)

type LogLevel int

const (
	TraceLevel LogLevel = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (level LogLevel) String() string {
	if level < TraceLevel || level > FatalLevel {
		return "unknown"
	}
	return [...]string{"trace", "debug", "info", "warn", "error", "fatal"}[level]
}

type Logger struct {
	logLevel    LogLevel
	traceLogger golog.Logger
	debugLogger golog.Logger
	infoLogger  golog.Logger
	warnLogger  golog.Logger
	errorLogger golog.Logger
	fatalLogger golog.Logger
}

func NewLogger(infoOutput, errorOutput io.Writer) *Logger {
	l := &Logger{
		traceLogger: *golog.New(infoOutput, "trace", golog.LstdFlags),
		debugLogger: *golog.New(infoOutput, "debug", golog.LstdFlags),
		infoLogger:  *golog.New(infoOutput, "info", golog.LstdFlags),
		warnLogger:  *golog.New(errorOutput, "warn", golog.LstdFlags),
		errorLogger: *golog.New(errorOutput, "error", golog.LstdFlags),
		fatalLogger: *golog.New(errorOutput, "fatal", golog.LstdFlags),
	}
	return l
}
func NewWithPrefix(infoOutput, errorOutput io.Writer, prefix string) *Logger {
	l := &Logger{
		traceLogger: *golog.New(infoOutput, prefix+" trace", golog.LstdFlags),
		debugLogger: *golog.New(infoOutput, prefix+" debug", golog.LstdFlags),
		infoLogger:  *golog.New(infoOutput, prefix+" info", golog.LstdFlags),
		warnLogger:  *golog.New(errorOutput, prefix+" warn", golog.LstdFlags),
		errorLogger: *golog.New(errorOutput, prefix+" error", golog.LstdFlags),
		fatalLogger: *golog.New(errorOutput, prefix+" fatal", golog.LstdFlags),
	}
	return l
}

var defaultLogger *Logger

func init() {
	defaultLogger = NewLogger(os.Stdout, os.Stderr)
}

func Trace(v ...interface{}) {
	defaultLogger.Trace(v...)
}

func Debug(v ...interface{}) {
	defaultLogger.Debug(v...)
}

func Info(v ...interface{}) {
	defaultLogger.Info(v...)
}

func Warn(v ...interface{}) {
	defaultLogger.Warn(v...)
}

func Error(v ...interface{}) {
	defaultLogger.Error(v...)
}

func Fatal(v ...interface{}) {
	defaultLogger.Fatal(v...)
}

func Tracef(format string, v ...interface{}) {
	defaultLogger.Tracef(format, v...)
}

func Debugf(format string, v ...interface{}) {
	defaultLogger.Debugf(format, v...)
}

func Infof(format string, v ...interface{}) {
	defaultLogger.Infof(format, v...)
}

func Warnf(format string, v ...interface{}) {
	defaultLogger.Warnf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	defaultLogger.Errorf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	defaultLogger.Fatalf(format, v...)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

func (l *Logger) Tracef(format string, v ...interface{}) {
	if l.logLevel <= TraceLevel {
		l.traceLogger.Printf(format, v...)
	}
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.logLevel <= DebugLevel {
		l.debugLogger.Printf(format, v...)
	}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.logLevel <= InfoLevel {
		l.infoLogger.Printf(format, v...)
	}
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.logLevel <= WarnLevel {
		l.warnLogger.Printf(format, v...)
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.logLevel <= ErrorLevel {
		l.errorLogger.Printf(format, v...)
	}
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.fatalLogger.Fatalf(format, v...)
}

func (l *Logger) Trace(v ...interface{}) {
	if l.logLevel <= TraceLevel {
		l.traceLogger.Println(v...)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	if l.logLevel <= DebugLevel {
		l.debugLogger.Println(v...)
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.logLevel <= InfoLevel {
		l.infoLogger.Println(v...)
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.logLevel <= WarnLevel {
		l.warnLogger.Println(v...)
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.logLevel <= ErrorLevel {
		l.errorLogger.Println(v...)
	}
}

func (l *Logger) Fatal(v ...interface{}) {
	l.fatalLogger.Fatal(v...)
}

func (l *Logger) SetLevelFromString(level string) error {
	switch level {
	case "": // Default choice.
		l.SetLevel(InfoLevel)
	case "trace":
		l.SetLevel(TraceLevel)
	case "debug":
		l.SetLevel(DebugLevel)
	case "info":
		l.SetLevel(InfoLevel)
	case "warn":
		l.SetLevel(WarnLevel)
	case "error":
		l.SetLevel(ErrorLevel)
	default:
		return fmt.Errorf("unknown loglevel: %s", level)
	}
	return nil
}

func (l *Logger) SetLevel(level LogLevel) {
	l.logLevel = level
}

func SetLevelFromString(level string) error {
	return defaultLogger.SetLevelFromString(level)
}

func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}
