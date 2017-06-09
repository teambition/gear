package gear

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
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

// Renderer interface is used by ctx.Render.
type Renderer interface {
	Render(ctx *Context, w io.Writer, name string, data interface{}) error
}

// URLParser interface is used by ctx.ParseUrl. Default to:
//  app.Set(gear.SetURLParser, DefaultURLParser)
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

// DefaultBodyParse is default BodyParse type, use 1MB as default max body length for json, xml, x-www form body
// use 10MB as default max body length for multipart form body
var DefaultBodyParse = func() *BodyParse {
	h := &BodyParse{}
	h.Set(MIMEApplicationJSON, ParseJSON, 1<<20)
	h.Set(MIMEApplicationXML, ParseXML, 1<<20)
	h.Set(MIMEApplicationForm, ParseApplicationForm, 1<<20)
	h.Set(MIMEMultipartForm, ParseMultipartForm(1<<20), 10<<20)
	return h
}()

// BodyParseFunc defines a function to support parse body
type BodyParseFunc func(data io.Reader, body interface{}, header http.Header) error

type funcAndMaxBytes struct {
	Fn       BodyParseFunc
	MaxBytes int64
}

// BodyParse is used by ctx.ParseBody. Default to:
//  app.Set(gear.SetBodyParse, DefaultBodyParse)
type BodyParse struct {
	Parsers map[string]funcAndMaxBytes
}

// Set set new BodyParseFunc to parse body.
func (h *BodyParse) Set(mediaType string, fn BodyParseFunc, maxBytes int64) error {
	if h.Parsers == nil {
		h.Parsers = make(map[string]funcAndMaxBytes)
	}

	mediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return err
	}

	if maxBytes == 0 {
		maxBytes = 1 << 20
	}

	h.Parsers[mediaType] = funcAndMaxBytes{fn, maxBytes}
	return nil
}

// Get return BodyParseFunc and maxBytes for parse mediaType.
func (h *BodyParse) Get(mediaType string) (BodyParseFunc, int64) {
	mediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return nil, 0
	}
	p := h.Parsers[mediaType]
	return p.Fn, p.MaxBytes
}

// ParseJSON is a BodyParseFunc to support parse json
func ParseJSON(data io.Reader, body interface{}, _ http.Header) error {
	blob, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(blob, body)
}

// ParseXML is a BodyParseFunc to support parse xml
func ParseXML(data io.Reader, body interface{}, _ http.Header) error {
	blob, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	return xml.Unmarshal(blob, body)
}

// ParseApplicationForm is a BodyParseFunc to support parse x-www form
func ParseApplicationForm(data io.Reader, body interface{}, _ http.Header) error {
	blob, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	val, err := url.ParseQuery(string(blob))
	if err == nil {
		err = ValuesToStruct(val, body, "form")
	}
	return err
}

// ParseMultipartForm is a BodyParseFunc to support parse multipart form
func ParseMultipartForm(maxMemory int64) BodyParseFunc {
	return func(data io.Reader, body interface{}, header http.Header) error {
		mediaType := header.Get(HeaderContentType)
		_, params, err := mime.ParseMediaType(mediaType)
		if err != nil {
			return err
		}
		boundary, ok := params["boundary"]
		if !ok {
			return http.ErrMissingBoundary
		}

		mr := multipart.NewReader(data, boundary)

		form, err := mr.ReadForm(maxMemory)
		if err != nil {
			return err
		}
		return FormToStruct(form, body, "form", "file")
	}
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
	bodyParse  *BodyParse
	urlParser   URLParser
	compress    Compressible  // Default to nil, do not compress response content.
	timeout     time.Duration // Default to 0, no time out.
	serverName  string        // Gear/1.7.2
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
	app.Set(SetBodyParse, DefaultBodyParse)
	app.Set(SetURLParser, DefaultURLParser{})
	app.Set(SetLogger, log.New(os.Stderr, "", log.LstdFlags))
	return app
}

// Use uses the given middleware `handle`.
func (app *App) Use(handle Middleware) {
	app.mds = append(app.mds, handle)
}

// UseHandler uses a instance that implemented Handler interface.
func (app *App) UseHandler(h Handler) {
	app.mds = append(app.mds, h.Serve)
}

type appSetting uint8

// Build-in app settings
const (
	// It will be used by `ctx.ParseBody`, value must be `gear.BodyParse`, default to:
	//  app.Set(gear.SetBodyParse, gear.DefaultBodyParse)
	SetBodyParse appSetting = iota

	// It will be used by `ctx.ParseURL`, value must implements `gear.URLParser` interface, default to:
	//  app.Set(gear.SetURLParser, gear.DefaultURLParser)
	SetURLParser

	// Enable compress for response, value must implements `gear.Compressible` interface, no default value.
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
	//  app.Set(gear.SetLogger, log.New(os.Stderr, "", log.LstdFlags))
	SetLogger

	// Set a on-error hook to app, value should be `func(ctx *Context, err *Error)`, no default value.
	SetOnError

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
)

// Set add key/value settings to app. The settings can be retrieved by `ctx.Setting(key)`.
func (app *App) Set(key, val interface{}) {
	if k, ok := key.(appSetting); ok {
		switch key {
		case SetBodyParse:
			if bodyParse, ok := val.(*BodyParse); !ok {
				panic(Err.WithMsg("SetBodyParse setting must be *gear.BodyParse"))
			} else {
				app.bodyParse = bodyParse
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
		}
		app.settings[k] = val
		return
	}
	app.settings[key] = val
}

// Env returns app' env. You can set app env with `app.Set(gear.SetEnv, "dome env")`
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
		app.logger.Println(err.String())
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
			ctx.Res.WriteHeader(http.StatusInternalServerError)
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
