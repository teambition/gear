package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/teambition/gear"
)

var crlfEscaper = strings.NewReplacer("\r", "\\r", "\n", "\\n")

// Messager is implemented by any value that has a Format method and a String method.
// They are using by Logger to format value to string.
type Messager interface {
	fmt.Stringer
	Format() (string, error)
}

// Log records key-value pairs for structured logging.
type Log map[string]interface{}

// Format try to marshal the structured log with json.Marshal.
func (l Log) Format() (string, error) {
	res, err := json.Marshal(l)
	if err == nil {
		return string(res), nil
	}
	return "", err
}

// GoString implemented fmt.GoStringer interface, returns a Go-syntax string.
func (l Log) GoString() string {
	count := len(l)
	buf := bytes.NewBufferString("Log{")
	for key, value := range l {
		if count--; count == 0 {
			fmt.Fprintf(buf, "%s:%#v}", key, value)
		} else {
			fmt.Fprintf(buf, "%s:%#v, ", key, value)
		}
	}
	return buf.String()
}

// String implemented fmt.Stringer interface, returns a Go-syntax string.
func (l Log) String() string {
	return l.GoString()
}

// From copy values from the Log argument, returns self.
//  log := Log{"key": "foo"}
//  logging.Info(log.From(Log{"key2": "foo2"}))
func (l Log) From(log Log) Log {
	for key, val := range log {
		l[key] = val
	}
	return l
}

// Into copy self values into the Log argument, returns the Log argument.
//  redisLog := Log{"kind": "redis"}
//  logging.Err(redisLog.Into(Log{"data": "foo"}))
func (l Log) Into(log Log) Log {
	for key, val := range l {
		log[key] = val
	}
	return log
}

// Reset delete all key-value on the log. Empty log will not be consumed.
//
//  log := logger.FromCtx(ctx)
//  if ctx.Path == "/" {
//  	log.Reset() // reset log, don't logging for path "/"
//  } else {
//  	log["Data"] = someData
//  }
//
func (l Log) Reset() {
	for key := range l {
		delete(l, key)
	}
}

// Level represents logging level
// https://tools.ietf.org/html/rfc5424
// https://en.wikipedia.org/wiki/Syslog
type Level uint8

const (
	// EmergLevel is 0, "Emergency", system is unusable
	EmergLevel Level = iota
	// AlertLevel is 1, "Alert", action must be taken immediately
	AlertLevel
	// CritiLevel is 2, "Critical", critical conditions
	CritiLevel
	// ErrLevel is 3, "Error", error conditions
	ErrLevel
	// WarningLevel is 4, "Warning", warning conditions
	WarningLevel
	// NoticeLevel is 5, "Notice", normal but significant condition
	NoticeLevel
	// InfoLevel is 6, "Informational", informational messages
	InfoLevel
	// DebugLevel is 7, "Debug", debug-level messages
	DebugLevel
)

var levels = []string{"EMERG", "ALERT", "CRIT", "ERR", "WARNING", "NOTICE", "INFO", "DEBUG"}
var std = New(os.Stderr)

// Default returns the default logger
// If devMode is true, logger will print a simple version of Common Log Format with terminal color
func Default(devMode ...bool) *Logger {
	if len(devMode) > 0 && devMode[0] {
		std.SetLogConsume(developmentConsume)
	}
	return std
}

// a simple version of Common Log Format with terminal color
// https://en.wikipedia.org/wiki/Common_Log_Format
//
//  127.0.0.1 - - [2017-06-01T12:23:13.161Z] "GET /context.go?query=xxx HTTP/1.1" 200 21559 5.228ms
//
func developmentConsume(log Log, ctx *gear.Context) {
	std.mu.Lock() // don't need Lock usually, logger.Output do it for us.
	defer std.mu.Unlock()

	end := time.Now().UTC()
	FprintWithColor(std.Out, fmt.Sprintf("%s", log["IP"]), ColorGreen)
	fmt.Fprintf(std.Out, ` - - [%s] "%s %s %s" `, end.Format(std.tf), log["Method"], log["URL"], log["Proto"])
	status := log["Status"].(int)
	FprintWithColor(std.Out, strconv.Itoa(status), colorStatus(status))
	resTime := float64(end.Sub(log["Start"].(time.Time))) / 1e6
	fmt.Fprintln(std.Out, fmt.Sprintf(" %d %.3fms", log["Length"], resTime))
}

