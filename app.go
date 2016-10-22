package gear

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

// Gear is the top-level framework app instance.
type Gear struct {
	middleware []Middleware
	pool       sync.Pool

	// ErrorLog specifies an optional logger for app's errors.
	ErrorLog *log.Logger

	// OnCtxError is error handle for Middleware error.
	OnCtxError func(Context, error) *HTTPError
	Renderer   Renderer
	Server     *http.Server
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

// Middleware defines a function to process middleware.
type Middleware func(Context) error

// Hook defines a function to process hook.
type Hook func(Context)

// HTTPError represents an error that occurred while handling a request.
type HTTPError struct {
	error
	Code int
}

// NewHTTPError creates an instance of HTTPError with status code and error message.
func NewHTTPError(code int, err string) *HTTPError {
	return &HTTPError{errors.New(err), code}
}

// Handler is the interface that wraps the Middleware function.
type Handler interface {
	Middleware(Context) error
}

// Renderer is the interface that wraps the Render function.
type Renderer interface {
	Render(Context, io.Writer, string, interface{}) error
}

// New creates an instance of Gear.
func New() *Gear {
	g := new(Gear)
	g.Server = new(http.Server)
	g.middleware = make([]Middleware, 0)
	g.pool.New = func() interface{} {
		ctx := &gearCtx{app: g, res: &Response{}}
		ctx.res.ctx = ctx
		return ctx
	}
	return g
}

func (g *Gear) toServeHandler() *servHandler {
	if len(g.middleware) == 0 {
		panic("No middleware")
	}
	return &servHandler{middleware: g.middleware[0:], app: g}
}

// OnError is default app error handler.
func (g *Gear) OnError(err error) {
	if g.ErrorLog != nil {
		g.ErrorLog.Println(err)
	} else {
		log.Println(err)
	}
}

// Use uses the given middleware `handle`.
func (g *Gear) Use(handle Middleware) {
	g.middleware = append(g.middleware, handle)
}

// UseHandler uses a instance that implemented Handler interface.
func (g *Gear) UseHandler(h Handler) {
	g.middleware = append(g.middleware, h.Middleware)
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

type servHandler struct {
	app        *Gear
	middleware []Middleware
}

func (h *servHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx := h.app.pool.Get().(*gearCtx)
	ctx.reset(w, r)

	for _, handle := range h.middleware {
		if err = handle(ctx); err != nil {
			break
		}
		if ctx.ended || ctx.res.finished {
			break // end up the process
		}
	}
	// set ended to true after app's middleware process
	ctx.ended = true

	// process middleware error with OnCtxError
	if err != nil && h.app.OnCtxError != nil {
		if ctxErr := h.app.OnCtxError(ctx, err); ctxErr != nil {
			ctx.Status(ctxErr.Code)
			err = ctxErr
		}
	}

	if err == nil { // ctx.afterHooks should not run when err
		ctx.runAfterHooks()
	} else {
		h.app.OnError(err)
		if ctx.res.Status < 400 {
			ctx.Status(500)
		}
		ctx.Body(err.Error())
	}

	err = ctx.res.respond()
	if err != nil {
		h.app.OnError(err)
	}

	ctx.reset(nil, nil)
	h.app.pool.Put(ctx)
}
