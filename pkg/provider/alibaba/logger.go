package alibaba

import (
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = newLogger()

type logLevel int

const (
	logLevelWarn  logLevel = iota
	logLevelInfo  logLevel = iota
	logLevelDebug logLevel = iota
)

type logger struct {
	logr.Logger
}

func (l logLevel) Level() int {
	return int(l)
}

func newLogger() *logger {
	return &logger{
		Logger: ctrl.Log.WithName("provider").WithName("alibaba").WithName("kms"),
	}
}

var _ retryablehttp.LeveledLogger = (*logger)(nil)
var _ retryablehttp.Logger = (*logger)(nil)

func (l *logger) WithField(key string, value interface{}) *logger {
	return l.WithFields(key, value)
}

func (l *logger) WithError(err error) *logger {
	return l.WithFields("error", err)
}

func (l *logger) WithFields(keysAndValues ...interface{}) *logger {
	newLogger := *l
	newLogger.Logger = l.Logger.WithValues(keysAndValues...)
	return &newLogger
}

func (l *logger) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(nil, msg, keysAndValues...)
}

func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.V(logLevelInfo.Level()).Info(msg, keysAndValues...)
}

func (l *logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.V(logLevelDebug.Level()).Info(msg, keysAndValues...)
}

func (l *logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Logger.V(logLevelWarn.Level()).Info(msg, keysAndValues...)
}

func (l *logger) Printf(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, keysAndValues...)
}
