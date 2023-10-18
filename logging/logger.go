package logging

import "time"

type Logger interface {
	LogExecutionTime(path string, duration time.Duration)
	LogError(path string, err error, errorCode int)
	Close()
}

type NilLogger struct{}

func (nl *NilLogger) LogExecutionTime(path string, duration time.Duration) {}
func (nl *NilLogger) LogError(path string, err error, errorCode int)       {}
func (nl *NilLogger) Close()                                               {}