// New creates a Logger instance with given io.Writer and DebugLevel log level.
// the logger timestamp format is "2006-01-02T15:04:05.999Z"(JavaScript ISO date string), log format is "[%s] %s %s"
func New(w io.Writer) *Logger {
	logger := &Logger{Out: w}
	logger.SetLevel(DebugLevel)
	logger.SetTimeFormat("2006-01-02T15:04:05.999Z")
	logger.SetLogFormat("[%s] %s %s")

	logger.init = func(log Log, ctx *gear.Context) {
		log["IP"] = ctx.IP()
		log["Method"] = ctx.Method
		log["URL"] = ctx.Req.URL.String()
		log["Proto"] = ctx.Req.Proto
		log["UserAgent"] = ctx.GetHeader(gear.HeaderUserAgent)
		log["Start"] = time.Now()
	}

	logger.consume = func(log Log, _ *gear.Context) {
		end := time.Now()
		if t, ok := log["Start"].(time.Time); ok {
			log["Time"] = end.Sub(t) / 1e6 // ms
			delete(log, "Start")
		}

		if str, err := log.Format(); err == nil {
			logger.Output(end, InfoLevel, str)
		} else {
			logger.Output(end, WarningLevel, log.String())
		}
	}
	return logger
}

// A Logger represents an active logging object that generates lines of
// output to an io.Writer. Each logging operation makes a single call to
// the Writer's Write method. A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the Writer.
//
// A custom logger example:
//
//  app := gear.New()
//
//  logger := logging.New(os.Stdout)
//  logger.SetLevel(logging.InfoLevel)
//  logger.SetLogInit(func(log logging.Log, ctx *gear.Context) {
//  	log["IP"] = ctx.IP()
//  	log["Method"] = ctx.Method
//  	log["URL"] = ctx.Req.URL.String()
//  	log["Start"] = time.Now()
//  	log["UserAgent"] = ctx.GetHeader(gear.HeaderUserAgent)
//  })
//  logger.SetLogConsume(func(log logging.Log, _ *gear.Context) {
//  	end := time.Now()
//  	if str, err := log.Format(); err == nil {
//  		logger.Output(end, logging.InfoLevel, str)
//  	} else {
//  		logger.Output(end, logging.WarningLevel, log.String())
//  	}
//  })
//
//  app.UseHandler(logger)
//  app.Use(func(ctx *gear.Context) error {
//  	log := logger.FromCtx(ctx)
//  	log["Data"] = []int{1, 2, 3}
//  	return ctx.HTML(200, "OK")
//  })
//
type Logger struct {
	// Destination for output, It's common to set this to a
	// file, or `os.Stderr`. You can also set this to
	// something more adventorous, such as logging to Kafka.
	Out     io.Writer
	l       Level                    // logging level
	tf, lf  string                   // time format, log format
	mu      sync.Mutex               // ensures atomic writes; protects the following fields
	init    func(Log, *gear.Context) // hook to initialize log with gear.Context
	consume func(Log, *gear.Context) // hook to consume log
}

// Check log output level statisfy output level or not, used internal, for performance
func (l *Logger) checkLogLevel(level Level) bool {
	// don't satisfy logger level, so skip
	return level <= l.l
}

// Emerg produce a "Emergency" log
func (l *Logger) Emerg(v interface{}) {
	l.Output(time.Now(), EmergLevel, format(v))
}

// Alert produce a "Alert" log
func (l *Logger) Alert(v interface{}) {
	if l.checkLogLevel(AlertLevel) {
		l.Output(time.Now(), AlertLevel, format(v))
	}
}

