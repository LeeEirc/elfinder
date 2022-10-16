package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type Lvl uint8

const (
	DEBUG Lvl = iota + 1
	INFO
	WARN
	ERROR
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

var GlobalLogger = DefaultLogger{
	prefix: "[Elfinder]",
	writer: os.Stdout,
	level:  0,
}

var _ Logger = (*DefaultLogger)(nil)

type DefaultLogger struct {
	prefix string
	writer io.Writer
	level  Lvl
	mutex  sync.Mutex
}

func (l *DefaultLogger) Output() io.Writer {
	return l.writer
}
func (l *DefaultLogger) SetOutput(w io.Writer) {
	l.writer = w
}

func (l *DefaultLogger) Prefix() string {
	return l.prefix
}
func (l *DefaultLogger) SetPrefix(p string) {
	l.prefix = p
}

func (l *DefaultLogger) Print(i ...interface{}) {
	l.log(0, "", i...)
}

func (l *DefaultLogger) Printf(format string, args ...interface{}) {
	l.log(0, format, args...)
}

func (l *DefaultLogger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

func (l *DefaultLogger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *DefaultLogger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

func (l *DefaultLogger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *DefaultLogger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *DefaultLogger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *DefaultLogger) Fatal(i ...interface{}) {
	l.log(fatalLevel, "", i...)
	os.Exit(1)
}

func (l *DefaultLogger) Fatalf(format string, args ...interface{}) {
	l.log(fatalLevel, format, args...)
	os.Exit(1)
}

func (l *DefaultLogger) Panic(i ...interface{}) {
	l.log(panicLevel, "", i...)
	panic(fmt.Sprint(i...))
}

func (l *DefaultLogger) Panicf(format string, args ...interface{}) {
	l.log(panicLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}

func (l *DefaultLogger) Level() Lvl {
	return l.level
}

func (l *DefaultLogger) SetLevel(v Lvl) {
	l.level = v
}

func (l *DefaultLogger) log(level Lvl, format string, args ...interface{}) {
	if l.level <= level || level == 0 {
		message := ""
		if format == "" {
			message = fmt.Sprint(args...)
		} else {
			message = fmt.Sprintf(format, args...)
		}
		l.mutex.Lock()
		defer l.mutex.Unlock()
		if l.prefix != "" {
			message = fmt.Sprintf("%s %s", l.prefix, message)
		}
		_, _ = l.writer.Write([]byte(message))
		_, _ = l.writer.Write([]byte{'\n'})
	}
}
