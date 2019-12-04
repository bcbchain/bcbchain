package log

type nopLogger struct{}

// Interface assertions
var _ Logger = (*nopLogger)(nil)

// NewNopLogger returns a logger that doesn't do anything.
func NewNopLogger() Logger { return &nopLogger{} }

func (nopLogger) Trace(string, ...interface{}) {}
func (nopLogger) Debug(string, ...interface{}) {}
func (nopLogger) Info(string, ...interface{})  {}
func (nopLogger) Warn(string, ...interface{})  {}
func (nopLogger) Error(string, ...interface{}) {}
func (nopLogger) Fatal(string, ...interface{}) {}

func (l *nopLogger) With(...interface{}) Logger {
	return l
}
func (nopLogger) Flush() {}
