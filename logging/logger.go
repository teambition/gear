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

// KV set key/value to the log, returns self.
//  log := Log{}
//  logging.Info(log.KV("key1", "foo").KV("key2", 123))
func (l Log) KV(key string, value interface{}) Log {
	l[key] = value
	return l
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

// With copy values from the argument, returns new log.
//  log := Log{"key": "foo"}
//  logging.Info(log.With(Log{"key2": "foo2"}))
func (l Log) With(log map[string]interface{}) Log {
	cp := l.Into(Log{})
	for key, val := range log {
		cp[key] = val
	}
	return cp
}

// Reset delete all key-value on the log. Empty log will not be consumed.
//
//  log := logger.FromCtx(ctx)
//  if ctx.Path == "/" {
//  	log.Reset() // reset log, don't logging for path "/"
//  } else {
//  	log["data"] = someData
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
	// CritLevel is 2, "Critical", critical conditions
	CritLevel
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

// CritiLevel is an alias of CritLevel
const CritiLevel = CritLevel

// https://en.wikipedia.org/wiki/Syslog
func (l Level) String() string {
	switch l {
	case EmergLevel:
		return "emerg"
	case AlertLevel:
		return "alert"
	case CritLevel:
		return "crit"
	case ErrLevel:
		return "err"
	case WarningLevel:
		return "warning"
	case NoticeLevel:
		return "notice"
	case InfoLevel:
		return "info"
	case DebugLevel:
		return "debug"
	default:
		return "log"
	}
}

// ParseLevel takes a string level and returns the gear logging level constant.
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "emergency", "emerg":
		return EmergLevel, nil
	case "alert":
		return AlertLevel, nil
	case "critical", "crit", "criti":
		return CritLevel, nil
	case "error", "err":
		return ErrLevel, nil
	case "warning", "warn":
		return WarningLevel, nil
	case "notice":
		return NoticeLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	}

	var l Level
	return l, fmt.Errorf("not a valid gear logging Level: %q", lvl)
}

// SetLoggerLevel set a string level to the logger.
func SetLoggerLevel(logger *Logger, lvl string) error {
	level, err := ParseLevel(lvl)
	if err == nil {
		logger.SetLevel(level)
	}
	return err
}

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
	FprintWithColor(std.Out, fmt.Sprintf("%s", log["ip"]), ColorGreen)
	fmt.Fprintf(std.Out, ` - - [%s] "%s %s %s" `, end.Format(std.tf), log["method"], log["uri"], log["proto"])
	status := log["status"].(int)
	FprintWithColor(std.Out, strconv.Itoa(status), colorStatus(status))
	resTime := float64(end.Sub(ctx.StartAt)) / 1e6
	fmt.Fprintln(std.Out, fmt.Sprintf(" %d %.3fms", log["length"], resTime))
}

