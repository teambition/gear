package gear

import (
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

// BodyParser interface is used by ctx.ParseBody.
type BodyParser interface {
	// Maximum allowed size for a request body
	MaxBytes() int64
	Parse(buf []byte, body interface{}, mediaType, charset string) error
}

// DefaultBodyParser is default BodyParser type.
// "AppBodyParser" used 1MB as default:
//
//  app.Set("AppBodyParser", DefaultBodyParser(1<<20))
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

// OnError interface is use to deal with errors returned by middlewares.
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
	if !isNil(e) {
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
	settings   map[string]interface{}

	onerror    OnError
	renderer   Renderer
	bodyParser BodyParser
	// Default to nil, do not compress response content.
	compress Compressible
	// Default to 0
	timeout time.Duration
	// ErrorLog specifies an optional logger for app's errors. Default to nil.
	logger *log.Logger
}

// New creates an instance of App.
func New() *App {
	app := new(App)
	app.Server = new(http.Server)
	app.middleware = make(middlewares, 0)
	app.settings = make(map[string]interface{})

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	app.Set("AppEnv", env)
	app.Set("AppOnError", &DefaultOnError{})
	app.Set("AppBodyParser", DefaultBodyParser(1<<20))
	app.Set("AppLogger", log.New(os.Stderr, "", log.LstdFlags))
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

// Set add app settings. The settings can be retrieved by ctx.Setting.
// There are 4 build-in app settings:
//
//  app.Set("AppOnError", val gear.OnError)       // Default to gear.DefaultOnError
//  app.Set("AppRenderer", val gear.Renderer)     // No default
//  app.Set("AppLogger", val *log.Logger)         // No default
//  app.Set("AppBodyParser", val gear.BodyParser) // Default to gear.DefaultBodyParser
//  app.Set("AppTimeout", val time.Duration)      // Default to 0, no timeout
//  app.Set("AppCompress", val gear.Compress)     // Enable to compress response content.
//  app.Set("AppEnv", val string) // It will read os env with key `APP_ENV` Default to "development"
//
func (app *App) Set(setting string, val interface{}) {
	switch setting {
	case "AppOnError":
		if onerror, ok := val.(OnError); !ok {
			panic(NewAppError("AppOnError setting must implemented gear.OnError interface"))
		} else {
			app.onerror = onerror
		}
	case "AppRenderer":
		if renderer, ok := val.(Renderer); !ok {
			panic(NewAppError("AppRenderer setting must implemented gear.Renderer interface"))
		} else {
			app.renderer = renderer
		}
	case "AppLogger":
		if logger, ok := val.(*log.Logger); !ok {
			panic(NewAppError("AppLogger setting must be *log.Logger instance"))
		} else {
			app.logger = logger
		}
	case "AppBodyParser":
		if bodyParser, ok := val.(BodyParser); !ok {
			panic(NewAppError("AppBodyParser setting must implemented gear.BodyParser interface"))
		} else {
			app.bodyParser = bodyParser
		}
	case "AppCompress":
		if compress, ok := val.(Compressible); !ok {
			panic(NewAppError("AppCompress setting must implemented gear.Compressible interface"))
		} else {
			app.compress = compress
		}
	case "AppTimeout":
		if timeout, ok := val.(time.Duration); !ok {
			panic(NewAppError("AppTimeout setting must be time.Duration instance"))
		} else {
			app.timeout = timeout
		}
	case "AppEnv":
		if _, ok := val.(string); !ok {
			panic(NewAppError("AppEnv setting must be string"))
		}
	}
	app.settings[setting] = val
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
		if !isNil(err) {
			app.Error(err)
		}
		return
	}

	if isNil(err) {
		// if context canceled abnormally...
		if err = ctx.Err(); err != nil {
			err = &Error{http.StatusGatewayTimeout, err.Error(), "", nil}
		}
	}

	if !isNil(err) {
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

// isNil checks if a specified object is nil or not, without Failing.
func isNil(val interface{}) bool {
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

type middlewares []Middleware

func (m middlewares) run(ctx *Context) (err error) {
	for _, handle := range m {
		if err = handle(ctx); !isNil(err) || ctx.ended.isTrue() {
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
