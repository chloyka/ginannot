package ginannot

import "log"

type Logger interface {
	Info(args ...interface{})
	Debug(args ...interface{})
	Error(args ...interface{})
	Panic(args ...interface{})
	Fatal(args ...interface{})
}

type DefaultLogger struct{}

func (l *DefaultLogger) Info(args ...interface{}) {
	log.Println(args...)
}

func (l *DefaultLogger) Debug(args ...interface{}) {
	log.Println(args...)
}

func (l *DefaultLogger) Error(args ...interface{}) {
	log.Println(args...)
}

func (l *DefaultLogger) Panic(args ...interface{}) {
	log.Println(args...)
}

func (l *DefaultLogger) Fatal(args ...interface{}) {
	log.Println(args...)
}
