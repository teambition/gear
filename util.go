package gear

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode/utf8"
)

type middlewares []Middleware

func (m middlewares) run(ctx *Context) (err error) {
	for _, fn := range m {
		if err = fn(ctx); !IsNil(err) || ctx.Res.ended.isTrue() {
			return
		}
	}
	return
}

// Compose composes a array of middlewares to one middleware
func Compose(mds ...Middleware) Middleware {
	switch len(mds) {
	case 0:
		return noOp
	case 1:
		return mds[0]
	default:
		return middlewares(mds).run
	}
}

var noOp Middleware = func(ctx *Context) error { return nil }

// WrapHandler wrap a http.Handler to Gear Middleware
func WrapHandler(handler http.Handler) Middleware {
	return func(ctx *Context) error {
		handler.ServeHTTP(ctx.Res, ctx.Req)
		return nil
	}
}

// WrapHandlerFunc wrap a http.HandlerFunc to Gear Middleware
func WrapHandlerFunc(fn http.HandlerFunc) Middleware {
	return func(ctx *Context) error {
		fn(ctx.Res, ctx.Req)
		return nil
	}
}

// IsNil checks if a specified object is nil or not, without Failing.
func IsNil(val interface{}) bool {
	if val == nil {
		return true
	}

	value := reflect.ValueOf(val)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// Error represents a numeric error with optional meta. It can be used in middleware as a return result.
type Error struct {
	Code  int         `json:"-"`
	Err   string      `json:"error"`
	Msg   string      `json:"message"`
	Data  interface{} `json:"data,omitempty"`
	Stack string      `json:"-"`
}

// Status implemented HTTPError interface.
func (err *Error) Status() int {
	return err.Code
}

// Error implemented HTTPError interface.
func (err *Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Err, err.Msg)
}

// String implemented fmt.Stringer interface.
func (err Error) String() string {
	return err.GoString()
}

// GoString implemented fmt.GoStringer interface, returns a Go-syntax string.
func (err Error) GoString() string {
	if v, ok := err.Data.([]byte); ok && utf8.Valid(v) {
		err.Data = string(v)
	}
	return fmt.Sprintf(`Error{Code:%d, Err:"%s", Msg:"%s", Data:%#v, Stack:"%s"}`,
		err.Code, err.Err, err.Msg, err.Data, err.Stack)
}

// WithErr returns a copy of err with given new error name.
//  err := gear.ErrBadRequest.WithErr("InvalidEmail") // 400 Bad Request error with error name InvalidEmail"
func (err Error) WithErr(name string) *Error {
	err.Err = name
	return &err
}

// WithMsg returns a copy of err with given new messages.
//  err := gear.Err.WithMsg() // just clone
//  err := gear.ErrBadRequest.WithMsg("invalid email") // 400 Bad Request error with message invalid email"
func (err Error) WithMsg(msgs ...string) *Error {
	if len(msgs) > 0 {
		err.Msg = strings.Join(msgs, ", ")
	}
	return &err
}

// WithMsgf returns a copy of err with given message in the manner of fmt.Printf.
//  err := gear.ErrBadRequest.WithMsgf(`invalid email: "%s"`, email)
func (err Error) WithMsgf(format string, args ...interface{}) *Error {
	return err.WithMsg(fmt.Sprintf(format, args...))
}

// WithCode returns a copy of err with given code.
//  BadRequestErr := gear.Err.WithCode(400)
func (err Error) WithCode(code int) *Error {
	err.Code = code
	if text := http.StatusText(code); text != "" {
		err.Err = text
	}
	return &err
}

// WithStack returns a copy of err with error stack.
//  err := gear.Err.WithMsg("some error").WithStack()
func (err Error) WithStack(skip ...int) *Error {
	return ErrorWithStack(&err, skip...)
}

