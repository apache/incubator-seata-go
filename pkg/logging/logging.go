package logging

import (
	"fmt"
	"log"
	"os"
)

// Level represents the level of logging.
type LogLevel uint8

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
	Fatal
	Panic
)

type ILogger interface {
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
	DefaultLogLevel  = Info
	DefaultNamespace = "default"
)

type SeataLogger struct {
	loggers   []*log.Logger
	namespace string
	logLevel  LogLevel
}

var Logger *SeataLogger

func init() {
	var loggers = make([]*log.Logger, 0)
	loggers = append(loggers, log.New(os.Stdout, "", log.LstdFlags))
	Logger = &SeataLogger{
		loggers:   loggers,
		namespace: DefaultNamespace,
		logLevel:  DefaultLogLevel,
	}
}

func merge(namespace, logLevel, msg string) string {
	return fmt.Sprintf("%s %s %s", namespace, logLevel, msg)
}

func SetNamespace(namespace string) {
	Logger.namespace = namespace
}

func SetLogLevel(logLevel LogLevel) {
	Logger.logLevel = logLevel
}

func AddLogger(logger *log.Logger) {
	Logger.loggers = append(Logger.loggers, logger)
}

func (l *SeataLogger) Debug(v ...interface{}) {
	if Debug < l.logLevel || len(v) == 0 {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "DEBUG", fmt.Sprint(v...)))
	}
}

func (l *SeataLogger) Debugf(format string, v ...interface{}) {
	if Debug < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "DEBUG", fmt.Sprintf(format, v...)))
	}
}

func (l *SeataLogger) Info(v ...interface{}) {
	if Info < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "INFO", fmt.Sprint(v...)))
	}
}

func (l *SeataLogger) Infof(format string, v ...interface{}) {
	if Info < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "INFO", fmt.Sprintf(format, v...)))
	}
}

func (l *SeataLogger) Warn(v ...interface{}) {
	if Warn < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "WARNING", fmt.Sprint(v...)))
	}
}

func (l *SeataLogger) Warnf(format string, v ...interface{}) {
	if Warn < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "WARNING", fmt.Sprintf(format, v...)))
	}
}

func (l *SeataLogger) Error(v ...interface{}) {
	if Error < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "ERROR", fmt.Sprint(v...)))
	}
}

func (l *SeataLogger) Errorf(format string, v ...interface{}) {
	if Error < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "ERROR", fmt.Sprintf(format, v...)))
	}
}

func (l *SeataLogger) Fatal(v ...interface{}) {
	if Fatal < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "FATAL", fmt.Sprint(v...)))
	}
}

func (l *SeataLogger) Fatalf(format string, v ...interface{}) {
	if Fatal < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "FATAL", fmt.Sprintf(format, v...)))
	}
}

func (l *SeataLogger) Panic(v ...interface{}) {
	if Panic < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "PANIC", fmt.Sprint(v...)))
	}
}

func (l *SeataLogger) Panicf(format string, v ...interface{}) {
	if Panic < l.logLevel {
		return
	}
	for _, log := range l.loggers {
		log.Print(merge(l.namespace, "PANIC", fmt.Sprintf(format, v...)))
	}
}
