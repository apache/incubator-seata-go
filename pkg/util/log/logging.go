package log

import getty "github.com/apache/dubbo-getty"

func Debug(v ...interface{}) {
	getty.GetLogger().Debug(v)
}

func Debugf(format string, v ...interface{}) {
	getty.GetLogger().Debugf(format, v)
}

func Info(v ...interface{}) {
	getty.GetLogger().Info(v)
}

func Infof(format string, v ...interface{}) {
	getty.GetLogger().Infof(format, v)
}

func Warn(v ...interface{}) {
	getty.GetLogger().Warn(v)
}

func Warnf(format string, v ...interface{}) {
	getty.GetLogger().Warnf(format, v)
}

func Error(v ...interface{}) {
	getty.GetLogger().Error(v)
}

func Errorf(format string, v ...interface{}) {
	getty.GetLogger().Errorf(format, v)
}
