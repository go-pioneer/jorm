package logger

import "strings"

type LEVEL int32
type UNIT int64
type _ROLLTYPE int //dailyRolling ,rollingFile

const _DATEFORMAT = "2006-01-02"

var logLevel LEVEL = 1

const (
	_       = iota
	KB UNIT = 1 << (iota * 10)
	MB
	GB
	TB
)

const (
	ALL LEVEL = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

const (
	_DAILY _ROLLTYPE = iota
	_ROLLFILE
)

func SetConsole(isConsole bool) {
	defaultlog.setConsole(isConsole)
}

func SetLevel(_level LEVEL) {
	defaultlog.setLevel(_level)
}

func SetFormat(logFormat string) {
	defaultlog.setFormat(logFormat)
}

func SetRollingFile(fileDir, fileName string, maxNumber int32, maxSize int64, _unit UNIT) {
	defaultlog.setRollingFile(fileDir, fileName, maxNumber, maxSize, _unit)
}

func SetRollingDaily(fileDir, fileName string) {
	defaultlog.setRollingDaily(fileDir, fileName)
}

func Debug(v ...interface{}) {
	defaultlog.debug(v...)
}
func Info(v ...interface{}) {
	defaultlog.info(v...)
}
func Warn(v ...interface{}) {
	defaultlog.warn(v...)
}
func Error(v ...interface{}) {
	defaultlog.error(v...)
}
func Fatal(v ...interface{}) {
	defaultlog.fatal(v...)
}
func IsDebug() bool {
	return defaultlog.isDebug()
}

func SetLevelFile(level LEVEL, dir, fileName string) {
	defaultlog.setLevelFile(level, dir, fileName)
}

func GetLevel(str string) LEVEL {
	str = strings.ToLower(str)
	switch str {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return ERROR
	}
}