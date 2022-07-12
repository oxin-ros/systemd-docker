package lib

import (
	"fmt"
	"log"
	"os"
)

type logger struct {
	log *log.Logger
}

type logLevelRegistry struct {
	Fatal string
	Error string
	Warn  string
	Info  string
	Debug string
}

var (
	logLevel = &logLevelRegistry{
		Fatal: "FATAL",
		Error: "ERR",
		Warn:  "WARN",
		Info:  "INFO",
		Debug: "DBG",
	}
)

func NewLogger() *logger {
	var logger *logger = &logger{
		log: log.New(os.Stderr, "", 0),
	}
	return logger
}

func (l *logger) printf(logLevel string, format string, v ...interface{}) {
	l.log.Printf(fmt.Sprintf("<%s> %s", logLevel, format), v...)
}

func (l *logger) Fatalf(format string, v ...interface{}) {
	l.printf(logLevel.Fatal, format, v...)
}

func (l *logger) Errorf(format string, v ...interface{}) {
	l.printf(logLevel.Error, format, v...)
}

func (l *logger) Warnf(format string, v ...interface{}) {
	l.printf(logLevel.Warn, format, v...)
}

func (l *logger) Infof(format string, v ...interface{}) {
	l.printf(logLevel.Info, format, v...)
}

func (l *logger) Debugf(format string, v ...interface{}) {
	l.printf(logLevel.Debug, format, v...)
}