// Crit produce a "Critical" log
func (l *Logger) Crit(v interface{}) {
	if l.checkLogLevel(CritiLevel) {
		l.Output(time.Now(), CritiLevel, format(v))
	}
}

// Err produce a "Error" log
func (l *Logger) Err(v interface{}) {
	if l.checkLogLevel(ErrLevel) {
		l.Output(time.Now(), ErrLevel, format(v))
	}
}

// Warning produce a "Warning" log
func (l *Logger) Warning(v interface{}) {
	if l.checkLogLevel(WarningLevel) {
		l.Output(time.Now(), WarningLevel, format(v))
	}
}

// Notice produce a "Notice" log
func (l *Logger) Notice(v interface{}) {
	if l.checkLogLevel(NoticeLevel) {
		l.Output(time.Now(), NoticeLevel, format(v))
	}
}

// Info produce a "Informational" log
func (l *Logger) Info(v interface{}) {
	if l.checkLogLevel(InfoLevel) {
		l.Output(time.Now(), InfoLevel, format(v))
	}
}

// Debug produce a "Debug" log
func (l *Logger) Debug(v interface{}) {
	if l.checkLogLevel(DebugLevel) {
		l.Output(time.Now(), DebugLevel, format(v))
	}
}

// Debugf produce a "Debug" log in the manner of fmt.Printf
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.checkLogLevel(DebugLevel) {
		l.Output(time.Now(), DebugLevel, fmt.Sprintf(format, args...))
	}
}

// Panic produce a "Emergency" log and then calls panic with the message
func (l *Logger) Panic(v interface{}) {
	s := format(v)
	l.Emerg(s)
	panic(s)
}

var exit = func() { os.Exit(1) }

// Fatal produce a "Emergency" log and then calls os.Exit(1)
func (l *Logger) Fatal(v interface{}) {
	l.Emerg(v)
	exit()
}

// Print produce a log in the manner of fmt.Print, without timestamp and log level
func (l *Logger) Print(args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprint(l.Out, args...)
}

// Printf produce a log in the manner of fmt.Printf, without timestamp and log level
func (l *Logger) Printf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.Out, format, args...)
}

// Println produce a log in the manner of fmt.Println, without timestamp and log level
func (l *Logger) Println(args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.Out, args...)
}

// Output writes a string log with timestamp and log level to the output.
// If the level is greater than logger level, the log will be omitted.
// The log will be format by timeFormat and logFormat.
func (l *Logger) Output(t time.Time, level Level, s string) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// log level checked before
	if level < 4 {
		s = gear.ErrorWithStack(s, 4).String()
	}
	if l := len(s); l > 0 && s[l-1] == '\n' {
		s = s[0 : l-1]
	}
	_, err = fmt.Fprintf(l.Out, l.lf, t.UTC().Format(l.tf), levels[level], crlfEscaper.Replace(s))
	if err == nil {
		l.Out.Write([]byte{'\n'})
	}
	return
}

// SetLevel set the logger's log level
// The default logger level is DebugLevel
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level > DebugLevel {
		panic(gear.Err.WithMsg("invalid logger level"))
	}
	l.l = level
}

// SetTimeFormat set the logger timestamp format
// The default logger timestamp format is "2006-01-02T15:04:05.999Z"(JavaScript ISO date string)
func (l *Logger) SetTimeFormat(timeFormat string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tf = timeFormat
}

// SetLogFormat set the logger log format
// it should accept 3 string values: timestamp, log level and log message
// The default logger log format is "[%s] %s %s": "[time] logLevel message"
func (l *Logger) SetLogFormat(logFormat string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lf = logFormat
}

// SetLogInit set a log init handle to the logger.
// It will be called when log created.
func (l *Logger) SetLogInit(fn func(Log, *gear.Context)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.init = fn
}

