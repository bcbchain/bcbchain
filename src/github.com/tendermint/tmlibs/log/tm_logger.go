package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const lengthOfChanLogInfoBuffer = 10000

type logInfo struct {
	time        string
	level       string
	msg         string
	keyvals     []interface{}
	fmt         bool
	fmtStr      string
	vals        []interface{}
	goroutineID string
}

type giLogger struct {
	path             string
	module           string
	allowed          level
	isOutputAsync    bool
	isOutputToScreen bool
	isOutputToFile   bool
	isOutput         bool
	withThreadID     bool
	maxFileSize      int
	curFileSize      int
	timeZone         string
	fileName         string
	logFile          *os.File
	caStop           chan bool    //用于控制异步写线程
	caStopped        chan bool    //用于控制异步写线程
	cf               chan logInfo //用于控制异步构造日志文本串
	mutex            *sync.Mutex
}

// Interface assertions
var _ Loggerf = (*giLogger)(nil)

// NewTMLogger returns a logger that encodes msg and keyvals to the Writer
// Note that underlying logger could be swapped with something else.
func NewTMLogger(path, module string) Loggerf {
	return &giLogger{
		path:             path,
		module:           module,
		allowed:          levelFatal | levelError | levelWarn | levelInfo,
		isOutputAsync:    false, //the default must be false, do not modify!!!
		isOutputToScreen: false,
		isOutputToFile:   true,
		isOutput:         true,
		maxFileSize:      DEFAULT_FILE_SIZE,
		curFileSize:      0,
		timeZone:         fmt.Sprintf("%v", time.Now().Format(TIMEZONEFORMAT)),
		fileName:         "",
		logFile:          nil,
		caStop:           make(chan bool, 1),
		caStopped:        make(chan bool, 1),
		cf:               make(chan logInfo, lengthOfChanLogInfoBuffer),
		mutex:            new(sync.Mutex),
		withThreadID:     true,
	}
}

func (log *giLogger) SetOutputToFile(isToFile bool) {
	log.isOutputToFile = isToFile
	log.isOutput = log.isOutputToFile || log.isOutputToScreen
}

func (log *giLogger) SetOutputToScreen(isToScreen bool) {
	log.isOutputToScreen = isToScreen
	log.isOutput = log.isOutputToFile || log.isOutputToScreen
}

func (log *giLogger) SetOutputFileSize(maxFileSize int) {
	if maxFileSize > 0 {
		log.maxFileSize = maxFileSize
	}
}

func (log *giLogger) SetWithThreadID(with bool) {
	log.withThreadID = with
}

func (log *giLogger) AllowLevel(lvl string) {
	switch strings.ToLower(lvl) {
	case "trace":
		log.allowed = levelFatal | levelError | levelWarn | levelInfo | levelDebug | levelTrace
	case "debug":
		log.allowed = levelFatal | levelError | levelWarn | levelInfo | levelDebug
	case "info":
		log.allowed = levelFatal | levelError | levelWarn | levelInfo
	case "warn":
		log.allowed = levelFatal | levelError | levelWarn
	case "error":
		log.allowed = levelFatal | levelError
	case "fatal":
		log.allowed = levelFatal
	case "none":
		log.allowed = 0
	default:
		fmt.Printf("Expected either \"trace\", \"debug\", \"info\", \"warn\", \"error\", \"fatal\" or \"none\" level, given %s", lvl)
	}
}

func (log *giLogger) SetOutputAsync(isAsync bool) {
	if log.isOutputAsync == isAsync {
		return
	}
	if log.isOutputAsync {
		log.isOutputAsync = false
		log.caStop <- true //设置停止标识
		<-log.caStopped    //等待线程停止
	} else {
		log.isOutputAsync = true
		go asyncRun(log)
	}
}

// Trace logs a message at level Trace.
func (log *giLogger) Trace(msg string, keyvals ...interface{}) {
	levelAllowed := log.allowed&levelTrace != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.Log("Trace", msg, keyvals...)
}

// Debug logs a message at level Debug.
func (log *giLogger) Debug(msg string, keyvals ...interface{}) {
	levelAllowed := log.allowed&levelDebug != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.Log("Debug", msg, keyvals...)
}

// Info logs a message at level Info.
func (log *giLogger) Info(msg string, keyvals ...interface{}) {
	levelAllowed := log.allowed&levelInfo != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.Log("Info", msg, keyvals...)
}

// Warn logs a message at level Debug.
func (log *giLogger) Warn(msg string, keyvals ...interface{}) {
	levelAllowed := log.allowed&levelWarn != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.Log("Warn", msg, keyvals...)
}

