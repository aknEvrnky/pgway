package badger

import badgerdb "github.com/dgraph-io/badger/v4"
import "go.uber.org/zap"

type badgerLogger struct{}

func NewBadgerLogger() badgerdb.Logger {
	return &badgerLogger{}
}

func (l *badgerLogger) Errorf(fmt string, args ...interface{}) {
	zap.S().Errorf(fmt, args...)
}
func (l *badgerLogger) Warningf(fmt string, args ...interface{}) {
	zap.S().Warnf(fmt, args...)
}
func (l *badgerLogger) Infof(fmt string, args ...interface{}) {
	zap.S().Infof(fmt, args...)
}
func (l *badgerLogger) Debugf(fmt string, args ...interface{}) {
	zap.S().Debugf(fmt, args...)
}
