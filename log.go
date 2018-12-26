package bcache

type Logger interface {
	Printf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}