// New creates a Logger instance with given io.Writer and DebugLevel log level.
// the logger timestamp format is "2006-01-02T15:04:05.000Z"(JavaScript ISO date string), log format is "[%s] %s %s"
func New(w io.Writer) *Logger {
	logger := &Logger{Out: w}
	logger.SetLevel(DebugLevel)
	logger.SetTimeFormat("2006-01-02T15:04:05.000Z")
	logger.SetLogFormat("[%s] %s %s")

	logger.init = func(log Log, ctx *gear.Context) {
		log["start"] = ctx.StartAt.Format(logger.tf)
		log["ip"] = ctx.IP().String()
		log["scheme"] = ctx.Scheme()
		log["proto"] = ctx.Req.Proto
		log["method"] = ctx.Method
		log["uri"] = ctx.Req.RequestURI
		if s := ctx.GetHeader(gear.HeaderUpgrade); s != "" {
			log["upgrade"] = s
		}
		if s := ctx.GetHeader(gear.HeaderOrigin); s != "" {
			log["origin"] = s
		}
		if s := ctx.GetHeader(gear.HeaderReferer); s != "" {
			log["referer"] = s
		}
		if s := ctx.GetHeader(gear.HeaderXCanary); s != "" {
			log["xCanary"] = s
		}
		log["userAgent"] = ctx.GetHeader(gear.HeaderUserAgent)
	}

	logger.consume = func(log Log, ctx *gear.Context) {
		end := time.Now().UTC()
		log["duration"] = end.Sub(ctx.StartAt) / 1e6 // ms

		if s := ctx.GetHeader(gear.HeaderXRequestID); s != "" {
			log["xRequestId"] = s
		} else if s := ctx.Res.Get(gear.HeaderXRequestID); s != "" {
			log["xRequestId"] = s
		}

		if router := gear.GetRouterPatternFromCtx(ctx); router != "" {
			log["router"] = fmt.Sprintf("%s %s", ctx.Method, router)
		}

		if err := logger.output(end, InfoLevel, log); err != nil {
			logger.output(end, ErrLevel, err)
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
//    log["ip"] = ctx.IP().String()
//    log["method"] = ctx.Method
//    log["uri"] = ctx.Req.RequestURI
//    log["proto"] = ctx.Req.Proto
//    log["userAgent"] = ctx.GetHeader(gear.HeaderUserAgent)
//    log["start"] = ctx.StartAt.Format("2006-01-02T15:04:05.000Z")
//    if s := ctx.GetHeader(gear.HeaderOrigin); s != "" {
//    	log["origin"] = s
//    }
//    if s := ctx.GetHeader(gear.HeaderReferer); s != "" {
//    	log["referer"] = s
//    }
//  })
//  logger.SetLogConsume(func(log logging.Log, _ *gear.Context) {
//  	end := time.Now().UTC()
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
//  	log["data"] = []int{1, 2, 3}
//  	return ctx.HTML(200, "OK")
//  })
//
type Logger struct {
	// Destination for output, It's common to set this to a
	// file, or `os.Stderr`. You can also set this to
	// something more adventorous, such as logging to Kafka.
	Out     io.Writer
	json    bool
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
	l.output(time.Now().UTC(), EmergLevel, v)
}

// Alert produce a "Alert" log
func (l *Logger) Alert(v interface{}) {
	if l.checkLogLevel(AlertLevel) {
		l.output(time.Now().UTC(), AlertLevel, v)
	}
}

// Crit produce a "Critical" log
func (l *Logger) Crit(v interface{}) {
	if l.checkLogLevel(CritLevel) {
		l.output(time.Now().UTC(), CritLevel, v)
	}
}

// Err produce a "Error" log
func (l *Logger) Err(v interface{}) {
	if l.checkLogLevel(ErrLevel) {
		l.output(time.Now().UTC(), ErrLevel, v)
	}
}

// Warning produce a "Warning" log
func (l *Logger) Warning(v interface{}) {
	if l.checkLogLevel(WarningLevel) {
		l.output(time.Now().UTC(), WarningLevel, v)
	}
}

// Notice produce a "Notice" log
func (l *Logger) Notice(v interface{}) {
	if l.checkLogLevel(NoticeLevel) {
		l.output(time.Now().UTC(), NoticeLevel, v)
	}
}

// Info produce a "Informational" log
func (l *Logger) Info(v interface{}) {
	if l.checkLogLevel(InfoLevel) {
		l.output(time.Now().UTC(), InfoLevel, v)
	}
}

// Debug produce a "Debug" log
func (l *Logger) Debug(v interface{}) {
	if l.checkLogLevel(DebugLevel) {
		l.output(time.Now().UTC(), DebugLevel, v)
	}
}

// Debugf produce a "Debug" log in the manner of fmt.Printf
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.checkLogLevel(DebugLevel) {
		l.output(time.Now().UTC(), DebugLevel, fmt.Sprintf(format, args...))
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

func (l *Logger) output(t time.Time, level Level, v interface{}) (err error) {
	if l.json {
		var log Log
		if level > ErrLevel {
			log = format2Log(v)
		} else {
			log = formatError2Log(v)
		}
		log["time"] = t.Format(l.tf)
		log["level"] = level.String()
		return l.OutputJSON(log)
	}

	var s string
	if level > ErrLevel {
		s = format(v)
	} else {
		s = formatError(v)
	}
	return l.Output(t, level, s)
}

// Output writes a string log with timestamp and log level to the output.
// The log will be format by timeFormat and logFormat.
func (l *Logger) Output(t time.Time, level Level, s string) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l := len(s); l > 0 && s[l-1] == '\n' {
		s = s[0 : l-1]
	}
	_, err = fmt.Fprintf(l.Out, l.lf, t.UTC().Format(l.tf), level.String(), crlfEscaper.Replace(s))
	if err == nil {
		l.Out.Write([]byte{'\n'})
	}
	return
}

// OutputJSON writes a Log log as JSON string to the output.
func (l *Logger) OutputJSON(log Log) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var str string
	if str, err = log.Format(); err == nil {
		_, err = fmt.Fprint(l.Out, crlfEscaper.Replace(str))
		if err == nil {
			l.Out.Write([]byte{'\n'})
		}
	}
	return
}

// GetLevel get the logger's log level
func (l *Logger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.l
}

// SetLevel set the logger's log level
// The default logger level is DebugLevel
func (l *Logger) SetLevel(level Level) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level > DebugLevel {
		panic(gear.Err.WithMsg("invalid logger level"))
	}
	l.l = level
	return l
}

// SetJSONLog set the logger writing JSON string log.
// It will become default in Gear@v2.
func (l *Logger) SetJSONLog() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.json = true
	return l
}

