package gear

import (
	"net/http"
	"strings"

	"github.com/teambition/trie-mux"
)

// Router is a trie base HTTP request handler for Gear which can be used to
// dispatch requests to different handler functions.
// A trivial example is:
//
//	package main
//
//	import (
//		"fmt"
//
//		"github.com/teambition/gear"
//	)
//
//	func SomeRouterMiddleware(ctx *gear.Context) error {
//		// do some thing.
//		fmt.Println("Router middleware...")
//		return nil
//	}
//
//	func ViewHello(ctx *gear.Context) error {
//		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//	}
//
//	func main() {
//		app := gear.New()
//		// Add app middleware
//
//		router := gear.NewRouter()
//		router.Use(SomeRouterMiddleware) // Add router middleware, optionally
//		router.Get("/", ViewHello)
//
//		app.UseHandler(router)
//		app.Error(app.Listen(":3000"))
//	}
//
// The router matches incoming requests by the request method and the path.
// If a handle is registered for this path and method, the router delegates the
// request to that function.
//
// The registered path, against which the router matches incoming requests, can
// contain six types of parameters:
//
//	| Syntax | Description |
//	|--------|------|
//	| `:name` | named parameter |
//	| `:name(regexp)` | named with regexp parameter |
//	| `:name+suffix` | named parameter with suffix matching |
//	| `:name(regexp)+suffix` | named with regexp parameter and suffix matching |
//	| `:name*` | named with catch-all parameter |
//	| `::name` | not named parameter, it is literal `:name` |
//
// Named parameters are dynamic path segments. They match anything until the next '/' or the path end:
//
// Defined: `/api/:type/:ID`
//
//	/api/user/123             matched: type="user", ID="123"
//	/api/user                 no match
//	/api/user/123/comments    no match
//
// Named with regexp parameters match anything using regexp until the next '/' or the path end:
//
// Defined: `/api/:type/:ID(^\d+$)`
//
//	/api/user/123             matched: type="user", ID="123"
//	/api/user                 no match
//	/api/user/abc             no match
//	/api/user/123/comments    no match
//
// Named parameters with suffix, such as [Google API Design](https://cloud.google.com/apis/design/custom_methods):
//
// Defined: `/api/:resource/:ID+:undelete`
//
//	/api/file/123                     no match
//	/api/file/123:undelete            matched: resource="file", ID="123"
//	/api/file/123:undelete/comments   no match
//
// Named with regexp parameters and suffix:
//
// Defined: `/api/:resource/:ID(^\d+$)+:cancel`
//
//	/api/task/123                   no match
//	/api/task/123:cancel            matched: resource="task", ID="123"
//	/api/task/abc:cancel            no match
//
// Named with catch-all parameters match anything until the path end, including the directory index (the '/' before the catch-all). Since they match anything until the end, catch-all parameters must always be the final path element.
//
// Defined: `/files/:filepath*`
//
//	/files                           no match
//	/files/LICENSE                   matched: filepath="LICENSE"
//	/files/templates/article.html    matched: filepath="templates/article.html"
//
// The value of parameters is saved on the `Matched.Params`. Retrieve the value of a parameter by name:
//
//	type := matched.Params("type")
//	id   := matched.Params("ID")
//
// More info: https://github.com/teambition/trie-mux
type Router struct {
	root       string
	rt         string
	trie       *trie.Trie
	otherwise  Middleware
	middleware Middleware
	mds        []Middleware
}

// RouterOptions is options for Router
type RouterOptions struct {
	// Router's namespace. Gear supports multiple routers with different namespace.
	// Root string should start with "/", default to "/"
	Root string

	// Ignore case when matching URL path.
	IgnoreCase bool

	// Enables automatic redirection if the current path can't be matched but
	// a handler for the fixed path exists.
	// For example if "/api//foo" is requested but a route only exists for "/api/foo", the
	// client is redirected to "/api/foo"" with http status code 301 for GET requests
	// and 307 for all other request methods.
	FixedPathRedirect bool

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if "/foo/" is requested but a route only exists for "/foo", the
	// client is redirected to "/foo"" with http status code 301 for GET requests
	// and 307 for all other request methods.
	TrailingSlashRedirect bool
}

var defaultRouterOptions = RouterOptions{
	Root:                  "/",
	IgnoreCase:            true,
	FixedPathRedirect:     true,
	TrailingSlashRedirect: true,
}

// NewRouter returns a new Router instance with root path and ignoreCase option.
// Gear support multi-routers. For example:
//
//	// Create app
//	app := gear.New()
//
//	// Create views router
//	viewRouter := gear.NewRouter()
//	viewRouter.Get("/", Ctl.IndexView)
//	// add more ...
//
//	apiRouter := gear.NewRouter(RouterOptions{
//		Root: "/api",
//		IgnoreCase: true,
//		FixedPathRedirect: true,
//		TrailingSlashRedirect: true,
//	})
//	// support one more middleware
//	apiRouter.Get("/user/:id", API.Auth, API.User)
//	// add more ..
//
//	app.UseHandler(apiRouter) // Must add apiRouter first.
//	app.UseHandler(viewRouter)
//	// Start app at 3000
//	app.Listen(":3000")
func NewRouter(routerOptions ...RouterOptions) *Router {
	opts := defaultRouterOptions
	if len(routerOptions) > 0 {
		opts = routerOptions[0]
	}
	if opts.Root == "" || opts.Root[len(opts.Root)-1] != '/' {
		opts.Root += "/"
	}

	return &Router{
		root: opts.Root,
		rt:   opts.Root[0 : len(opts.Root)-1],
		mds:  make([]Middleware, 0),
		trie: trie.New(trie.Options{
			IgnoreCase:            opts.IgnoreCase,
			FixedPathRedirect:     opts.FixedPathRedirect,
			TrailingSlashRedirect: opts.TrailingSlashRedirect,
		}),
	}
}

