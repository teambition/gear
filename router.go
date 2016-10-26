package gear

import (
	"fmt"
	"net/http"
	"strings"
)

// Router is a tire base HTTP request handler for Gear which can be used to
// dispatch requests to different handler functions
type Router struct {
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
		root:       root,
		trie:       t,
		middleware: make([]Middleware, 0),
	}
}

// Use registers a new Middleware handler in the router.
func (r *Router) Use(handle Middleware) {
	r.middleware = append(r.middleware, handle)
}

// Handle registers a new Middleware handler with method and path in the router.
func (r *Router) Handle(method, pattern string, handle Middleware) {
	if method == "" {
		panic(NewAppError("invalid method"))
	}
	if handle == nil {
		panic(NewAppError("invalid middleware"))
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
	var handle Middleware

	if !strings.HasPrefix(path, r.root) {
		return nil
	}

	res := r.trie.match(strings.TrimPrefix(path, r.root))
	if res.node == nil {
		if r.otherwise == nil {
			return &Error{Code: 501, Msg: fmt.Sprintf(`"%s" not implemented`, path)}
		}
		handle = r.otherwise
	} else {
		handle = res.node.methods[method]
		if handle == nil {
			// OPTIONS support
			if method == http.MethodOptions {
				ctx.Set(HeaderAllow, res.node.allowMethods)
				ctx.End(204)
				return nil
			}

			if r.otherwise == nil {
				// If no route handler is returned, it's a 405 error
				ctx.Set(HeaderAllow, res.node.allowMethods)
				return &Error{Code: 405, Msg: fmt.Sprintf(`"%s" not allowed in "%s"`, method, path)}
			}
			handle = r.otherwise
		}
	}

	if res.params != nil {
		ctx.SetAny(paramsKey, res.params)
	}
	err := r.run(ctx, handle)
	ctx.setEnd(true)
	return err
}

func (r *Router) run(ctx *Context, fn Middleware) (err error) {
	for _, handle := range r.middleware {
		if err = handle(ctx); err != nil {
			return
		}
		if ctx.IsEnded() {
			return // middleware and fn should not run if true
		}
	}
	return fn(ctx)
}

func normalizePath(path string) string {
	if !strings.Contains(path, "//") {
		return path
	}
	return normalizePath(strings.Replace(path, "//", "/", -1))
}