// SetTimeFormat set the logger timestamp format
// The default logger timestamp format is "2006-01-02T15:04:05.000Z"(JavaScript ISO date string)
func (l *Logger) SetTimeFormat(timeFormat string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tf = timeFormat
	return l
}

// SetLogFormat set the logger log format
// it should accept 3 string values: timestamp, log level and log message
// The default logger log format is "[%s] %s %s": "[time] logLevel message"
func (l *Logger) SetLogFormat(logFormat string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lf = logFormat
	return l
}

// SetLogInit set a log init handle to the logger.
// It will be called when log created.
func (l *Logger) SetLogInit(fn func(Log, *gear.Context)) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.init = fn
	return l
}

// SetLogConsume set a log consumer handle to the logger.
// It will be called on a "end hook" and should write the log to underlayer logging system.
// The default implements is for development, the output log format:
//
//   127.0.0.1 GET /text 200 6500 - 0.765 ms
//
// Please implements a Log Consume for your production.
func (l *Logger) SetLogConsume(fn func(Log, *gear.Context)) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.consume = fn
	return l
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

// SetTo sets key/value to the Log instance on ctx.
//  app.Use(func(ctx *gear.Context) error {
//  	logging.SetTo(ctx, "Data", []int{1, 2, 3})
//  	return ctx.HTML(200, "OK")
//  })
func (l *Logger) SetTo(ctx *gear.Context, key string, val interface{}) {
	any, _ := ctx.Any(l)
	any.(Log)[key] = val
}

// Serve implements gear.Handler interface, we can use logger as gear middleware.
//
//  app := gear.New()
//  app.UseHandler(logging.Default())
//  app.Use(func(ctx *gear.Context) error {
//  	log := logging.FromCtx(ctx)
//  	log["data"] = []int{1, 2, 3}
//  	return ctx.HTML(200, "OK")
//  })
//
func (l *Logger) Serve(ctx *gear.Context) error {
	// should be inited when start
	log := l.FromCtx(ctx)
	// Add a "end hook" to flush logs
	ctx.OnEnd(func() {
		// Ignore empty log
		if len(log) == 0 {
			return
		}
		log["status"] = ctx.Res.Status()
		log["length"] = len(ctx.Res.Body())

		if ctx.Res.Status() == 500 {
			if body, _ := ctx.Any("GEAR_REQUEST_BODY"); body != nil {
				if b, ok := body.([]byte); ok {
					log["requestBody"] = string(b)
					if contentType, _ := ctx.Any("GEAR_REQUEST_CONTENT_TYPE"); contentType != nil {
						log["requestContentType"] = contentType
					}
				}
			}

			if b := ctx.Res.Body(); b != nil {
				log["responseBody"] = string(b)
				log["responseContentType"] = ctx.Res.Get(gear.HeaderContentType)
			}
		}

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

// SetTo sets key/value to the Log instance on ctx for the default logger.
//  app.UseHandler(logging.Default())
//  app.Use(func(ctx *gear.Context) error {
//  	logging.SetTo(ctx, "Data", []int{1, 2, 3})
//  	return ctx.HTML(200, "OK")
//  })
func SetTo(ctx *gear.Context, key string, val interface{}) {
	std.SetTo(ctx, key, val)
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

func formatError(i interface{}) string {
	err := gear.ErrorWithStack(i)
	if err == nil {
		return ""
	}
	if str, e := err.Format(); e == nil {
		return str
	}
	return err.String()
}

func formatError2Log(i interface{}) Log {
	err := gear.ErrorWithStack(i)
	if err == nil {
		return Log{}
	}
	return Log{
		"code":    err.Code,
		"error":   err.Err,
		"message": err.Msg,
		"data":    err.Data,
		"stack":   err.Stack,
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

func format2Log(i interface{}) Log {
	switch v := i.(type) {
	case Log:
		return v
	case map[string]interface{}:
		return Log(v)
	default:
		return Log{"message": format(i)}
	}
}

// func WithLogger()
// func LogFromCtx()
// func LoggerFromCtx()
// func AddLogToCtx()
