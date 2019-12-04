package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type level byte

const (
	levelTrace level = 1 << iota
	levelDebug
	levelInfo
	levelWarn
	levelError
	levelFatal
)

const (
	DATEFORMAT         = "20060102150405"
	DATETIMEFORMAT     = "[2006-01-02 15:04:05.000]"
	TIMEZONEFORMAT     = "-07"
	DEFAULT_FILE_COUNT = 10
	DEFAULT_FILE_SIZE  = 20 * 1024 * 1024 //20MB
)

// A filter struct to slice log file
type filter struct {
	next           Logger
	logfile        string
	loglevel       string
	mtx            sync.Mutex
	allowed        level            // XOR'd levels for default case
	allowedKeyvals map[keyval]level // When key-value match, use this level
}

type keyval struct {
	key   interface{}
	value interface{}
}

var fl = (*filter)(nil)

// NewFilter wraps next and implements filtering. See the commentary on the
// Option functions for a detailed description of how to configure levels. If
// no options are provided, all leveled log events created with Debug, Info or
// Error helper methods are squelched.
func NewFilter(next Logger, options ...Option) Logger {
	l := &filter{
		next:           next,
		allowedKeyvals: make(map[keyval]level),
	}
	for _, option := range options {
		option(l)
	}
	return l
}

// NewFileFilter wraps next and implements filtering. See the commentary on the
// Option functions for a detailed description of how to configure levels. If
// no options are provided, all leveled log events created with Debug, Info or
// Error helper methods are squelched.
func NewFileFilter(file string, lvl string, next Logger, options ...Option) Logger {
	if len(file) == 0 {
		return nil
	}
	fl = &filter{
		next:           next,
		logfile:        file,
		loglevel:       lvl,
		allowedKeyvals: make(map[keyval]level),
	}
	for _, option := range options {
		option(fl)
	}
	go rotateRoutine(fl.logfile, fl.loglevel)
	return fl
}

// rotateRoutine function slices log file by size limitation in a routine
func rotateRoutine(file string, level string) {
	var src, dst *os.File
	var f os.FileInfo
	var err error
	var isRotate bool
	var size int64
	var newfile string

	for {
		f, err = os.Stat(file)
		if err != nil {
			panic(err)
		}
		isRotate = false
		if size = f.Size(); size > DEFAULT_FILE_SIZE {
			isRotate = true
		}

		if isRotate == true {
			checkAndRemoveFile(file)

			fl.mtx.Lock()
			newfile = fmt.Sprintf("%s-%s", file, time.Now().UTC().Format(DATEFORMAT))
			if dst, err = os.OpenFile(newfile, os.O_WRONLY|os.O_CREATE, 0644); err == nil {
				if src, err = os.OpenFile(file, os.O_RDWR, 0644); err == nil {
					io.Copy(dst, src)
					src.Truncate(0)
					src.Close()
				}
				dst.Close()
			}
			fl.mtx.Unlock()
		}
		time.Sleep(10 * time.Second)
	}
	return
}

func checkAndRemoveFile(file string) {
	var count int32
	var odf os.FileInfo

	bn := filepath.Base(file)
	fl, _ := os.Stat(file)
	tm := fl.ModTime()
	//TO check and remove the oldest log file
	dir, _ := filepath.Abs(filepath.Dir(file))
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), bn+"-") {
			count++

			if f.ModTime().Before(tm) {
				tm = f.ModTime()
				odf = f
			}
		}
	}

	if count >= DEFAULT_FILE_COUNT {
		os.Remove(dir + "/" + odf.Name())
	}
}
func rotate(file string, level string) {
	f, err := os.Stat(file)
	isRotate := false
	if err == nil {
		size := f.Size()
		if size > DEFAULT_FILE_SIZE {
			isRotate = true
		}
	}
	if isRotate == true {
		fl.mtx.Lock()
		defer fl.mtx.Unlock()
		newfile := fmt.Sprintf("%s-%s", file, time.Now().UTC().Format(DATEFORMAT))
		dst, err := os.OpenFile(newfile, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return
		}
		defer dst.Close()
		src, err := os.OpenFile(file, os.O_RDWR, 0644)
		if err != nil {
			return
		}
		defer src.Close()
		if _, err = io.Copy(dst, src); err != nil {
			fmt.Println("Failed to copy source file to destination file ", err)
			return
		}
		if err = src.Truncate(0); err != nil {
			fmt.Println("Failed to empty source file ")
			return
		}
	}

	return
}

func (l *filter) Trace(msg string, keyvals ...interface{}) {
	levelAllowed := l.allowed&levelTrace != 0
	if !levelAllowed {
		return
	}
	l.next.Trace(msg, keyvals...)
}

func (l *filter) Debug(msg string, keyvals ...interface{}) {
	levelAllowed := l.allowed&levelDebug != 0
	if !levelAllowed {
		return
	}
	l.next.Debug(msg, keyvals...)
}

func (l *filter) Info(msg string, keyvals ...interface{}) {
	levelAllowed := l.allowed&levelInfo != 0
	if !levelAllowed {
		return
	}
	l.next.Info(msg, keyvals...)
}

func (l *filter) Warn(msg string, keyvals ...interface{}) {
	levelAllowed := l.allowed&levelWarn != 0
	if !levelAllowed {
		return
	}
	l.next.Warn(msg, keyvals...)
}