// SetLogConsume set a log consumer handle to the logger.
// It will be called on a "end hook" and should write the log to underlayer logging system.
// The default implements is for development, the output log format:
//
//   127.0.0.1 GET /text 200 6500 - 0.765 ms
//
// Please implements a Log Consume for your production.
func (l *Logger) SetLogConsume(fn func(Log, *gear.Context)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.consume = fn
}

// New implements gear.Any interface,then we can use ctx.Any to retrieve a Log instance from ctx.
// Here also some initialization work after created.
func (l *Logger) New(ctx *gear.Context) (interface{}, error) {
	log := Log{}
	l.init(log, ctx)
	return log, nil
}

// FromCtx retrieve the Log instance from the ctx with ctx.Any.
// Logger.New and ctx.Any will guarantee it exists.
func (l *Logger) FromCtx(ctx *gear.Context) Log {
	any, _ := ctx.Any(l)
	return any.(Log)
}

// Serve implements gear.Handler interface, we can use logger as gear middleware.
//
//  app := gear.New()
//  app.UseHandler(logging.Default())
//  app.Use(func(ctx *gear.Context) error {
//  	log := logging.FromCtx(ctx)
//  	log["Data"] = []int{1, 2, 3}
//  	return ctx.HTML(200, "OK")
//  })
//
func (l *Logger) Serve(ctx *gear.Context) error {
	log := l.FromCtx(ctx)
	// Add a "end hook" to flush logs
	ctx.OnEnd(func() {
		// Ignore empty log
		if len(log) == 0 {
			return
		}
		log["Status"] = ctx.Res.Status()
		log["Length"] = len(ctx.Res.Body())
		l.consume(log, ctx)
	})
	return nil
}

// Emerg produce a "Emergency" log with the default logger
func Emerg(v interface{}) {
	std.Emerg(v)
}

// Alert produce a "Alert" log with the default logger
func Alert(v interface{}) {
	std.Alert(v)
}

// Crit produce a "Critical" log with the default logger
func Crit(v interface{}) {
	std.Crit(v)
}

// Err produce a "Error" log with the default logger
func Err(v interface{}) {
	std.Err(v)
}

// Warning produce a "Warning" log with the default logger
func Warning(v interface{}) {
	std.Warning(v)
}

// Notice produce a "Notice" log with the default logger
func Notice(v interface{}) {
	std.Notice(v)
}

// Info produce a "Informational" log with the default logger
func Info(v interface{}) {
	std.Info(v)
}

// Debug produce a "Debug" log with the default logger
func Debug(v interface{}) {
	std.Debug(v)
}

// Debugf produce a "Debug" log in the manner of fmt.Printf with the default logger
func Debugf(format string, args ...interface{}) {
	std.Debugf(format, args...)
}

// Panic produce a "Emergency" log with the default logger and then calls panic with the message
func Panic(v interface{}) {
	std.Panic(v)
}

// Fatal produce a "Emergency" log with the default logger and then calls os.Exit(1)
func Fatal(v interface{}) {
	std.Fatal(v)
}

// Print produce a log in the manner of fmt.Print with the default logger,
// without timestamp and log level
func Print(args ...interface{}) {
	std.Print(args...)
}

// Printf produce a log in the manner of fmt.Printf with the default logger,
// without timestamp and log level
func Printf(format string, args ...interface{}) {
	std.Printf(format, args...)
}

// Println produce a log in the manner of fmt.Println with the default logger,
// without timestamp and log level
func Println(args ...interface{}) {
	std.Println(args...)
}

// FromCtx retrieve the Log instance for the default logger.
func FromCtx(ctx *gear.Context) Log {
	return std.FromCtx(ctx)
}

func colorStatus(code int) ColorType {
	switch {
	case code < 300:
		return ColorGreen
	case code >= 300 && code < 400:
		return ColorCyan
	case code >= 400 && code < 500:
		return ColorYellow
	default:
		return ColorRed
	}
}

func format(i interface{}) string {
	switch v := i.(type) {
	case Messager:
		if str, err := v.Format(); err == nil {
			return str
		}
		return v.String()
	default:
		return fmt.Sprint(i)
	}
}
