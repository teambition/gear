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
	"os"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Middleware defines a function to process as middleware.
type Middleware func(ctx *Context) error

// Handler interface is used by app.UseHandler as a middleware.
type Handler interface {
	Serve(ctx *Context) error
}

// Sender interface is used by ctx.Send.
type Sender interface {
	Send(ctx *Context, code int, data interface{}) error
}

// Renderer interface is used by ctx.Render.
type Renderer interface {
	Render(ctx *Context, w io.Writer, name string, data interface{}) error
}

// URLParser interface is used by ctx.ParseUrl. Default to:
//
//	app.Set(gear.SetURLParser, gear.DefaultURLParser)
type URLParser interface {
	Parse(val map[string][]string, body interface{}, tag string) error
}

// DefaultURLParser is default URLParser type.
type DefaultURLParser struct{}

// Parse implemented URLParser interface.
func (d DefaultURLParser) Parse(val map[string][]string, body interface{}, tag string) error {
	return ValuesToStruct(val, body, tag)
}

// BodyParser interface is used by ctx.ParseBody. Default to:
//
//	app.Set(gear.SetBodyParser, gear.DefaultBodyParser(1<<20))
type BodyParser interface {
	// Maximum allowed size for a request body
	MaxBytes() int64
	Parse(buf []byte, body interface{}, mediaType, charset string) error
}

// DefaultBodyParser is default BodyParser type.
// SetBodyParser used 1MB as default:
//
//	app.Set(gear.SetBodyParser, gear.DefaultBodyParser(1<<20))
type DefaultBodyParser int64

// MaxBytes implemented BodyParser interface.
func (d DefaultBodyParser) MaxBytes() int64 {
	return int64(d)
}

// Parse implemented BodyParser interface.
func (d DefaultBodyParser) Parse(buf []byte, body interface{}, mediaType, charset string) error {
	if len(buf) == 0 {
		return ErrBadRequest.WithMsg("request entity empty")
	}
	switch true {
	case strings.HasPrefix(mediaType, MIMEApplicationJSON), isLikeMediaType(mediaType, "json"):
		err := json.Unmarshal(buf, body)
		if err == nil {
			return nil
		}

		if ute, ok := err.(*json.UnmarshalTypeError); ok {
			if ute.Field == "" { // go1.11
				return fmt.Errorf("Unmarshal type error: expected=%v, got=%v, offset=%v",
					ute.Type, ute.Value, ute.Offset)
			}
			return fmt.Errorf("Unmarshal type error: field=%v, expected=%v, got=%v, offset=%v",
				ute.Field, ute.Type, ute.Value, ute.Offset)
		} else if se, ok := err.(*json.SyntaxError); ok {
			return fmt.Errorf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())
		} else {
			return err
		}
	case strings.HasPrefix(mediaType, MIMEApplicationXML), isLikeMediaType(mediaType, "xml"):
		return xml.Unmarshal(buf, body)
	}

	return ErrUnsupportedMediaType.WithMsgf("unsupported media type: %s", mediaType)
}

// HTTPError interface is used to create a server error that include status code and error message.
type HTTPError interface {
	// Error returns error's message.
	Error() string
	// Status returns error's http status code.
	Status() int
}

// App is the top-level framework struct.
//
// Hello Gear!
//
//	package main
//
//	import "github.com/teambition/gear"
//
//	func main() {
//		app := gear.New() // Create app
//		app.Use(func(ctx *gear.Context) error {
//			return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//		})
//		app.Error(app.Listen(":3000"))
//	}
type App struct {
	Server *http.Server
	mds    middlewares

	keys        []string
	renderer    Renderer
	sender      Sender
	bodyParser  BodyParser
	urlParser   URLParser
	compress    Compressible  // Default to nil, do not compress response content.
	timeout     time.Duration // Default to 0, no time out.
	serverName  string        // Gear/1.7.6
	logger      *log.Logger
	parseError  func(error) HTTPError
	renderError func(HTTPError) (code int, contentType string, body []byte)
	onerror     func(*Context, HTTPError)
	withContext func(*http.Request) context.Context
	settings    map[interface{}]interface{}
}

