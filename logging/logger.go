package logging

import "time"

type Logger interface {
	LogExecutionTime(path string, handlerType string, duration time.Duration)
	LogError(path string, handlerType string, err error, errorCode int)
	LogMeasurement(measurement string, tags map[string]string, fields map[string]interface{})
	Close()
}

type NilLogger struct{}

func (nl *NilLogger) LogExecutionTime(path string, handlerType string, duration time.Duration) {}
func (nl *NilLogger) LogError(path string, handlerType string, err error, errorCode int)       {}
func (nl *NilLogger) LogMeasurement(measurement string, tags map[string]string, fields map[string]interface{}) {
}
func (nl *NilLogger) Close() {}
