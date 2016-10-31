package gear

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"reflect"
	"strings"
	"sync"
)

// Version is Gear's version
const Version = "v0.10.0"

// MIME types
const (
	// All const values got from https://github.com/labstack/echo
	charsetUTF8 = "charset=utf-8"

	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

// Headers
const (
	HeaderAcceptEncoding                = "Accept-Encoding"
	HeaderAllow                         = "Allow"
	HeaderAuthorization                 = "Authorization"
	HeaderContentDisposition            = "Content-Disposition"
	HeaderContentEncoding               = "Content-Encoding"
	HeaderContentLength                 = "Content-Length"
	HeaderContentType                   = "Content-Type"
	HeaderCookie                        = "Cookie"
	HeaderSetCookie                     = "Set-Cookie"
	HeaderIfModifiedSince               = "If-Modified-Since"
	HeaderLastModified                  = "Last-Modified"
	HeaderLocation                      = "Location"
	HeaderUpgrade                       = "Upgrade"
	HeaderUserAgent                     = "User-Agent"
	HeaderVary                          = "Vary"
	HeaderWWWAuthenticate               = "WWW-Authenticate"
	HeaderXForwardedProto               = "X-Forwarded-Proto"
	HeaderXHTTPMethodOverride           = "X-HTTP-Method-Override"
	HeaderXForwardedFor                 = "X-Forwarded-For"
	HeaderXRealIP                       = "X-Real-IP"
	HeaderServer                        = "Server"
	HeaderOrigin                        = "Origin"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderXCSRFToken              = "X-CSRF-Token"
)

// Handler interface is used by app.UseHandler as a middleware.
type Handler interface {
	Serve(*Context) error
}

// Renderer interface is used by ctx.Render.
type Renderer interface {
	Render(*Context, io.Writer, string, interface{}) error
}

// HTTPError interface is used to create a server error that include status code and error message.
type HTTPError interface {
	Error() string
	Status() int
}

// Hook defines a function to process as hook.
type Hook func(*Context)

// Middleware defines a function to process as middleware.
type Middleware func(*Context) error

// Error represents a numeric error with optional meta. It can be used in middleware as a return result.
type Error struct {
	Code int
	Msg  string
	Meta interface{}
}

// Status return error's http status code.
func (err *Error) Status() int {
	return err.Code
}

func (err *Error) Error() string {
	return err.Msg
}

// NewAppError create a error instance with "[Gear] " prefix.
func NewAppError(err string) error {
	return fmt.Errorf("[Gear] %s", err)
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
			err = &Error{Code: _e.Code, Msg: _e.Msg, Meta: e}
		case HTTPError:
			_e := e.(HTTPError)
			err = &Error{Code: _e.Status(), Msg: _e.Error(), Meta: e}
		default:
			err = &Error{Code: 500, Msg: e.Error(), Meta: e}
			if len(code) > 0 && code[0] > 0 {
				err.Code = code[0]
			}
		}
	}
	return err
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

// Gear is the top-level framework app instance.
//
// Hello Gear!
//
//  package main
//
//  import "github.com/teambition/gear"
//
//  func main() {
//  	app := gear.New() // Create app
//  	app.Use(gear.NewDefaultLogger())
//  	app.Use(func(ctx *gear.Context) error {
//  		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//  	})
//  	app.Error(app.Listen(":3000"))
//  }
//
type Gear struct {
	middleware []Middleware
	pool       sync.Pool

	// OnError is default ctx error handler.
	// Override it for yourself.
	OnError  func(*Context, error) *Error
	Renderer Renderer
	// ErrorLog specifies an optional logger for app's errors. Default to nil
	ErrorLog *log.Logger
	Server   *http.Server
}

// New creates an instance of Gear.
func New() *Gear {
	g := new(Gear)
	g.Server = new(http.Server)
	g.middleware = make([]Middleware, 0)
	g.pool.New = func() interface{} {
		return newContext(g)
	}
	g.OnError = func(ctx *Context, err error) *Error {
		var code int
		if ctx.Res.Status > 400 {
			code = ctx.Res.Status
		}
		return ParseError(err, code)
	}
	return g
}

