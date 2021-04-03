package lib

import (
	"fmt"
	"io"
	slog "log"
	"os"
	"strings"
)

const (
	LOG_LEVEL_NO = iota
	LOG_LEVEL_ERROR
	LOG_LEVEL_WARN
	LOG_LEVEL_NOTICE
	LOG_LEVEL_LOG
	LOG_LEVEL_DEBUG
	LOG_LEVEL_PANIC
)

var logMap = map[string]int{
	"NO":     LOG_LEVEL_NO,
	"ERROR":  LOG_LEVEL_ERROR,
	"WARN":   LOG_LEVEL_WARN,
	"NOTICE": LOG_LEVEL_NOTICE,
	"LOG":    LOG_LEVEL_LOG,
	"DEBUG":  LOG_LEVEL_DEBUG,
}

func LogLevel(str string) (level int) {
	return logMap[strings.ToUpper(str)]
}

// var loger *log.Logger = log.New(os.Stdout, "", log.LstdFlags)

type Logger interface {
	LogLevel() int
	SetLogLevel(level int)
	Debug(args ...interface{})
	Debugln(args ...interface{})
	Debugf(format string, args ...interface{})
	Log(args ...interface{})
	Logln(args ...interface{})
	Logf(format string, args ...interface{})
	Notice(args ...interface{})
	Noticeln(args ...interface{})
	Noticef(format string, args ...interface{})
	Warn(args ...interface{})
	Warnln(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorln(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Close()
}

type DefaultLogger struct {
	Logger    *slog.Logger
	LOG_LEVEL int

	closable []io.Closer
}

func (l *DefaultLogger) LogLevel() int {
	return l.LOG_LEVEL
}

func (l *DefaultLogger) SetLogLevel(level int) {
	l.LOG_LEVEL = level
}

func (l *DefaultLogger) Debug(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_DEBUG {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[DEBUG] "
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprint(v...))
	}
}

func (l *DefaultLogger) Debugln(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_DEBUG {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[DEBUG]"
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprintln(v...))
	}
}

func (l *DefaultLogger) Debugf(format string, args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_DEBUG {
		l.Logger.Output(2, fmt.Sprintf("[DEBUG] "+format, args...))
	}
}

func (l *DefaultLogger) Log(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_LOG {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[INFO] "
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprint(v...))
	}
}

func (l *DefaultLogger) Logln(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_LOG {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[INFO]"
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprintln(v...))
	}
}

func (l *DefaultLogger) Logf(format string, args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_LOG {
		l.Logger.Output(2, fmt.Sprintf("[INFO] "+format, args...))
	}
}

func (l *DefaultLogger) Notice(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_NOTICE {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[NOTICE] "
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprint(v...))
	}
}

func (l *DefaultLogger) Noticeln(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_NOTICE {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[NOTICE]"
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprintln(v...))
	}
}

func (l *DefaultLogger) Noticef(format string, args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_NOTICE {
		l.Logger.Output(2, fmt.Sprintf("[NOTICE] "+format, args...))
	}
}

func (l *DefaultLogger) Warn(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_WARN {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[WARN] "
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprint(v...))
	}
}

func (l *DefaultLogger) Warnln(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_WARN {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[WARN]"
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprintln(v...))
	}
}

func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_WARN {
		l.Logger.Output(2, fmt.Sprintf("[WARN] "+format, args...))
	}
}

func (l *DefaultLogger) Error(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_ERROR {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[ERROR] "
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprint(v...))
	}
}

func (l *DefaultLogger) Errorln(args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_ERROR {
		v := make([]interface{}, 1, len(args)+1)
		v[0] = "[ERROR]"
		v = append(v, args...)
		l.Logger.Output(2, fmt.Sprintln(v...))
	}
}

func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
	if l.LOG_LEVEL >= LOG_LEVEL_ERROR {
		l.Logger.Output(2, fmt.Sprintf("[ERROR] "+format, args...))
	}
}

func (l *DefaultLogger) Fatalf(format string, args ...interface{}) {
	l.Logger.Output(2, fmt.Sprintf("[Fatal] "+format, args...))
	os.Exit(1)
}

func (l *DefaultLogger) Close() {
	for _, closer := range l.closable {
		closer.Close()
	}
}

var __appLogger Logger
var __logFile string
var __logLevel int
var __useStdout bool = true

func setAppLogger(path string, logLevel int, toStdout bool) {
	__useStdout = toStdout
	__logFile = path
	__logLevel = logLevel

	var logFile *os.File
	var mw io.Writer
	var err error
	if len(path) > 0 {
		logFile, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			slog.Fatalln("log conf error:", err.Error())
		}
		if __useStdout {
			mw = io.MultiWriter(logFile, os.Stdout)
		} else {
			mw = io.MultiWriter(logFile)
		}
	} else {
		mw = os.Stdout
	}

	dl := &DefaultLogger{
		Logger:    slog.New(mw, "", slog.LstdFlags|slog.Lshortfile),
		LOG_LEVEL: logLevel,
	}
	if len(path) > 0 {
		dl.closable = []io.Closer{logFile}
	}
	__appLogger = dl
}

func resetAppLogger(path string, logLevel int) {
	old := __appLogger
	setAppLogger(path, logLevel, __useStdout)
	if old != nil {
		old.Close()
	}
	__logLevel = logLevel
}

func SetLogFile(filePath string) {
	setAppLogger(filePath, __logLevel, __useStdout)
}

func UseStdout(open bool) {
	setAppLogger(__logFile, __logLevel, open)
}

func SetFileAndLevel(logFile string, levelStr string) {
	resetAppLogger(logFile, LogLevel(levelStr))
}

func init() {
	// 从环境变量中读取是否开启日志写入到标准输出
	notToStdout := os.Getenv("NO_LOG_TO_STDOUT")
	if notToStdout == "1" {
		__useStdout = false
	}

	setAppLogger("", LOG_LEVEL_DEBUG, __useStdout)
}

func AppLog() Logger {
	return __appLogger
}

func SetLogLevel(level int) {
	__appLogger.SetLogLevel(level)
}
