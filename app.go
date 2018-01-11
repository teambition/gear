package gear

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
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
//  app.Set(gear.SetURLParser, gear.DefaultURLParser)
//
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
//  app.Set(gear.SetBodyParser, gear.DefaultBodyParser(1<<20))
//
type BodyParser interface {
	// Maximum allowed size for a request body
	MaxBytes() int64
	Parse(buf []byte, body interface{}, mediaType, charset string) error
}

// DefaultBodyParser is default BodyParser type.
// SetBodyParser used 1MB as default:
//
//  app.Set(gear.SetBodyParser, gear.DefaultBodyParser(1<<20))
//
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
	switch mediaType {
	case MIMEApplicationJSON:
		return json.Unmarshal(buf, body)
	case MIMEApplicationXML:
		return xml.Unmarshal(buf, body)
	case MIMEApplicationForm:
		val, err := url.ParseQuery(string(buf))
		if err == nil {
			err = ValuesToStruct(val, body, "form")
		}
		return err
	}

	if isLikeJSONType(mediaType) {
		return json.Unmarshal(buf, body)
	}
	return ErrUnsupportedMediaType.WithMsg("unsupported media type")
}

// HTTPError interface is used to create a server error that include status code and error message.
type HTTPError interface {
	// Error returns error's message.
	Error() string
	// Status returns error's http status code.
	Status() int
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
	onerror     func(*Context, HTTPError)
	withContext func(*http.Request) context.Context
	settings    map[interface{}]interface{}
}

// New creates an instance of App.
func New() *App {
	app := new(App)
	app.Server = new(http.Server)
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

	// Set a on-error hook to app, value should be `func(ctx *Context, err *Error)`, no default value.
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
				panic(Err.WithMsg("SetBodyParser setting must implemented gear.BodyParser interface"))
			} else {
				app.bodyParser = bodyParser
			}
		case SetURLParser:
			if urlParser, ok := val.(URLParser); !ok {
				panic(Err.WithMsg("SetURLParser setting must implemented gear.URLParser interface"))
			} else {
				app.urlParser = urlParser
			}
		case SetCompress:
			if compress, ok := val.(Compressible); !ok {
				panic(Err.WithMsg("SetCompress setting must implemented gear.Compressible interface"))
			} else {
				app.compress = compress
			}
		case SetKeys:
			if keys, ok := val.([]string); !ok {
				panic(Err.WithMsg("SetKeys setting must be []string"))
			} else {
				app.keys = keys
			}
		case SetLogger:
			if logger, ok := val.(*log.Logger); !ok {
				panic(Err.WithMsg("SetLogger setting must be *log.Logger instance"))
			} else {
				app.logger = logger
			}
		case SetOnError:
			if onerror, ok := val.(func(ctx *Context, err HTTPError)); !ok {
				panic(Err.WithMsg("SetOnError setting must be func(ctx *Context, err *Error)"))
			} else {
				app.onerror = onerror
			}
		case SetSender:
			if sender, ok := val.(Sender); !ok {
				panic(Err.WithMsg("SetSender setting must implemented gear.Sender interface"))
			} else {
				app.sender = sender
			}
		case SetRenderer:
			if renderer, ok := val.(Renderer); !ok {
				panic(Err.WithMsg("SetRenderer setting must implemented gear.Renderer interface"))
			} else {
				app.renderer = renderer
			}
		case SetTimeout:
			if timeout, ok := val.(time.Duration); !ok {
				panic(Err.WithMsg("SetTimeout setting must be time.Duration instance"))
			} else {
				app.timeout = timeout
			}
		case SetWithContext:
			if withContext, ok := val.(func(*http.Request) context.Context); !ok {
				panic(Err.WithMsg("SetWithContext setting must be func(*http.Request) context.Context"))
			} else {
				app.withContext = withContext
			}
		case SetEnv:
			if _, ok := val.(string); !ok {
				panic(Err.WithMsg("SetEnv setting must be string"))
			}
		case SetServerName:
			if name, ok := val.(string); !ok {
				panic(Err.WithMsg("SetServerName setting must be string"))
			} else {
				app.serverName = name
			}
		case SetTrustedProxy:
			if _, ok := val.(bool); !ok {
				panic(Err.WithMsg("SetTrustedProxy setting must be bool"))
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
		panic(Err.WithMsgf("failed to listen on %v: %v", laddr, err))
	}

	c := make(chan error)
	go func() {
		c <- app.Server.Serve(l)
	}()
	return &ServerListener{l, c}
}

// Error writes error to underlayer logging system.
func (app *App) Error(err error) {
	if err := ErrorWithStack(err, 4); err != nil {
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
	defer func() {
		if err := recover(); err != nil && err != http.ErrAbortHandler {
			ctx.Res.afterHooks = nil
			ctx.Res.ResetHeader()
			ctx.respondError(ErrorWithStack(err))
		}
		// execute "end hooks" with LIFO order after Response.WriteHeader.
		// they run in a goroutine, in order to not block current HTTP Request/Response.
		if len(ctx.Res.endHooks) > 0 {
			go runHooks(ctx.Res.endHooks)
		}
	}()

	go func() {
		<-ctx.Done()
		ctx.Res.ended.setTrue()
	}()

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
			// 499 Client Closed Request Used when the client has closed the request before the server could send a response.
			ctx.Res.WriteHeader(ErrClientClosedRequest.Code)
			return
		}
		err = ErrGatewayTimeout.WithMsg(e.Error())
	}

	if !IsNil(err) {
		ctx.Error(err)
	} else {
		// ensure respond
		ctx.Res.WriteHeader(0)
	}
}

// Close closes the underlying server.
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
