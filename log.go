package bcache

// Logger defines interface that must be implemented by
// the logger of bcache
type Logger interface {
	Errorf(format string, v ...interface{})
	Printf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

// nopLogger is logger that doing nothing
type nopLogger struct {
}

func (nl *nopLogger) Errorf(format string, v ...interface{}) {
}
func (nl *nopLogger) Printf(format string, v ...interface{}) {
}
func (nl *nopLogger) Debugf(format string, v ...interface{}) {
}
