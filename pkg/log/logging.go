package log

var loggers = make([]Logger,0)

func AddLogger(logger Logger) {
	l := append(loggers, logger)
	loggers = l
}

func Debug(v ...interface{}) {
	for _, log := range loggers {
		log.Debug(v)
	}
}

func Debugf(format string, v ...interface{}) {
	for _, log := range loggers {
		log.Debugf(format, v)
	}
}

func Info(v ...interface{}) {
	for _, log := range loggers {
		log.Info(v)
	}
}

func Infof(format string, v ...interface{}) {
	for _, log := range loggers {
		log.Infof(format, v)
	}
}

func Warn(v ...interface{}) {
	for _, log := range loggers {
		log.Warn(v)
	}
}

func Warnf(format string, v ...interface{}) {
	for _, log := range loggers {
		log.Warnf(format, v)
	}
}

func Error(v ...interface{}) {
	for _, log := range loggers {
		log.Error(v)
	}
}

func Errorf(format string, v ...interface{}) {
	for _, log := range loggers {
		log.Errorf(format, v)
	}
}

func Fatal(v ...interface{}) {
	for _, log := range loggers {
		log.Fatal(v)
	}
}

func Fatalf(format string, v ...interface{}) {
	for _, log := range loggers {
		log.Fatalf(format, v)
	}
}

func Panic(v ...interface{}) {
	for _, log := range loggers {
		log.Panic(v)
	}
}

func Panicf(format string, v ...interface{}) {
	for _, log := range loggers {
		log.Panicf(format, v)
	}
}

func SetSeataLoggerLogLevel(level LogLevel) {
	seataLogger.LogLevel = level
}