// New creates an instance of App.
func New() *App {
	app := new(App)
	app.Server = new(http.Server)
	// https://medium.com/@simonfrey/go-as-in-golang-standard-net-http-config-will-break-your-production-environment-1360871cb72b
	app.Server.ReadHeaderTimeout = 20 * time.Second
	app.Server.ReadTimeout = 60 * time.Second
	app.Server.WriteTimeout = 120 * time.Second
	app.Server.IdleTimeout = 90 * time.Second

	app.mds = make(middlewares, 0)
	app.settings = make(map[interface{}]interface{})

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	app.Set(SetEnv, env)
	app.Set(SetServerName, "Gear/"+Version)
	app.Set(SetTrustedProxy, false)
	app.Set(SetBodyParser, DefaultBodyParser(2<<20)) // 2MB
	app.Set(SetURLParser, DefaultURLParser{})
	app.Set(SetLogger, log.New(os.Stderr, "", 0))
	app.Set(SetGraceTimeout, 10*time.Second)
	app.Set(SetParseError, func(err error) HTTPError {
		return ParseError(err)
	})
	app.Set(SetRenderError, defaultRenderError)
	app.Set(SetOnError, func(ctx *Context, err HTTPError) {
		ctx.Error(err)
	})
	return app
}

// Use uses the given middleware `handle`.
func (app *App) Use(handle Middleware) *App {
	app.mds = append(app.mds, handle)
	return app
}

// UseHandler uses a instance that implemented Handler interface.
func (app *App) UseHandler(h Handler) *App {
	app.mds = append(app.mds, h.Serve)
	return app
}

type appSetting uint8

// Build-in app settings
const (
	// It will be used by `ctx.ParseBody`, value should implements `gear.BodyParser` interface, default to:
	//  app.Set(gear.SetBodyParser, gear.DefaultBodyParser(1<<20))
	SetBodyParser appSetting = iota

	// It will be used by `ctx.ParseURL`, value should implements `gear.URLParser` interface, default to:
	//  app.Set(gear.SetURLParser, gear.DefaultURLParser)
	SetURLParser

	// Enable compress for response, value should implements `gear.Compressible` interface, no default value.
	// Example:
	//  import "github.com/teambition/compressible-go"
	//
	//  app := gear.New()
	//  app.Set(gear.SetCompress, compressible.WithThreshold(1024))
	SetCompress

	// Set secret keys for signed cookies, it will be used by `ctx.Cookies`, value should be `[]string` type,
	// no default value. More document https://github.com/go-http-utils/cookie, Example:
	//  app.Set(gear.SetKeys, []string{"some key2", "some key1"})
	SetKeys

	// Set a logger to app, value should be `*log.Logger` instance, default to:
	//  app.Set(gear.SetLogger, log.New(os.Stderr, "", 0))
	// Maybe you need LoggerFilterWriter to filter some server errors in production:
	//  app.Set(gear.SetLogger, log.New(gear.DefaultFilterWriter(), "", 0))
	// We recommand set logger flags to 0.
	SetLogger

	// Set a ParseError hook to app that convert middleware error to HTTPError,
	// value should be `func(err error) HTTPError`, default to:
	//  app.Set(SetParseError, func(err error) HTTPError {
	//  	return ParseError(err)
	//  })
	SetParseError

	// Set a SetRenderError hook to app that convert error to raw response,
	// value should be `func(HTTPError) (code int, contentType string, body []byte)`, default to:
	//   app.Set(SetRenderError, func(err HTTPError) (int, string, []byte) {
	//  	// default to render error as json
	//  	body, e := json.Marshal(err)
	//  	if e != nil {
	//  		body, _ = json.Marshal(map[string]string{"error": err.Error()})
	//  	}
	//  	return err.Status(), MIMEApplicationJSONCharsetUTF8, body
	//  })
	//
	// you can use another recommand one:
	//
	//  app.Set(gear.SetRenderError, gear.RenderErrorResponse)
	//
	SetRenderError

	// Set a on-error hook to app that handle middleware error.
	// value should be `func(ctx *Context, err HTTPError)`, default to:
	//  app.Set(SetOnError, func(ctx *Context, err HTTPError) {
	//  	ctx.Error(err)
	//  })
	SetOnError

	// Set a SetSender to app, it will be used by `ctx.Send`, value should implements `gear.Sender` interface,
	// no default value.
	SetSender

	// Set a renderer to app, it will be used by `ctx.Render`, value should implements `gear.Renderer` interface,
	// no default value.
	SetRenderer

	// Set a timeout to for the middleware process, value should be `time.Duration`. No default.
	// Example:
	//  app.Set(gear.SetTimeout, 3*time.Second)
	SetTimeout

	// Set a graceful timeout to for gracefully shuts down, value should be `time.Duration`. Default to 10*time.Second.
	// Example:
	//  app.Set(gear.SetGraceTimeout, 60*time.Second)
	SetGraceTimeout

	// Set a function that Wrap the gear.Context' underlayer context.Context. No default.
	SetWithContext

	// Set a app env string to app, it can be retrieved by `ctx.Setting(gear.SetEnv)`.
	// Default to os process "APP_ENV" or "development".
	SetEnv

	// Set a server name that respond to client as "Server" header.
	// Default to "Gear/{version}".
	SetServerName

	// Set true and proxy header fields will be trusted
	// Default to false.
	SetTrustedProxy
)