// From returns a copy of err with given error. It will try to merge the given error.
// If the given error is a *Error instance, it will be returned without copy.
//  err := gear.ErrBadRequest.From(errors.New("invalid email"))
//  err := gear.Err.From(someErr)
func (err Error) From(e error) *Error {
	if IsNil(e) {
		return nil
	}

	switch v := e.(type) {
	case *Error:
		return v
	case HTTPError:
		err.Code = v.Status()
		err.Msg = v.Error()
	case *textproto.Error:
		err.Code = v.Code
		err.Msg = v.Msg
	default:
		err.Msg = e.Error()
	}

	if err.Err == "" {
		err.Err = http.StatusText(err.Code)
	}
	return &err
}

// ParseError parse a error, textproto.Error or HTTPError to HTTPError
func ParseError(e error, code ...int) HTTPError {
	if IsNil(e) {
		return nil
	}

	switch v := e.(type) {
	case HTTPError:
		return v
	case *textproto.Error:
		err := Err.WithCode(v.Code)
		err.Msg = v.Msg
		return err
	default:
		err := ErrInternalServerError.WithMsg(e.Error())
		if len(code) > 0 && code[0] > 0 {
			err = err.WithCode(code[0])
		}
		return err
	}
}

// ErrorWithStack create a error with stacktrace
func ErrorWithStack(val interface{}, skip ...int) *Error {
	if IsNil(val) {
		return nil
	}

	var err *Error
	switch v := val.(type) {
	case *Error:
		err = v.WithMsg() // must clone, should not change the origin *Error instance
	case error:
		err = ErrInternalServerError.From(v)
	case string:
		err = ErrInternalServerError.WithMsg(v)
	default:
		err = ErrInternalServerError.WithMsgf("%#v", v)
	}

	if err.Stack == "" {
		buf := make([]byte, 2048)
		buf = buf[:runtime.Stack(buf, false)]
		s := 1
		if len(skip) != 0 {
			s = skip[0]
		}
		err.Stack = pruneStack(buf, s)
	}
	return err
}

// ValuesToStruct converts url.Values into struct object. It supports specific types that implementing encoding.TextUnmarshaler interface.
//
//  type jsonQueryTemplate struct {
//  	ID   string `json:"id" form:"id"`
//  	Pass string `json:"pass" form:"pass"`
//  }
//
//  target := jsonQueryTemplate{}
//
//  gear.ValuesToStruct(map[string][]string{
//  	"id": []string{"some id"},
//  	"pass": []string{"some pass"},
//  }, &target, "form")
//
func ValuesToStruct(values map[string][]string, target interface{}, tag string) (err error) {
	if values == nil {
		return fmt.Errorf("invalid values: %v", values)
	}
	if len(values) == 0 {
		return
	}
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("invalid struct: %v", rv)
	}

	rv = rv.Elem()
	rt := rv.Type()
	n := rv.NumField()
	for i := 0; i < n; i++ {
		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}

		fk := rt.Field(i).Tag.Get(tag)
		if fk == "" {
			continue
		}

		if vals, ok := values[fk]; ok {
			if fv.Kind() == reflect.Slice {
				err = setRefSlice(fv, vals)
			} else if len(vals) > 0 {
				err = setRefField(fv, vals[0])
			}
			if err != nil {
				return
			}
		}
	}

	return
}

