package log

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
	DATEFORMAT        = "20060102150405"
	DATETIMEFORMAT    = "[2006-01-02 15:04:05.000]"
	TIMEZONEFORMAT    = "-07"
	DEFAULT_FILE_SIZE = 20 * 1024 * 1024 //20MB
)

type Logger interface {
	Trace(msg string, keyvals ...interface{})
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
	Fatal(msg string, keyvals ...interface{})

	With(keyvals ...interface{}) Logger
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

	Flush()
}