// Set add key/value settings to app. The settings can be retrieved by `ctx.Setting(key)`.
func (app *App) Set(key, val interface{}) *App {
	if k, ok := key.(appSetting); ok {
		switch key {
		case SetBodyParser:
			if bodyParser, ok := val.(BodyParser); !ok {
				panic(Err.WithMsg("SetBodyParser setting must implemented `gear.BodyParser` interface"))
			} else {
				app.bodyParser = bodyParser
			}
		case SetURLParser:
			if urlParser, ok := val.(URLParser); !ok {
				panic(Err.WithMsg("SetURLParser setting must implemented `gear.URLParser` interface"))
			} else {
				app.urlParser = urlParser
			}
		case SetCompress:
			if compress, ok := val.(Compressible); !ok {
				panic(Err.WithMsg("SetCompress setting must implemented `gear.Compressible` interface"))
			} else {
				app.compress = compress
			}
		case SetKeys:
			if keys, ok := val.([]string); !ok {
				panic(Err.WithMsg("SetKeys setting must be `[]string`"))
			} else {
				app.keys = keys
			}
		case SetLogger:
			if logger, ok := val.(*log.Logger); !ok {
				panic(Err.WithMsg("SetLogger setting must be `*log.Logger` instance"))
			} else {
				app.logger = logger
			}
		case SetParseError:
			if parseError, ok := val.(func(error) HTTPError); !ok {
				panic(Err.WithMsg("SetParseError setting must be `func(error) HTTPError`"))
			} else {
				app.parseError = parseError
			}
		case SetRenderError:
			if renderError, ok := val.(func(HTTPError) (int, string, []byte)); !ok {
				panic(Err.WithMsg("SetRenderError setting must be `func(HTTPError) (int, string, []byte)`"))
			} else {
				app.renderError = renderError
			}
		case SetOnError:
			if onerror, ok := val.(func(*Context, HTTPError)); !ok {
				panic(Err.WithMsg("SetOnError setting must be `func(*Context, HTTPError)`"))
			} else {
				app.onerror = onerror
			}
		case SetSender:
			if sender, ok := val.(Sender); !ok {
				panic(Err.WithMsg("SetSender setting must implemented `gear.Sender` interface"))
			} else {
				app.sender = sender
			}
		case SetRenderer:
			if renderer, ok := val.(Renderer); !ok {
				panic(Err.WithMsg("SetRenderer setting must implemented `gear.Renderer` interface"))
			} else {
				app.renderer = renderer
			}
		case SetTimeout:
			if timeout, ok := val.(time.Duration); !ok {
				panic(Err.WithMsg("SetTimeout setting must be `time.Duration` instance"))
			} else {
				app.timeout = timeout
			}
		case SetGraceTimeout:
			if _, ok := val.(time.Duration); !ok {
				panic(Err.WithMsg("SetGraceTimeout setting must be `time.Duration` instance"))
			}
		case SetWithContext:
			if withContext, ok := val.(func(*http.Request) context.Context); !ok {
				panic(Err.WithMsg("SetWithContext setting must be `func(*http.Request) context.Context`"))
			} else {
				app.withContext = withContext
			}
		case SetEnv:
			if _, ok := val.(string); !ok {
				panic(Err.WithMsg("SetEnv setting must be `string`"))
			}
		case SetServerName:
			if name, ok := val.(string); !ok {
				panic(Err.WithMsg("SetServerName setting must be `string`"))
			} else {
				app.serverName = name
			}
		case SetTrustedProxy:
			if _, ok := val.(bool); !ok {
				panic(Err.WithMsg("SetTrustedProxy setting must be `bool`"))
			}
		}
		app.settings[k] = val
		return app
	}
	app.settings[key] = val
	return app
}

