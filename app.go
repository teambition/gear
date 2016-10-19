package gweb

import "net/http"

// Gweb docs
type Gweb struct {
	server *http.Server
	mds    []Middleware
}

type Middleware func(*Context) error

// New docs
func New() *Gweb {
	return &Gweb{mds: make([]Middleware, 0)}
}

// ToHandler docs
func (g *Gweb) ToHandler() *Handler {
	if len(g.mds) == 0 {
		panic("No middleware")
	}
	return &Handler{mds: g.mds[0:]}
}

// OnError docs
func (g *Gweb) OnError() {
}

// Use docs
func (g *Gweb) Use(fn Middleware) {
	g.mds = append(g.mds, fn)
}

// Listen docs
func (g *Gweb) Listen(addr string) error {
	if g.server == nil {
		g.server = &http.Server{}
	}
	g.server.Addr = addr
	g.server.Handler = g.ToHandler()
	return g.server.ListenAndServe()
}

// ListenTLS docs
func (g *Gweb) ListenTLS(addr, certFile, keyFile string) error {
	if g.server == nil {
		g.server = &http.Server{}
	}
	g.server.Addr = addr
	g.server.Handler = g.ToHandler()
	return g.server.ListenAndServeTLS(certFile, keyFile)
}

type Handler struct {
	mds []Middleware
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	c := NewContext(w, r)
	for _, fn := range h.mds {
		if err = fn(c); err != nil {
			break
		}
	}
	if err != nil {
		c.Status(500)
		c.Text(err.Error())
	}
	c.Res.respond()
}
