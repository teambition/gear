package gear

import (
	"fmt"
	"net/http"
	"strings"
)

// Router is a tire base HTTP request handler for Gear which can be used to
// dispatch requests to different handler functions
type Router struct {
	// If enabled, the router automatically replies to OPTIONS requests.
	// Default to true
	HandleOPTIONS bool

	// If enabled, the router automatically replies to OPTIONS requests.
	// Default to true
	IsEndpoint bool

	root       string
	trie       *trie
	otherwise  Middleware
	middleware []Middleware
}

// NewRouter returns a new Router instance with root path and ignoreCase option.
func NewRouter(root string, ignoreCase bool) *Router {
	t := newTrie(ignoreCase)
	if root == "" {
		root = "/"
	}
	return &Router{
		HandleOPTIONS: true,
		IsEndpoint:    true,
		root:          root,
		trie:          t,
		middleware:    make([]Middleware, 0),
	}
}

// Use registers a new Middleware handler in the router.
func (r *Router) Use(handle Middleware) {
	r.middleware = append(r.middleware, handle)
}

// Handle registers a new Middleware handler with method and path in the router.
func (r *Router) Handle(method, pattern string, handle Middleware) {
	if method == "" {
		panic("Invalid method")
	}
	if handle == nil {
		panic("Invalid middleware")
	}
	r.trie.define(pattern).handle(strings.ToUpper(method), handle)
}

// Get registers a new GET route for a path with matching handler in the router.
func (r *Router) Get(pattern string, handle Middleware) {
	r.Handle(http.MethodGet, pattern, handle)
}

// Head registers a new HEAD route for a path with matching handler in the router.
func (r *Router) Head(pattern string, handle Middleware) {
	r.Handle(http.MethodHead, pattern, handle)
}

// Post registers a new POST route for a path with matching handler in the router.
func (r *Router) Post(pattern string, handle Middleware) {
	r.Handle(http.MethodPost, pattern, handle)
}

// Put registers a new PUT route for a path with matching handler in the router.
func (r *Router) Put(pattern string, handle Middleware) {
	r.Handle(http.MethodPut, pattern, handle)
}

// Patch registers a new PATCH route for a path with matching handler in the router.
func (r *Router) Patch(pattern string, handle Middleware) {
	r.Handle(http.MethodPatch, pattern, handle)
}

// Del registers a new DELETE route for a path with matching handler in the router.
func (r *Router) Del(pattern string, handle Middleware) {
	r.Handle(http.MethodDelete, pattern, handle)
}

// Delete registers a new DELETE route for a path with matching handler in the router.
func (r *Router) Delete(pattern string, handle Middleware) {
	r.Handle(http.MethodDelete, pattern, handle)
}

// Options registers a new OPTIONS route for a path with matching handler in the router.
func (r *Router) Options(pattern string, handle Middleware) {
	r.Handle(http.MethodOptions, pattern, handle)
}

// Otherwise registers a new Middleware handler in the router
// that will run if there is no other handler matching.
func (r *Router) Otherwise(handle Middleware) {
	r.otherwise = handle
}

// Serve implemented gear.Handler interface
func (r *Router) Serve(ctx *Context) error {
	path := ctx.Path
	method := ctx.Method

	if !strings.HasPrefix(path, r.root) {
		return nil
	}

	path = strings.TrimPrefix(path, r.root)
	res := r.trie.match(path)
	if res.node == nil {
		if r.otherwise != nil {
			r.run(ctx, r.otherwise)
		}
		ctx.Status(501)
		return fmt.Errorf("%s not implemented", path)
	}

	// OPTIONS support
	if method == http.MethodOptions && r.HandleOPTIONS {
		ctx.Set(HeaderAllow, res.node.allowMethods)
		ctx.End(204, nilByte)
		return nil
	}

	handle := res.node.methods[method]
	if handle == nil {
		if r.otherwise != nil {
			r.run(ctx, r.otherwise)
		}
		// If no route handler is returned, it's a 405 error
		ctx.Status(405)
		ctx.Set(HeaderAllow, res.node.allowMethods)
		return fmt.Errorf("%s not allowed in %s", method, path)
	}

	if res.params != nil {
		ctx.SetValue(GearParamsKey, res.params)
	}
	err := r.run(ctx, handle)
	if err == nil && r.IsEndpoint {
		ctx.End(0, nilByte)
	}
	return err
}

func (r *Router) run(ctx *Context, h Middleware) (err error) {
	for _, handle := range r.middleware {
		if err = handle(ctx); err != nil {
			return
		}
		if ctx.IsEnded() {
			return
		}
	}
	return h(ctx)
}

func normalizePath(path string) string {
	if !strings.Contains(path, "//") {
		return path
	}
	return normalizePath(strings.Replace(path, "//", "/", -1))
}