// Env returns app' env. You can set app env with `app.Set(gear.SetEnv, "some env")`
// Default to os process "APP_ENV" or "development".
func (app *App) Env() string {
	return app.settings[SetEnv].(string)
}

// Listen starts the HTTP server.
func (app *App) Listen(addr string) error {
	app.Server.Addr = addr
	app.Server.ErrorLog = app.logger
	app.Server.Handler = h2c.NewHandler(app, &http2.Server{})
	return app.Server.ListenAndServe()
}

// ListenTLS starts the HTTPS server.
func (app *App) ListenTLS(addr, certFile, keyFile string) error {
	app.Server.Addr = addr
	app.Server.ErrorLog = app.logger
	app.Server.Handler = app
	return app.Server.ListenAndServeTLS(certFile, keyFile)
}

// ListenWithContext starts the HTTP server (or HTTPS server with keyPair) with a context
//
// Usage:
//
//	 func main() {
//	 	app := gear.New() // Create app
//	 	do some thing...
//
//	 	app.ListenWithContext(gear.ContextWithSignal(context.Background()), addr)
//		  // starts the HTTPS server.
//		  // app.ListenWithContext(gear.ContextWithSignal(context.Background()), addr, certFile, keyFile)
//	 }
func (app *App) ListenWithContext(ctx context.Context, addr string, keyPair ...string) error {
	timeout := app.settings[SetGraceTimeout].(time.Duration)
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := app.Close(c); err != nil {
			app.Error(err)
		}
	}()

	if len(keyPair) >= 2 && keyPair[0] != "" && keyPair[1] != "" {
		return app.ListenTLS(addr, keyPair[0], keyPair[1])
	}
	return app.Listen(addr)
}

// ServeWithContext accepts incoming connections on the Listener l, starts the HTTP server (or HTTPS server with keyPair) with a context
//
// Usage:
//
//	 func main() {
//			l, err := net.Listen("tcp", ":8080")
//			if err != nil {
//				log.Fatal(err)
//			}
//
//	 	app := gear.New() // Create app
//	 	do some thing...
//
//	 	app.ServeWithContext(gear.ContextWithSignal(context.Background()), l)
//		  // starts the HTTPS server.
//		  // app.ServeWithContext(gear.ContextWithSignal(context.Background()), l, certFile, keyFile)
//	 }
func (app *App) ServeWithContext(ctx context.Context, l net.Listener, keyPair ...string) error {
	timeout := app.settings[SetGraceTimeout].(time.Duration)
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := app.Close(c); err != nil {
			app.Error(err)
		}
	}()

	app.Server.ErrorLog = app.logger
	app.Server.Handler = app
	if len(keyPair) >= 2 && keyPair[0] != "" && keyPair[1] != "" {
		return app.Server.ServeTLS(l, keyPair[0], keyPair[1])
	}
	return app.Server.Serve(l)
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
		panic(Err.WithMsgf("failed to listen on %v: %v", laddr, err))
	}

	c := make(chan error)
	go func() {
		c <- app.Server.Serve(l)
	}()
	return &ServerListener{l, c}
}