func shouldDeref(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool, reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func setRefSlice(v reflect.Value, vals []string) error {
	l := len(vals)
	slice := reflect.MakeSlice(v.Type(), l, l)

	for i := 0; i < l; i++ {
		if err := setRefField(slice.Index(i), vals[i]); err != nil {
			return err
		}
	}

	v.Set(slice)
	return nil
}

func setRefField(v reflect.Value, str string) error {
	if ok, err := tryUnmarshalValue(v, str); ok {
		return err
	}

	if v.Kind() == reflect.Ptr && shouldDeref(v.Type().Elem().Kind()) {
		v.Set(reflect.New(v.Type().Elem()))
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(str)
		return nil
	case reflect.Bool:
		return setRefBool(v, str)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setRefInt(v, str, v.Type().Bits())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setRefUint(v, str, v.Type().Bits())
	case reflect.Float32, reflect.Float64:
		return setRefFloat(v, str, v.Type().Bits())
	default:
		return fmt.Errorf("unknown field type: %v", v.Type())
	}
}

func setRefBool(v reflect.Value, str string) error {
	val, err := strconv.ParseBool(str)
	if err == nil {
		v.SetBool(val)
	}
	return err
}

func setRefInt(v reflect.Value, str string, size int) error {
	val, err := strconv.ParseInt(str, 10, size)
	if err == nil {
		v.SetInt(val)
	}
	return err
}

func setRefUint(v reflect.Value, str string, size int) error {
	val, err := strconv.ParseUint(str, 10, size)
	if err == nil {
		v.SetUint(val)
	}
	return err
}

func setRefFloat(v reflect.Value, str string, size int) error {
	val, err := strconv.ParseFloat(str, size)
	if err == nil {
		v.SetFloat(val)
	}
	return err
}

func tryUnmarshalValue(v reflect.Value, str string) (bool, error) {
	if v.Kind() != reflect.Ptr && v.CanAddr() && v.Type().Name() != "" {
		v = v.Addr()
	}

	if v.Type().NumMethod() > 0 {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
			return true, u.UnmarshalText([]byte(str))
		}
	}
	return false, nil
}

// pruneStack make a thin conversion for stack information
// limit the count of lines to 5
// src:
// ```
// goroutine 9 [running]:
// runtime/debug.Stack(0x6, 0x6, 0xc42003c898)
//     /usr/local/Cellar/go/1.7.4_2/libexec/src/runtime/debug/stack.go:24 +0x79
// github.com/teambition/gear/logging.(*Logger).OutputWithStack(0xc420012a50, 0xed0092215, 0x573fdbb, 0x471f20, 0x0, 0xc42000dc1a, 0x6, 0xc42000dc01, 0xc42000dca0)
//     /Users/xus/go/src/github.com/teambition/gear/logging/logger.go:267 +0x4e
// github.com/teambition/gear/logging.(*Logger).Emerg(0xc420012a50, 0x2a9cc0, 0xc42000dca0)
//     /Users/xus/go/src/github.com/teambition/gear/logging/logger.go:171 +0xd3
// github.com/teambition/gear/logging.TestGearLogger.func2(0xc420018600)
//     /Users/xus/go/src/github.com/teambition/gear/logging/logger_test.go:90 +0x3c1
// testing.tRunner(0xc420018600, 0x33d240)
//     /usr/local/Cellar/go/1.7.4_2/libexec/src/testing/testing.go:610 +0x81
// created by testing.(*T).Run
//     /usr/local/Cellar/go/1.7.4_2/libexec/src/testing/testing.go:646 +0x2ec
// ```
// dst:
// ```
// Stack:
//     /usr/local/Cellar/go/1.7.4_2/libexec/src/runtime/debug/stack.go:24
//     /Users/xus/go/src/github.com/teambition/gear/logging/logger.go:283
//     /Users/xus/go/src/github.com/teambition/gear/logging/logger.go:171
//     /Users/xus/go/src/github.com/teambition/gear/logging/logger_test.go:90
//     /usr/local/Cellar/go/1.7.4_2/libexec/src/testing/testing.go:610
//     /usr/local/Cellar/go/1.7.4_2/libexec/src/testing/testing.go:646
// ```
func pruneStack(stack []byte, skip int) string {
	// remove first line
	// `goroutine 1 [running]:`
	lines := strings.Split(string(stack), "\n")[1:]
	newLines := make([]string, 0, len(lines)/2)

	num := 0
	for idx, line := range lines {
		if idx%2 == 0 {
			continue
		}
		skip--
		if skip >= 0 {
			continue
		}
		num++

		loc := strings.Split(line, " ")[0]
		loc = strings.Replace(loc, "\t", "\\t", -1)
		// only need odd line
		newLines = append(newLines, loc)
		if num == 10 {
			break
		}
	}
	return strings.Join(newLines, "\\n")
}