// Error logs a message at level Error.
func (log *giLogger) Error(msg string, keyvals ...interface{}) {
	levelAllowed := log.allowed&levelError != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.Log("Error", msg, keyvals...)
}

// Fatal logs a message at level Fatal.
func (log *giLogger) Fatal(msg string, keyvals ...interface{}) {
	levelAllowed := log.allowed&levelFatal != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.Log("Fatal", msg, keyvals...)
}

// Trace logs a message at level Trace.
func (log *giLogger) Tracef(fmtStr string, vals ...interface{}) {
	levelAllowed := log.allowed&levelTrace != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.LogEx("Trace", fmtStr, vals...)
}

// Debug logs a message at level Debug.
func (log *giLogger) Debugf(fmtStr string, vals ...interface{}) {
	levelAllowed := log.allowed&levelDebug != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.LogEx("Debug", fmtStr, vals...)
}

// Info logs a message at level Info.
func (log *giLogger) Infof(fmtStr string, vals ...interface{}) {
	levelAllowed := log.allowed&levelInfo != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.LogEx("Info", fmtStr, vals...)
}

// Warn logs a message at level Debug.
func (log *giLogger) Warnf(fmtStr string, vals ...interface{}) {
	levelAllowed := log.allowed&levelWarn != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.LogEx("Warn", fmtStr, vals...)
}

// Error logs a message at level Error.
func (log *giLogger) Errorf(fmtStr string, vals ...interface{}) {
	levelAllowed := log.allowed&levelError != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.LogEx("Error", fmtStr, vals...)
}

// Fatal logs a message at level Fatal.
func (log *giLogger) Fatalf(fmtStr string, vals ...interface{}) {
	levelAllowed := log.allowed&levelFatal != 0
	if !levelAllowed || !log.isOutput {
		return
	}
	log.LogEx("Fatal", fmtStr, vals...)
}

func GetGID() string {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	return string(b)
}

func (log *giLogger) genLogText(strTime, goroutineID, level, msg string, keyvals []interface{}) (logText string) {
	var kvs []interface{}
	var tag string

	if goroutineID != "" {
		tag = fmt.Sprintf(
			"%v[%v][%-5v][%5s] ",
			strTime,
			log.timeZone,
			level,
			goroutineID)
	} else {
		tag = fmt.Sprintf(
			"%v[%v][%-5v] ",
			strTime,
			log.timeZone,
			level)
	}

	kvs = append(kvs, tag, msg)
	kvs = append(kvs, keyvals...)
	if len(kvs)%2 != 0 {
		kvs = append(kvs, "<Missing something!>")
	}

	logText = ""
	for i, v := range kvs {
		if i <= 1 {
			logText += fmt.Sprintf("%v", v)
		} else if i%2 == 0 {
			logText += fmt.Sprintf("[%v=", v)
		} else {
			logText += fmt.Sprintf("%v]", v)
		}
		if i%2 != 0 && i != len(kvs)-1 {
			logText += fmt.Sprintf(", ")
		}
	}
	logText += "\n"
	return
}

func (log *giLogger) genLogTextEx(strTime, goroutineID, level, fmtStr string, vals []interface{}) (logText string) {
	if goroutineID != "" {
		logText = fmt.Sprintf(
			"%v[%v][%-5v][%5s] ",
			strTime,
			log.timeZone,
			level,
			goroutineID)
	} else {
		logText = fmt.Sprintf(
			"%v[%v][%-5v] ",
			strTime,
			log.timeZone,
			level)
	}
	logText += fmt.Sprintf(fmtStr, vals...)
	logText += "\n"
	return
}

func (log *giLogger) newLogFile() {
	//目录不存在需要创建
	_, err := os.Stat(log.path)
	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(log.path, 0775)
		}
	}

	//创建新的日志文件
	log.fileName = log.path + "/" + log.module + time.Now().UTC().Format(DATEFORMAT) + ".log"
	log.logFile, err = os.OpenFile(log.fileName, os.O_WRONLY|os.O_CREATE, 0644)
	log.curFileSize = 0

	if err == nil {
		dup2(int(log.logFile.Fd()), int(os.Stderr.Fd()))
	}
}

