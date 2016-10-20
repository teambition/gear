package gweb

// Router docs, TODO
type Router struct {
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) GET(path string, handle Middleware) {
}

func (r *Router) HEAD(path string, handle Middleware) {
}

func (r *Router) OPTIONS(path string, handle Middleware) {
}

func (r *Router) POST(path string, handle Middleware) {
}

func (r *Router) PUT(path string, handle Middleware) {
}

func (r *Router) PATCH(path string, handle Middleware) {
}

func (r *Router) DELETE(path string, handle Middleware) {
}

func (r *Router) ToMiddleware() Middleware {
	return func(c *Context) error {
		return nil
	}
}