type atomicBool int32

func (b *atomicBool) isTrue() bool {
	return atomic.LoadInt32((*int32)(b)) == 1
}

func (b *atomicBool) swapTrue() bool {
	return atomic.SwapInt32((*int32)(b), 1) == 0
}

func (b *atomicBool) setTrue() {
	atomic.StoreInt32((*int32)(b), 1)
}

// IsStatusCode returns true if status is HTTP status code.
// https://en.wikipedia.org/wiki/List_of_HTTP_status_codes
func IsStatusCode(status int) bool {
	switch status {
	case 100, 101, 102,
		200, 201, 202, 203, 204, 205, 206, 207, 208, 226,
		300, 301, 302, 303, 304, 305, 306, 307, 308,
		400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415, 416, 417, 418,
		421, 422, 423, 424, 426, 428, 429, 431, 440, 444, 449, 450, 451, 494, 495, 496, 497, 498, 499,
		500, 501, 502, 503, 504, 505, 506, 507, 508, 509, 510, 511, 520, 521, 522, 523, 524, 525, 526, 527:
		return true
	default:
		return false
	}
}

func isRedirectStatus(status int) bool {
	switch status {
	case 300, 301, 302, 303, 305, 307, 308:
		return true
	default:
		return false
	}
}

func isEmptyStatus(status int) bool {
	switch status {
	case 204, 205, 304:
		return true
	default:
		return false
	}
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", "\"", "\\\"")

// ContentDisposition implements a simple version of https://tools.ietf.org/html/rfc2183
// Use mime.ParseMediaType to parse Content-Disposition header.
func ContentDisposition(fileName, dispositionType string) (header string) {
	if dispositionType == "" {
		dispositionType = "attachment"
	}
	if fileName == "" {
		return dispositionType
	}

	header = fmt.Sprintf(`%s; filename="%s"`, dispositionType, quoteEscaper.Replace(fileName))
	fallbackName := url.PathEscape(fileName)
	if len(fallbackName) != len(fileName) {
		header = fmt.Sprintf(`%s; filename*=UTF-8''%s`, header, fallbackName)
	}
	return
}

// LoggerFilterWriter is a writer for Logger to filter bytes.
// In a https server, avoid some handshake mismatch condition such as loadbalance healthcheck:
//
//  2017/06/09 07:18:04 http: TLS handshake error from 10.10.5.1:45001: tls: first record does not look like a TLS handshake
//  2017/06/14 02:39:29 http: TLS handshake error from 10.0.1.2:54975: read tcp 10.10.5.22:8081->10.0.1.2:54975: read: connection reset by peer
//
// Usage:
//
//  func main() {
//  	app := gear.New() // Create app
//  	app.Use(func(ctx *gear.Context) error {
//  		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//  	})
//
//  	app.Set(gear.SetLogger, log.New(gear.DefaultFilterWriter(), "", log.LstdFlags))
//  	app.Listen(":3000")
//  }
//
type LoggerFilterWriter struct {
	phrases [][]byte
	out     io.Writer
}

var loggerFilterWriter = &LoggerFilterWriter{
	phrases: [][]byte{[]byte("http: TLS handshake error"), []byte("EOF")},
	out:     os.Stderr,
}

// DefaultFilterWriter returns the default LoggerFilterWriter instance.
func DefaultFilterWriter() *LoggerFilterWriter {
	return loggerFilterWriter
}

// SetOutput sets the output destination for the loggerFilterWriter.
func (s *LoggerFilterWriter) SetOutput(out io.Writer) {
	s.out = out
}

// Add add a phrase string to filter
func (s *LoggerFilterWriter) Add(err string) {
	s.phrases = append(s.phrases, []byte(err))
}

func (s *LoggerFilterWriter) Write(p []byte) (n int, err error) {
	for _, phrase := range s.phrases {
		if bytes.Contains(p, phrase) {
			return len(p), nil
		}
	}

	return s.out.Write(p)
}
