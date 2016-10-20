package gweb

import (
	"log"
	"net/http"
)

// Gweb docs
type Gweb struct {
	server     *http.Server
	middleware []Middleware
	ErrorLog   *log.Logger
}

type Middleware func(*Context) error

type Handler interface {
	ToMiddleware() Middleware
}

// New docs
func New() *Gweb {
	g := new(Gweb)
	g.middleware = make([]Middleware, 0)
	g.server = &http.Server{}
	return g
}

// ToHandler docs
func (g *Gweb) toHandler() *servHandler {
	if len(g.middleware) == 0 {
		panic("No middleware")
	}
	return &servHandler{middleware: g.middleware[0:]}
}

// OnError docs
func (g *Gweb) OnError(err error) {
	if g.ErrorLog != nil {
		g.ErrorLog.Println(err)
	} else {
		log.Println(err)
	}
}

// Use docs
func (g *Gweb) Use(fn Middleware) {
	g.middleware = append(g.middleware, fn)
}

func (g *Gweb) UseHandler(h Handler) {
	g.middleware = append(g.middleware, h.ToMiddleware())
}

// Listen docs
func (g *Gweb) Listen(addr string) error {
	g.server.Addr = addr
	g.server.Handler = g.toHandler()
	if g.ErrorLog != nil {
		g.server.ErrorLog = g.ErrorLog
	}
	return g.server.ListenAndServe()
}

// ListenTLS docs
func (g *Gweb) ListenTLS(addr, certFile, keyFile string) error {
	g.server.Addr = addr
	g.server.Handler = g.toHandler()
	if g.ErrorLog != nil {
		g.server.ErrorLog = g.ErrorLog
	}
	return g.server.ListenAndServeTLS(certFile, keyFile)
}

type servHandler struct {
	app        *Gweb
	middleware []Middleware
}

func (h *servHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx := NewContext(w, r)

	for _, fn := range h.middleware {
		if err = fn(ctx); err != nil {
			break
		}
		if ctx.ended || ctx.Res.finished {
			break // end up the process
		}
	}

	if !ctx.Res.finished && err == nil {
		for _, fn := range ctx.hooks {
			if err = fn(ctx); err != nil {
				break
			}
			if ctx.Res.finished {
				break // end up the process
			}
		}
	}

	if err != nil {
		ctx.Res.end(500)
		h.app.OnError(err)
	} else {
		ctx.Res.end(0)
	}
}