func (l *filter) Error(msg string, keyvals ...interface{}) {
	levelAllowed := l.allowed&levelError != 0
	if !levelAllowed {
		return
	}
	l.next.Error(msg, keyvals...)
}

func (l *filter) Fatal(msg string, keyvals ...interface{}) {
	levelAllowed := l.allowed&levelFatal != 0
	if !levelAllowed {
		return
	}
	l.next.Fatal(msg, keyvals...)
}

func (l *filter) Flush() {

}

// With implements Logger by constructing a new filter with a keyvals appended
// to the logger.
//
// If custom level was set for a keyval pair using one of the
// Allow*With methods, it is used as the logger's level.
//
// Examples:
//     logger = log.NewFilter(logger, log.AllowError(), log.AllowInfoWith("module", "crypto"))
//		 logger.With("module", "crypto").Info("Hello") # produces "I... Hello module=crypto"
//
//     logger = log.NewFilter(logger, log.AllowError(), log.AllowInfoWith("module", "crypto"), log.AllowNoneWith("user", "Sam"))
//		 logger.With("module", "crypto", "user", "Sam").Info("Hello") # returns nil
//
//     logger = log.NewFilter(logger, log.AllowError(), log.AllowInfoWith("module", "crypto"), log.AllowNoneWith("user", "Sam"))
//		 logger.With("user", "Sam").With("module", "crypto").Info("Hello") # produces "I... Hello module=crypto user=Sam"
func (l *filter) With(keyvals ...interface{}) Logger {
	for i := len(keyvals) - 2; i >= 0; i -= 2 {
		for kv, allowed := range l.allowedKeyvals {
			if keyvals[i] == kv.key && keyvals[i+1] == kv.value {
				return &filter{next: l.next.With(keyvals...), allowed: allowed, allowedKeyvals: l.allowedKeyvals}
			}
		}
	}
	return &filter{next: l.next.With(keyvals...), allowed: l.allowed, allowedKeyvals: l.allowedKeyvals}
}

//--------------------------------------------------------------------------------

// Option sets a parameter for the filter.
type Option func(*filter)

// AllowLevel returns an option for the given level or error if no option exist
// for such level.
func AllowLevel(lvl string) (Option, error) {
	switch lvl {
	case "trace":
		return AllowTrace(), nil
	case "debug":
		return AllowDebug(), nil
	case "info":
		return AllowInfo(), nil
	case "warn":
		return AllowWarn(), nil
	case "error":
		return AllowError(), nil
	case "fatal":
		return AllowFatal(), nil
	case "none":
		return AllowNone(), nil
	default:
		return nil, fmt.Errorf("Expected either \"trace\", \"debug\", \"info\", \"warn\", \"error\", \"fatal\" or \"none\" level, given %s", lvl)
	}
}

// AllowAll is an alias for AllowDebug.
func AllowAll() Option {
	return AllowTrace()
}

// AllowTrace allows fatal, error, info, debug and trace level log events to pass.
func AllowTrace() Option {
	return allowed(levelFatal | levelError | levelWarn | levelInfo | levelDebug | levelTrace)
}

// AllowDebug allows fatal, error, info and debug level log events to pass.
func AllowDebug() Option {
	return allowed(levelFatal | levelError | levelWarn | levelInfo | levelDebug)
}

// AllowInfo allows fatal, error and info level log events to pass.
func AllowInfo() Option {
	return allowed(levelFatal | levelError | levelWarn | levelInfo)
}

// AllowWarn allows fatal, error and warn level log events to pass.
func AllowWarn() Option {
	return allowed(levelFatal | levelError | levelWarn)
}

// AllowError allows fatal and error level log events to pass.
func AllowError() Option {
	return allowed(levelFatal | levelError)
}

// AllowFatal allows only fatal level log events to pass.
func AllowFatal() Option {
	return allowed(levelFatal)
}

// AllowNone allows no leveled log events to pass.
func AllowNone() Option {
	return allowed(0)
}

func allowed(allowed level) Option {
	return func(l *filter) { l.allowed = allowed }
}

// AllowDebugWith allows error, info and debug level log events to pass for a specific key value pair.
func AllowDebugWith(key interface{}, value interface{}) Option {
	return func(l *filter) {
		l.allowedKeyvals[keyval{key, value}] = levelError | levelWarn | levelInfo | levelDebug
	}
}

// AllowInfoWith allows error, warn and info level log events to pass for a specific key value pair.
func AllowInfoWith(key interface{}, value interface{}) Option {
	return func(l *filter) {
		l.allowedKeyvals[keyval{key, value}] = levelError | levelWarn | levelInfo
	}
}

// AllowWarnWith allows error and warn level log events to pass for a specific key value pair.
func AllowWarnWith(key interface{}, value interface{}) Option {
	return func(l *filter) {
		l.allowedKeyvals[keyval{key, value}] = levelError | levelWarn
	}
}

// AllowErrorWith allows only error level log events to pass for a specific key value pair.
func AllowErrorWith(key interface{}, value interface{}) Option {
	return func(l *filter) {
		l.allowedKeyvals[keyval{key, value}] = levelError
	}
}

// AllowNoneWith allows no leveled log events to pass for a specific key value pair.
func AllowNoneWith(key interface{}, value interface{}) Option {
	return func(l *filter) { l.allowedKeyvals[keyval{key, value}] = 0 }
}
