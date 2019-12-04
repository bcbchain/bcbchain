package log

import (
	"io"

	kitlog "github.com/go-kit/kit/log"
)

// Logger is what any Tendermint library should take.
type Logger interface {
	Trace(msg string, keyvals ...interface{})
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
	Fatal(msg string, keyvals ...interface{})

	With(keyvals ...interface{}) Logger

	Flush()
}

type Loggerf interface {
	Logger

	Tracef(fmtStr string, vals ...interface{})
	Debugf(fmtStr string, vals ...interface{})
	Infof(fmtStr string, vals ...interface{})
	Warnf(fmtStr string, vals ...interface{})
	Errorf(fmtStr string, vals ...interface{})
	Fatalf(fmtStr string, vals ...interface{})

	AllowLevel(lvl string)
	SetOutputToFile(isToFile bool)
	SetOutputToScreen(isToScreen bool)
	SetOutputAsync(isAsync bool)
	SetOutputFileSize(maxFileSize int)
	SetWithThreadID(with bool)
}

// NewSyncWriter returns a new writer that is safe for concurrent use by
// multiple goroutines. Writes to the returned writer are passed on to w. If
// another write is already in progress, the calling goroutine blocks until
// the writer is available.
//
// If w implements the following interface, so does the returned writer.
//
//    interface {
//        Fd() uintptr
//    }
func NewSyncWriter(w io.Writer) io.Writer {
	return kitlog.NewSyncWriter(w)
}
