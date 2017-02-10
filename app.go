package gear

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// Handler interface is used by app.UseHandler as a middleware.
type Handler interface {
	Serve(ctx *Context) error
}

// Renderer interface is used by ctx.Render.
type Renderer interface {
	Render(ctx *Context, w io.Writer, name string, data interface{}) error
}

// BodyParser interface is used by ctx.ParseBody. Default to:
//  app.Set(gear.SetBodyParser, DefaultBodyParser(1<<20))
//
type BodyParser interface {
	// Maximum allowed size for a request body
	MaxBytes() int64
	Parse(buf []byte, body interface{}, mediaType, charset string) error
}

// DefaultBodyParser is default BodyParser type.
// SetBodyParser used 1MB as default:
//
//  app.Set(gear.SetBodyParser, DefaultBodyParser(1<<20))
//
type DefaultBodyParser int64

// MaxBytes implemented BodyParser interface.
func (d DefaultBodyParser) MaxBytes() int64 {
	return int64(d)
}

// Parse implemented BodyParser interface.
func (d DefaultBodyParser) Parse(buf []byte, body interface{}, mediaType, charset string) error {
	if len(buf) == 0 {
		return &Error{Code: http.StatusBadRequest, Msg: "Request entity empty"}
	}
	switch mediaType {
	case MIMEApplicationJSON:
		return json.Unmarshal(buf, body)
	case MIMEApplicationXML:
		return xml.Unmarshal(buf, body)
	}
	return &Error{Code: http.StatusUnsupportedMediaType, Msg: "Unsupported media type"}
}

// OnError interface is use to deal with errors returned by middlewares. Default to:
//  app.Set("SetOnError", &DefaultOnError{})
//
type OnError interface {
	OnError(ctx *Context, err error) *Error
}

// DefaultOnError is default ctx error handler.
type DefaultOnError struct{}

// OnError implemented OnError interface.
func (o *DefaultOnError) OnError(ctx *Context, err error) *Error {
	code := ctx.Status()
	if code < 400 {
		code = 0
	}
	return ParseError(err, code)
}

// HTTPError interface is used to create a server error that include status code and error message.
type HTTPError interface {
	// Error returns error's message.
	Error() string

	// Status returns error's http status code.
	Status() int
}

// Error represents a numeric error with optional meta. It can be used in middleware as a return result.
type Error struct {
	Code  int
	Msg   string
	Stack string
	Meta  interface{}
}