func (g *Gear) toServeHandler() *serveHandler {
	if len(g.middleware) == 0 {
		panic(NewAppError("no middleware"))
	}
	return &serveHandler{middleware: g.middleware[0:], app: g}
}

// Use uses the given middleware `handle`.
func (g *Gear) Use(handle Middleware) {
	g.middleware = append(g.middleware, handle)
}

// UseHandler uses a instance that implemented Handler interface.
func (g *Gear) UseHandler(h Handler) {
	g.middleware = append(g.middleware, h.Serve)
}

// Listen starts the HTTP server.
func (g *Gear) Listen(addr string) error {
	g.Server.Addr = addr
	g.Server.Handler = g.toServeHandler()
	if g.ErrorLog != nil {
		g.Server.ErrorLog = g.ErrorLog
	}
	return g.Server.ListenAndServe()
}

// ListenTLS starts the HTTPS server.
func (g *Gear) ListenTLS(addr, certFile, keyFile string) error {
	g.Server.Addr = addr
	g.Server.Handler = g.toServeHandler()
	if g.ErrorLog != nil {
		g.Server.ErrorLog = g.ErrorLog
	}
	return g.Server.ListenAndServeTLS(certFile, keyFile)
}

// Start starts a non-blocking app instance. It is useful for testing.
// If addr omit, the app will listen on a random addr, use ServerListener.Addr() to get it.
// The non-blocking app instance must close by ServerListener.Close().
func (g *Gear) Start(addr ...string) *ServerListener {
	laddr := "127.0.0.1:0"
	if len(addr) > 0 && addr[0] != "" {
		laddr = addr[0]
	}
	g.Server.Handler = g.toServeHandler()
	if g.ErrorLog != nil {
		g.Server.ErrorLog = g.ErrorLog
	}

	l, err := net.Listen("tcp", laddr)
	if err != nil {
		panic(NewAppError(fmt.Sprintf("failed to listen on %v: %v", laddr, err)))
	}

	c := make(chan error)
	go func() {
		c <- g.Server.Serve(l)
	}()
	return &ServerListener{l, c}
}

// Error writes error to underlayer logging system (ErrorLog).
func (g *Gear) Error(err error) {
	if !isNil(err) {
		if g.ErrorLog != nil {
			g.ErrorLog.Println(err)
		} else {
			log.Println(err)
		}
	}
}

type serveHandler struct {
	app        *Gear
	middleware []Middleware
}

func (h *serveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx := h.app.pool.Get().(*Context)
	ctx.reset(w, r)

	// reuse Context instance, recover panic error
	defer func() {
		if err := recover(); err != nil {
			httprequest, _ := httputil.DumpRequest(ctx.Req, false)
			ctx.Error(&Error{Code: 500, Msg: http.StatusText(500)})
			h.app.Error(fmt.Errorf("panic recovered: %s; %s",
				err, strings.Replace(string(httprequest), "\n", "\\n", -1)))
		}
		ctx.reset(nil, nil)
		h.app.pool.Put(ctx)
	}()

	// process app middleware
	for _, handle := range h.middleware {
		if err = handle(ctx); !isNil(err) {
			break
		}
		if ctx.ended {
			break // end up the middleware process
		}
	}

	// ensure that ended is true after middleware process finished.
	ctx.ended = true
	if !isNil(err) {
		ctx.Type("text")     // reset Content-Type, but you can set it in OnError again.
		ctx.afterHooks = nil // clear afterHooks when error
		// process middleware error with OnError
		if ctxErr := h.app.OnError(ctx, err); ctxErr != nil {
			ctx.Status(ctxErr.Status())
			// 5xx Server Error will send to app.Error
			if ctx.Res.Status >= 500 {
				h.app.Error(ctxErr)
			}
			ctx.String(ctxErr.Error())
		}
	}

	// ensure respond
	ctx.Res.respond()
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
