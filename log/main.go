// Package log provides logging services. All logging goes through this layer so that we can
// easily change the logging implementation.
// The goal here is to be able to redirect logs with levels warn and error to stderr and have the rest
// go to stdout.
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

const defaultLevel = InfoLevel

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

const defaultOpts = golog.LstdFlags | golog.Lmsgprefix

// NewLogger creates a new logger. It takes two io.Writers, one for trace,debug,info and one for warn,error,fatal logs.
func NewLogger(infoOutput, errorOutput io.Writer) *Logger {
	l := &Logger{
		logLevel:    defaultLevel,
		traceLogger: *golog.New(infoOutput, "trace ", defaultOpts),
		debugLogger: *golog.New(infoOutput, "debug ", defaultOpts),
		infoLogger:  *golog.New(infoOutput, "info ", defaultOpts),
		warnLogger:  *golog.New(errorOutput, "warn ", defaultOpts),
		errorLogger: *golog.New(errorOutput, "error ", defaultOpts),
		fatalLogger: *golog.New(errorOutput, "fatal ", defaultOpts),
	}
	return l
}

// NewWithPrefix creates a new logger. It takes two io.Writers, one for trace,debug,info and one for warn,error,fatal logs.
// It also takes a prefix which will be added to all log messages.
func NewWithPrefix(infoOutput, errorOutput io.Writer, prefix string) *Logger {
	l := &Logger{
		logLevel:    defaultLevel,
		traceLogger: *golog.New(infoOutput, prefix+" trace ", defaultOpts),
		debugLogger: *golog.New(infoOutput, prefix+" debug ", defaultOpts),
		infoLogger:  *golog.New(infoOutput, prefix+" info ", defaultOpts),
		warnLogger:  *golog.New(errorOutput, prefix+" warn ", defaultOpts),
		errorLogger: *golog.New(errorOutput, prefix+" error ", defaultOpts),
		fatalLogger: *golog.New(errorOutput, prefix+" fatal ", defaultOpts),
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

// Parse the level string and return the corresponding LogLevel, if it doesn't recognize the level, return a string
func ParseLogLevel(level string) (LogLevel, error) {
	switch level {
	case "trace":
		return TraceLevel, nil
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "fatal":
		return FatalLevel, nil
	default:
		return 0, fmt.Errorf("unknown log level: '%s'", level)
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.logLevel = level
}

func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}
