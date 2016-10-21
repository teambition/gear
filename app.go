package gear

import (
	"errors"
	"log"
	"net/http"
	"sync"
)

// Gear is the top-level framework app instance.
type Gear struct {
	server     *http.Server
	middleware []Middleware
	pool       sync.Pool
	ErrorLog   *log.Logger
}

// Middleware defines a function to process middleware.
type Middleware func(Context) error

// HTTPError represents an error that occurred while handling a request.
type HTTPError struct {
	error
	Code int
}

// NewHTTPError creates an instance of HTTPError with status code and error message.
func NewHTTPError(code int, err string) *HTTPError {
	return &HTTPError{errors.New(err), code}
}

// Handler is a interface defines a function work as middleware.
type Handler interface {
	Middleware(Context) error
}

// New creates an instance of Gear.
func New() *Gear {
	g := new(Gear)
	g.server = new(http.Server)
	g.middleware = make([]Middleware, 0)
	g.pool.New = func() interface{} {
		ctx := &gearCtx{}
		res := &Response{ctx: ctx}
		ctx.res = res
		return ctx
	}
	return g
}

func (g *Gear) toServHandler() *servHandler {
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
	g.server.Addr = addr
	g.server.Handler = g.toServHandler()
	if g.ErrorLog != nil {
		g.server.ErrorLog = g.ErrorLog
	}
	return g.server.ListenAndServe()
}

// ListenTLS starts the HTTPS server.
func (g *Gear) ListenTLS(addr, certFile, keyFile string) error {
	g.server.Addr = addr
	g.server.Handler = g.toServHandler()
	if g.ErrorLog != nil {
		g.server.ErrorLog = g.ErrorLog
	}
	return g.server.ListenAndServeTLS(certFile, keyFile)
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

	if !ctx.res.finished && err == nil {
		for _, handle := range ctx.hooks {
			if err = handle(ctx); err != nil {
				break
			}
			if ctx.res.finished {
				break // end up the process
			}
		}
	}

	if err != nil {
		h.app.OnError(err)
		if ctx.res.Status < 400 {
			ctx.Status(500)
		}
		ctx.Body(err.Error())
	}
	ctx.res.respond()
	h.app.pool.Put(ctx)
}
