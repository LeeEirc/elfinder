package elfinder

import (
	"io"
)

type Lvl uint8

const (
	DEBUG Lvl = iota + 1
	INFO
	WARN
	ERROR
	OFF
	panicLevel
	fatalLevel
)

type (
	// Logger defines the logging interface.
	Logger interface {
		Output() io.Writer
		SetOutput(w io.Writer)
		Prefix() string
		SetPrefix(p string)
		Level() Lvl
		SetLevel(v Lvl)
		Print(i ...interface{})
		Printf(format string, args ...interface{})
		Debug(i ...interface{})
		Debugf(format string, args ...interface{})
		Info(i ...interface{})
		Infof(format string, args ...interface{})
		Warn(i ...interface{})
		Warnf(format string, args ...interface{})
		Error(i ...interface{})
		Errorf(format string, args ...interface{})
		Fatal(i ...interface{})
		Fatalf(format string, args ...interface{})
		Panic(i ...interface{})
		Panicf(format string, args ...interface{})
	}
)

type DefaultLogger struct {
	prefix string
	writer io.Writer
	level  Lvl
}

func (l *DefaultLogger) Level() Lvl {
	return l.level
}

func (l *DefaultLogger) log(level Lvl, format string, args ...interface{}) {

}
