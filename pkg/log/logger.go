package log

import (
	"fmt"
	"log"
	"os"
)

// Level represents the level of logging.
type LogLevel uint8

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})

	Warn(v ...interface{})
	Warnf(format string, v ...interface{})

	Error(v ...interface{})
	Errorf(format string, v ...interface{})

	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})

	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
}

const (
	DefaultLogLevel  = InfoLevel
	DefaultNamespace = "Seata"
)

type SeataLogger struct {
	Loggers   map[string]*log.Logger
	Namespace string
	LogLevel  LogLevel
}

var seataLogger *SeataLogger

func init() {
	seataLogger = &SeataLogger{
		Loggers:   make(map[string]*log.Logger),
		Namespace: DefaultNamespace,
		LogLevel:  DefaultLogLevel,
	}
	seataLogger.Loggers["Stdout"] = log.New(os.Stdout, "", log.LstdFlags)
	seataLogger.Loggers["Stderr"] = log.New(os.Stderr, "", log.LstdFlags)
	AddLogger(seataLogger)
}

func merge(namespace, logLevel, msg string) string {
	return fmt.Sprintf("%s %s %s", namespace, logLevel, msg)
}

func (l *SeataLogger) Debug(v ...interface{}) {
	if DebugLevel < l.LogLevel || len(v) == 0 {
		return
	}
	l.Loggers["Stdout"].Print(merge(l.Namespace, "DEBUG", fmt.Sprint(v...)))
}

func (l *SeataLogger) Debugf(format string, v ...interface{}) {
	if DebugLevel < l.LogLevel {
		return
	}
	l.Loggers["Stdout"].Print(merge(l.Namespace, "DEBUG", fmt.Sprintf(format, v...)))
}

func (l *SeataLogger) Info(v ...interface{}) {
	if InfoLevel < l.LogLevel {
		return
	}
	l.Loggers["Stdout"].Print(merge(l.Namespace, "INFO", fmt.Sprint(v...)))
}

func (l *SeataLogger) Infof(format string, v ...interface{}) {
	if InfoLevel < l.LogLevel {
		return
	}
	l.Loggers["Stdout"].Print(merge(l.Namespace, "INFO", fmt.Sprintf(format, v...)))
}

func (l *SeataLogger) Warn(v ...interface{}) {
	if WarnLevel < l.LogLevel {
		return
	}
	l.Loggers["Stdout"].Print(merge(l.Namespace, "WARNING", fmt.Sprint(v...)))
}

func (l *SeataLogger) Warnf(format string, v ...interface{}) {
	if WarnLevel < l.LogLevel {
		return
	}
	l.Loggers["Stdout"].Print(merge(l.Namespace, "WARNING", fmt.Sprintf(format, v...)))
}

func (l *SeataLogger) Error(v ...interface{}) {
	if ErrorLevel < l.LogLevel {
		return
	}
	l.Loggers["Stderr"].Print(merge(l.Namespace, "ERROR", fmt.Sprint(v...)))
}

func (l *SeataLogger) Errorf(format string, v ...interface{}) {
	if ErrorLevel < l.LogLevel {
		return
	}
	l.Loggers["Stderr"].Print(merge(l.Namespace, "ERROR", fmt.Sprintf(format, v...)))
}

func (l *SeataLogger) Fatal(v ...interface{}) {
	if FatalLevel < l.LogLevel {
		return
	}
	l.Loggers["Stderr"].Print(merge(l.Namespace, "FATAL", fmt.Sprint(v...)))
}

func (l *SeataLogger) Fatalf(format string, v ...interface{}) {
	if FatalLevel < l.LogLevel {
		return
	}
	l.Loggers["Stderr"].Print(merge(l.Namespace, "FATAL", fmt.Sprintf(format, v...)))
}

func (l *SeataLogger) Panic(v ...interface{}) {
	if PanicLevel < l.LogLevel {
		return
	}
	l.Loggers["Stderr"].Print(merge(l.Namespace, "PANIC", fmt.Sprint(v...)))
}

func (l *SeataLogger) Panicf(format string, v ...interface{}) {
	if PanicLevel < l.LogLevel {
		return
	}
	l.Loggers["Stderr"].Print(merge(l.Namespace, "PANIC", fmt.Sprintf(format, v...)))
}