func (log *giLogger) flush(logBuf *bytes.Buffer) {
	if log.isOutputToScreen {
		fmt.Printf(logBuf.String())
	}
	if log.isOutputToFile {
		if log.logFile == nil {
			//第一次输出日志到文件，创建日志文件
			log.newLogFile()
		}
		_, err := os.Stat(log.fileName)
		if err != nil {
			//读取文件状态失败，可能文件被删除了，再次创建日志文件
			log.newLogFile()
		}
		if log.logFile == nil {
			//日志文件创建失败
			if !log.isOutputToScreen {
				//在没有设定向屏幕输出的情况下，强制向屏幕打印日志
				fmt.Printf(logBuf.String())
			}
			return
		}
		log.logFile.Write(logBuf.Bytes())
		log.logFile.Sync()

		//判断日志文件是否已经达到预设的大小
		if log.curFileSize += logBuf.Len(); log.curFileSize >= log.maxFileSize {
			log.logFile.Close()
			log.newLogFile()
		}
	} else {
		if log.logFile != nil {
			log.logFile.Close()
			log.logFile = nil
			log.curFileSize = 0
		}
	}
}

func (log *giLogger) flushMutex(logBuf *bytes.Buffer) {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	log.flush(logBuf)
}

func asyncRun(log *giLogger) {
	for {
		lenChan := len(log.cf)
		if lenChan > 0 {
			if lenChan == lengthOfChanLogInfoBuffer {
				print("log buffer full!!!\n")
			}
			logBuf := bytes.NewBuffer(nil)
			logText := ""
			if li := <-log.cf; li.fmt == false {
				logText = log.genLogText(li.time, li.goroutineID, li.level, li.msg, li.keyvals)
			} else {
				logText = log.genLogTextEx(li.time, li.goroutineID, li.level, li.fmtStr, li.vals)
			}
			logBuf.WriteString(logText)
			log.flush(logBuf)
		} else {
			if len(log.caStop) > 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	log.caStopped <- true
}

// Log a message at specific level.
func (log *giLogger) Log(level, msg string, keyvals ...interface{}) {
	strTime := time.Now().UTC().Format(DATETIMEFORMAT)
	if log.isOutputAsync {
		li := logInfo{
			time:  strTime,
			level: level,
			fmt:   false,
			msg:   fileLine() + msg,
		}
		if log.withThreadID {
			li.goroutineID = GetGID()
		}
		li.keyvals = append(li.keyvals, keyvals...)
		log.cf <- li
	} else {
		rid := ""
		if log.withThreadID {
			rid = GetGID()
		}
		logText := log.genLogText(strTime, rid, level, fileLine()+msg, keyvals)
		log.flushMutex(bytes.NewBuffer([]byte(logText)))
	}
}

func fileLine() string {
	_, fileName, fileLine, ok := runtime.Caller(3)

	// tendermint 有一层 filter ，调用栈深一层
	exe, _ := os.Executable()
	exeName := filepath.Base(exe)
	if exeName == "tendermint" || exeName == "tmcore" {
		_, fileName, fileLine, ok = runtime.Caller(4)
	}

	var s string
	if ok {
		f := strings.Split(fileName, "/")
		s = fmt.Sprintf("[%s:%d] ", f[len(f)-1], fileLine)
	} else {
		s = ""
	}
	return s
}

// Log a message at specific level.
func (log *giLogger) LogEx(level, fmtStr string, vals ...interface{}) {
	strTime := time.Now().UTC().Format(DATETIMEFORMAT)
	if log.isOutputAsync {
		li := logInfo{
			time:   strTime,
			level:  level,
			fmt:    true,
			fmtStr: fileLine() + fmtStr,
			vals:   vals,
		}
		if log.withThreadID {
			li.goroutineID = GetGID()
		}
		log.cf <- li
	} else {
		rid := ""
		if log.withThreadID {
			rid = GetGID()
		}
		logText := log.genLogTextEx(strTime, rid, level, fileLine()+fmtStr, vals)
		log.flushMutex(bytes.NewBuffer([]byte(logText)))
	}
}

func (log *giLogger) Flush() {
	if log.isOutputAsync == true {
		log.SetOutputAsync(false)
	}
	if len(log.cf) > 0 {
		logBuf := bytes.NewBuffer(nil)

		for {
			if len(log.cf) > 0 {
				logText := ""
				if li := <-log.cf; li.fmt == false {
					logText = log.genLogText(li.time, li.goroutineID, li.level, li.msg, li.keyvals)
				} else {
					logText = log.genLogTextEx(li.time, li.goroutineID, li.level, li.fmtStr, li.vals)
				}
				logBuf.WriteString(logText)
			} else {
				log.flushMutex(logBuf)
				break
			}
		}
	}
}

// With returns a new contextual logger with keyvals prepended to those passed
// to calls to Trace, Debug, Info, Warn, Error or Fatal.
func (log *giLogger) With(keyvals ...interface{}) Logger {
	return log
}