// ErrorWithStack create a error with stacktrace
func ErrorWithStack(v interface{}, skip ...int) *Error {
	var err *Error
	switch tmp := v.(type) {
	case error:
		err = ParseError(tmp)
	case string:
		err = &Error{500, tmp, "", nil}
	default:
		err = &Error{500, fmt.Sprintf("%#v", tmp), "", nil}
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

// Status implemented HTTPError interface.
func (err *Error) Status() int {
	return err.Code
}

// Error implemented HTTPError interface.
func (err *Error) Error() string {
	return err.Msg
}

// String implemented fmt.Stringer interface, returns a Go-syntax string.
func (err *Error) String() string {
	meta := ""
	if err.Meta != nil {
		switch err.Meta.(type) {
		case []byte:
			meta = string(err.Meta.([]byte))
		}
	}
	return fmt.Sprintf(`Error{Code:%3d, Msg:"%s", Stack:"%s", Meta:%#v}`, err.Code, err.Msg, err.Stack, meta)
}

// Middleware defines a function to process as middleware.
type Middleware func(*Context) error

// NewAppError create a error instance with "Gear: " prefix.
func NewAppError(err string) error {
	return fmt.Errorf("Gear: %s", err)
}

// ParseError parse a error, textproto.Error or HTTPError to *Error
func ParseError(e error, code ...int) *Error {
	var err *Error
	if !IsNil(e) {
		switch e.(type) {
		case *Error:
			err = e.(*Error)
		case *textproto.Error:
			_e := e.(*textproto.Error)
			err = &Error{_e.Code, _e.Msg, "", nil}
		case HTTPError:
			_e := e.(HTTPError)
			err = &Error{_e.Status(), _e.Error(), "", nil}
		default:
			err = &Error{500, e.Error(), "", nil}
			if len(code) > 0 && code[0] > 0 {
				err.Code = code[0]
			}
		}
	}
	return err
}

// App is the top-level framework app instance.
//
// Hello Gear!
//
//  package main
//
//  import "github.com/teambition/gear"
//
//  func main() {
//  	app := gear.New() // Create app
//  	app.Use(func(ctx *gear.Context) error {
//  		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//  	})
//  	app.Error(app.Listen(":3000"))
//  }
//
type App struct {
	Server     *http.Server
	middleware middlewares
	settings   map[interface{}]interface{}

	onerror    OnError
	renderer   Renderer
	bodyParser BodyParser
	keys       []string
	// Default to nil, do not compress response content.
	compress Compressible
	// Default to 0
	timeout time.Duration

	withContext func(context.Context) context.Context
	// ErrorLog specifies an optional logger for app's errors. Default to nil.
	logger *log.Logger
}

// New creates an instance of App.
func New() *App {
	app := new(App)
	app.Server = new(http.Server)
	app.middleware = make(middlewares, 0)
	app.settings = make(map[interface{}]interface{})

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	app.Set(SetEnv, env)
	app.Set(SetOnError, &DefaultOnError{})
	app.Set(SetBodyParser, DefaultBodyParser(1<<20))
	app.Set(SetLogger, log.New(os.Stderr, "", log.LstdFlags))
	return app
}

// Use uses the given middleware `handle`.
func (app *App) Use(handle Middleware) {
	app.middleware = append(app.middleware, handle)
}

// UseHandler uses a instance that implemented Handler interface.
func (app *App) UseHandler(h Handler) {
	app.middleware = append(app.middleware, h.Serve)
}

type appSetting uint8

// Build-in app settings
const (
	// It will be used by `ctx.ParseBody`, value should implements `gear.BodyParser` interface, default to:
	//
	//   app.Set(gear.SetBodyParser, gear.DefaultBodyParser(1<<20))
	//
	SetBodyParser appSetting = iota

	// Enable compress for response, value should implements `gear.Compressible` interface, no default value.
	// Example:
	//  import "github.com/teambition/compressible-go"
	//
	//  app := gear.New()
	//  app.Set(gear.SetCompress, compressible.WithThreshold(1024))
	//
	SetCompress

	// Set secret keys for signed cookies, it will be used by `ctx.Cookies`, value should be `[]string` type,
	// no default value. More document https://github.com/go-http-utils/cookie, Example:
	//
	//  app.Set(gear.SetKeys, []string{"some key2", "some key1"})
	//
	SetKeys

	// Set a logger to app, value should be `*log.Logger` instance, default to:
	//
	//   app.Set(gear.SetLogger, log.New(os.Stderr, "", log.LstdFlags))
	//
	SetLogger

	// Set a on-error hook to app, value should implements `gear.OnError` interface, default to:
	//
	//   app.Set(gear.SetOnError, &gear.DefaultOnError{})
	//
	SetOnError

	// Set a renderer to app, it will be used by `ctx.Render`, value should implements `gear.Renderer` interface,
	// no default value.
	SetRenderer

	// Set a timeout to for the middleware process, value should be `time.Duration`. No default.
	// Example:
	//
	//  app.Set(gear.SetTimeout, 3*time.Second)
	//
	SetTimeout

	// Set a function that Wrap the gear.Context' underlayer context.Context. No default.
	SetWithContext

	// Set a app env string to app, it can be retrieved by `ctx.Setting(gear.SetEnv)`.
	// Default to os process "APP_ENV" or "development".
	SetEnv
)

// Set add key/value settings to app. The settings can be retrieved by `ctx.Setting(key)`.
func (app *App) Set(key, val interface{}) {
	if k, ok := key.(appSetting); ok {
		switch key {
		case SetBodyParser:
			if bodyParser, ok := val.(BodyParser); !ok {
				panic(NewAppError("SetBodyParser setting must implemented gear.BodyParser interface"))
			} else {
				app.bodyParser = bodyParser
			}
		case SetCompress:
			if compress, ok := val.(Compressible); !ok {
				panic(NewAppError("SetCompress setting must implemented gear.Compressible interface"))
			} else {
				app.compress = compress
			}
		case SetKeys:
			if keys, ok := val.([]string); !ok {
				panic(NewAppError("SetKeys setting must be []string"))
			} else {
				app.keys = keys
			}
		case SetLogger:
			if logger, ok := val.(*log.Logger); !ok {
				panic(NewAppError("SetLogger setting must be *log.Logger instance"))
			} else {
				app.logger = logger
			}
		case SetOnError:
			if onerror, ok := val.(OnError); !ok {
				panic(NewAppError("SetOnError setting must implemented gear.OnError interface"))
			} else {
				app.onerror = onerror
			}
		case SetRenderer:
			if renderer, ok := val.(Renderer); !ok {
				panic(NewAppError("SetRenderer setting must implemented gear.Renderer interface"))
			} else {
				app.renderer = renderer
			}
		case SetTimeout:
			if timeout, ok := val.(time.Duration); !ok {
				panic(NewAppError("SetTimeout setting must be time.Duration instance"))
			} else {
				app.timeout = timeout
			}
		case SetWithContext:
			if withContext, ok := val.(func(context.Context) context.Context); !ok {
				panic(NewAppError("SetWithContext setting must be func instance"))
			} else {
				app.withContext = withContext
			}
		case SetEnv:
			if _, ok := val.(string); !ok {
				panic(NewAppError("SetEnv setting must be string"))
			}
		}
		app.settings[k] = val
		return
	}
	app.settings[key] = val
}

// Listen starts the HTTP server.
func (app *App) Listen(addr string) error {
	app.Server.Addr = addr
	app.Server.ErrorLog = app.logger
	app.Server.Handler = app
	return app.Server.ListenAndServe()
}

// ListenTLS starts the HTTPS server.
func (app *App) ListenTLS(addr, certFile, keyFile string) error {
	app.Server.Addr = addr
	app.Server.ErrorLog = app.logger
	app.Server.Handler = app
	return app.Server.ListenAndServeTLS(certFile, keyFile)
}

// Start starts a non-blocking app instance. It is useful for testing.
// If addr omit, the app will listen on a random addr, use ServerListener.Addr() to get it.
// The non-blocking app instance must close by ServerListener.Close().
func (app *App) Start(addr ...string) *ServerListener {
	laddr := "127.0.0.1:0"
	if len(addr) > 0 && addr[0] != "" {
		laddr = addr[0]
	}
	app.Server.ErrorLog = app.logger
	app.Server.Handler = app

	l, err := net.Listen("tcp", laddr)
	if err != nil {
		panic(NewAppError(fmt.Sprintf("failed to listen on %v: %v", laddr, err)))
	}

	c := make(chan error)
	go func() {
		c <- app.Server.Serve(l)
	}()
	return &ServerListener{l, c}
}

// Error writes error to underlayer logging system.
func (app *App) Error(err error) {
	if err := ParseError(err); err != nil {
		if err.Code == 500 || err.Code > 501 || err.Code < 400 {
			app.logger.Println(err.String())
		}
	}
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(app, w, r)

	if compressWriter := ctx.handleCompress(); compressWriter != nil {
		defer compressWriter.Close()
	}

	// recover panic error
	defer func() {
		if err := recover(); err != nil {
			ctx.Res.ResetHeader()
			ctx.salvage(ErrorWithStack(err))
		}
	}()

	go func() {
		<-ctx.Done()
		ctx.ended.setTrue()
	}()

	// process app middleware
	err := app.middleware.run(ctx)
	if ctx.Res.wroteHeader.isTrue() {
		if !IsNil(err) {
			app.Error(err)
		}
		return
	}

	if IsNil(err) {
		// if context canceled abnormally...
		if err = ctx.Err(); err != nil {
			err = &Error{http.StatusGatewayTimeout, err.Error(), "", nil}
		}
	}

	if !IsNil(err) {
		ctx.ended.setTrue()
		ctx.Res.ResetHeader()
		// process middleware error with OnError
		if err := app.onerror.OnError(ctx, err); err != nil {
			ctx.salvage(err)
			return
		}
	}
	// ensure respond
	ctx.Res.WriteHeader(0)
}

// ServerListener is returned by a non-blocking app instance.
type ServerListener struct {
	l net.Listener
	c <-chan error
}

// Close closes the non-blocking app instance.
func (s *ServerListener) Close() error {
	return s.l.Close()
}

// Addr returns the non-blocking app instance addr.
func (s *ServerListener) Addr() net.Addr {
	return s.l.Addr()
}

// Wait make the non-blocking app instance blocking.
func (s *ServerListener) Wait() error {
	return <-s.c
}

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

// Compose composes a array of middlewares to one middleware
func Compose(mds ...Middleware) Middleware {
	if len(mds) == 0 {
		panic(NewAppError("middleware functions required"))
	}
	return middlewares(mds).run
}

type middlewares []Middleware

func (m middlewares) run(ctx *Context) (err error) {
	for _, handle := range m {
		if err = handle(ctx); !IsNil(err) || ctx.ended.isTrue() {
			return err
		}
	}
	return nil
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
