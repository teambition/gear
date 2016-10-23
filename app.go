package gear

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

// Version is Gear's version
const Version = "v0.5.0"

// HTTPError represents an error that occurred while handling a request.
type HTTPError interface {
	error
	Status() int
}

// Handler is the interface that wraps the HandlerFunc function.
type Handler interface {
	Serve(*Context) error
}

// Renderer is the interface that wraps the Render function.
type Renderer interface {
	Render(*Context, io.Writer, string, interface{}) error
}

// Hook defines a function to process hook.
type Hook func(*Context)

// Middleware defines a function to process middleware.
type Middleware func(*Context) error

// Error is error struct that implemented HTTPError
type Error struct {
	error
	Code int
}

// Status returns Error's status code.
func (err Error) Status() int {
	return err.Code
}

// NewError create a Error instance with error and status code.
func NewError(err error, code int) Error {
	return Error{error: err, Code: code}
}

// ServerBG is a server returned by a background app instance.
type ServerBG struct {
	l net.Listener
	c <-chan error
}

// Close closes the background app instance.
func (s *ServerBG) Close() error {
	return s.l.Close()
}

// Addr returns the background app instance addr.
func (s *ServerBG) Addr() net.Addr {
	return s.l.Addr()
}

// Wait waits the background app instance close.
func (s *ServerBG) Wait() error {
	return <-s.c
}

// Gear is the top-level framework app instance.
type Gear struct {
	middleware []Middleware
	pool       sync.Pool

	// OnError is default ctx error handler.
	// Override it for your business logic.
	OnError  func(*Context, error) HTTPError
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
		return NewContext(g)
	}
	g.OnError = func(ctx *Context, err error) HTTPError {
		return NewError(err, 500)
	}
	return g
}

func (g *Gear) toServeHandler() *serveHandler {
	if len(g.middleware) == 0 {
		panic("No middleware")
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

// StartBG starts a background app instance. It is useful for testing.
// The background app instance must close by ServerBG.Close().
func (g *Gear) StartBG(laddr string) *ServerBG {
	if laddr == "" {
		laddr = "127.0.0.1:0"
	}
	g.Server.Handler = g.toServeHandler()
	if g.ErrorLog != nil {
		g.Server.ErrorLog = g.ErrorLog
	}

	l, err := net.Listen("tcp", laddr)
	if err != nil {
		panic(fmt.Sprintf("failed to listen on %v: %v", laddr, err))
	}

	c := make(chan error)
	go func() {
		c <- g.Server.Serve(l)
	}()
	return &ServerBG{l, c}
}

// Error writes error to underlayer logging system.
func (g *Gear) Error(err error) {
	if err == nil {
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
	ctx.Reset(w, r)

	for _, handle := range h.middleware {
		if err = handle(ctx); err != nil {
			break
		}
		if ctx.ended || ctx.Res.finished {
			break // end up the process
		}
	}
	// set ended to true after app's middleware process
	ctx.ended = true

	// process middleware error with OnCtxError
	if err != nil {
		if ctxErr := h.app.OnError(ctx, err); ctxErr != nil {
			ctx.Status(ctxErr.Status())
			err = ctxErr
		}
	}

	if err == nil { // ctx.afterHooks should not run when err
		ctx.runAfterHooks()
	} else {
		if ctx.Res.Status < 400 {
			ctx.Res.Status = 500
		}
		if ctx.Res.Status >= 500 { // Only handle 5xx Server Error
			h.app.Error(err)
		}
		ctx.Body(err.Error())
	}

	err = ctx.Res.respond()
	if err != nil {
		h.app.Error(err)
	}

	ctx.Reset(nil, nil)
	h.app.pool.Put(ctx)
}

// WrapHandler wrap a http.Handler to Gear Middleware
func WrapHandler(h http.Handler) Middleware {
	return func(ctx *Context) error {
		h.ServeHTTP(ctx.Res, ctx.Req)
		return nil
	}
}

// WrapHandlerFunc wrap a http.HandlerFunc to Gear Middleware
func WrapHandlerFunc(h http.HandlerFunc) Middleware {
	return func(ctx *Context) error {
		h(ctx.Res, ctx.Req)
		return nil
	}
}