// Error writes error to underlayer logging system.
func (app *App) Error(err interface{}) {
	if err := ErrorWithStack(err, 2); err != nil {
		str, e := err.Format()
		f := app.logger.Flags() == 0
		switch {
		case f && e == nil:
			app.logger.Printf("[%s] ERR %s\n", time.Now().UTC().Format("2006-01-02T15:04:05.999Z"), str)
		case f && e != nil:
			app.logger.Printf("[%s] CRIT %s\n", time.Now().UTC().Format("2006-01-02T15:04:05.999Z"), err.String())
		case !f && e == nil:
			app.logger.Printf("ERR %s\n", str)
		default:
			app.logger.Printf("CRIT %s\n", err.String())
		}
	}
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(app, w, r)

	if compressWriter := ctx.handleCompress(); compressWriter != nil {
		defer compressWriter.Close()
	}

	// recover panic error
	defer catchRequest(ctx)
	go handleCtxEnd(ctx)

	// process app middleware
	err := app.mds.run(ctx)
	if ctx.Res.wroteHeader.isTrue() {
		if !IsNil(err) {
			app.Error(err)
		}
		return
	}

	// if context canceled abnormally...
	if e := ctx.Err(); e != nil {
		if e == context.Canceled {
			// https://stackoverflow.com/questions/46234679/what-is-the-correct-http-status-code-for-a-cancelled-request
			// 499 Client Closed Request Used when the client has closed
			// the request before the server could send a response.
			ctx.Res.WriteHeader(ErrClientClosedRequest.Code)
			return
		}
		err = ErrGatewayTimeout.WithMsg(e.Error())
	}

	// handle middleware errors
	if !IsNil(err) {
		ctx.Res.afterHooks = nil // clear afterHooks when any error
		ctx.Res.ResetHeader()
		e := app.parseError(err)
		app.onerror(ctx, e)
		// try to ensure respond error if `app.onerror` does't do it.
		ctx.respondError(e)
	} else {
		// try to ensure respond
		ctx.Res.respond(0, nil)
	}
}

// Close closes the underlying server gracefully.
// If context omit, Server.Close will be used to close immediately.
// Otherwise Server.Shutdown will be used to close gracefully.
func (app *App) Close(ctx ...context.Context) error {
	if len(ctx) > 0 {
		return app.Server.Shutdown(ctx[0])
	}
	return app.Server.Close()
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

func catchRequest(ctx *Context) {
	if err := recover(); err != nil && err != http.ErrAbortHandler {
		ctx.Res.afterHooks = nil
		ctx.Res.ResetHeader()
		e := ErrorWithStack(err, 3)
		ctx.app.onerror(ctx, e)
		// try to ensure respond error if `app.onerror` does't do it.
		ctx.respondError(e)
	}
	// execute "end hooks" with LIFO order after Response.WriteHeader.
	// they run in a goroutine, in order to not block current HTTP Request/Response.
	if len(ctx.Res.endHooks) > 0 {
		go tryRunHooks(ctx.app, ctx.Res.endHooks)
	}
}

func handleCtxEnd(ctx *Context) {
	<-ctx.done
	ctx.Res.ended.setTrue()
}

func runHooks(hooks []func()) {
	// run hooks in LIFO order
	for i := len(hooks) - 1; i >= 0; i-- {
		hooks[i]()
	}
}

func tryRunHooks(app *App, hooks []func()) {
	defer catchErr(app)
	runHooks(hooks)
}

func catchErr(app *App) {
	if err := recover(); err != nil && err != http.ErrAbortHandler {
		app.Error(err)
	}
}