// Use registers a new Middleware in the router, that will be called when router mathed.
func (r *Router) Use(handle Middleware) *Router {
	r.mds = append(r.mds, handle)
	r.middleware = Compose(r.mds...)
	return r
}

// Handle registers a new Middleware handler with method and path in the router.
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Handle(method, pattern string, handlers ...Middleware) *Router {
	if method == "" {
		panic(Err.WithMsg("invalid method"))
	}
	if len(handlers) == 0 {
		panic(Err.WithMsg("invalid middleware"))
	}
	r.trie.Define(pattern).Handle(strings.ToUpper(method), Compose(handlers...))
	return r
}

// Get registers a new GET route for a path with matching handler in the router.
func (r *Router) Get(pattern string, handlers ...Middleware) *Router {
	return r.Handle(http.MethodGet, pattern, handlers...)
}

// Head registers a new HEAD route for a path with matching handler in the router.
func (r *Router) Head(pattern string, handlers ...Middleware) *Router {
	return r.Handle(http.MethodHead, pattern, handlers...)
}

// Post registers a new POST route for a path with matching handler in the router.
func (r *Router) Post(pattern string, handlers ...Middleware) *Router {
	return r.Handle(http.MethodPost, pattern, handlers...)
}

// Put registers a new PUT route for a path with matching handler in the router.
func (r *Router) Put(pattern string, handlers ...Middleware) *Router {
	return r.Handle(http.MethodPut, pattern, handlers...)
}

// Patch registers a new PATCH route for a path with matching handler in the router.
func (r *Router) Patch(pattern string, handlers ...Middleware) *Router {
	return r.Handle(http.MethodPatch, pattern, handlers...)
}

// Delete registers a new DELETE route for a path with matching handler in the router.
func (r *Router) Delete(pattern string, handlers ...Middleware) *Router {
	return r.Handle(http.MethodDelete, pattern, handlers...)
}

// Options registers a new OPTIONS route for a path with matching handler in the router.
func (r *Router) Options(pattern string, handlers ...Middleware) *Router {
	return r.Handle(http.MethodOptions, pattern, handlers...)
}

// Otherwise registers a new Middleware handler in the router
// that will run if there is no other handler matching.
func (r *Router) Otherwise(handlers ...Middleware) *Router {
	if len(handlers) == 0 {
		panic(Err.WithMsg("invalid middleware"))
	}
	r.otherwise = Compose(handlers...)
	return r
}

// Serve implemented gear.Handler interface
func (r *Router) Serve(ctx *Context) error {
	path := ctx.Path
	method := ctx.Method
	var handler Middleware

	if !strings.HasPrefix(path, r.root) && path != r.rt {
		return nil
	}

	if path == r.rt {
		path = "/"
	} else if l := len(r.rt); l > 0 {
		path = path[l:]
	}

	matched := r.trie.Match(path)

	if matched.Node == nil {
		// FixedPathRedirect or TrailingSlashRedirect
		if matched.TSR != "" || matched.FPR != "" {
			ctx.Req.URL.Path = matched.TSR
			if matched.FPR != "" {
				ctx.Req.URL.Path = matched.FPR
			}
			if len(r.root) > 1 {
				ctx.Req.URL.Path = r.root + ctx.Req.URL.Path[1:]
			}

			code := http.StatusMovedPermanently
			if method != "GET" {
				code = http.StatusTemporaryRedirect
			}
			ctx.Status(code)
			return ctx.Redirect(ctx.Req.URL.String())
		}

		if r.otherwise == nil {
			return nil
		}
		handler = r.otherwise
	} else {
		ok := false
		if handler, ok = matched.Node.GetHandler(method).(Middleware); !ok {
			// OPTIONS support
			if method == http.MethodOptions {
				ctx.SetHeader(HeaderAllow, matched.Node.GetAllow())
				return ctx.End(http.StatusNoContent)
			}

			if r.otherwise == nil {
				// If no route handler is returned, it's a 405 error
				ctx.SetHeader(HeaderAllow, matched.Node.GetAllow())
				return ErrMethodNotAllowed.WithMsgf(`"%s" is not allowed in "%s"`, method, ctx.Path)
			}
			handler = r.otherwise
		}
	}

	ctx.SetAny(paramsKey, matched.Params)
	ctx.SetAny(routerNodeKey, matched.Node)
	ctx.SetAny(routerRootKey, r.rt)
	if len(r.mds) > 0 {
		handler = Compose(r.middleware, handler)
	}
	return handler(ctx)
}

// GetRouterNodeFromCtx returns matched Node from router
//
//	router.Get("/api/:type/:ID", func(ctx *Context) error {
//		assert.Equal("/api/:type/:ID", GetRouterNodeFromCtx(ctx).GetPattern())
//		return ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
//	})
func GetRouterNodeFromCtx(ctx *Context) *trie.Node {
	if res, _ := ctx.Any(routerNodeKey); res != nil {
		return res.(*trie.Node)
	}
	return nil
}

// GetRouterPatternFromCtx returns matched Node from router
//
//	routerV2.Get("/api/:type/:ID", func(ctx *Context) error {
//		assert.Equal("/v2/api/:type/:ID", GetRouterPatternFromCtx(ctx))
//		return ctx.HTML(200, "ok")
//	})
func GetRouterPatternFromCtx(ctx *Context) string {
	if node := GetRouterNodeFromCtx(ctx); node != nil {
		rt := ctx.MustAny(routerRootKey)
		return rt.(string) + node.GetPattern()
	}
	return ""
}
