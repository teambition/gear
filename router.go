package gear

import (
	"fmt"
	"net/http"
	"strings"
)

// Router is a tire base HTTP request handler for Gear which can be used to
// dispatch requests to different handler functions.
// A trivial example is:
//
//  package main
//
//  import (
//  	"fmt"
//
//  	"github.com/teambition/gear"
//  )
//
//  func SomeRouterMiddleware(ctx *gear.Context) error {
//  	// do some thing.
//  	fmt.Println("Router middleware...")
//  	return nil
//  }
//
//  func ViewHello(ctx *gear.Context) error {
//  	return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//  }
//
//  func main() {
//  	app := gear.New()
//  	// Add app middleware
//  	app.Use(gear.NewDefaultLogger())
//
//  	router := gear.NewRouter()
//  	router.Use(SomeRouterMiddleware) // Add router middleware, optionally
//  	router.Get("/", ViewHello)
//
//  	app.UseHandler(router)
//  	app.Error(app.Listen(":3000"))
//  }
//
// The router matches incoming requests by the request method and the path.
// If a handle is registered for this path and method, the router delegates the
// request to that function.
//
// The registered path, against which the router matches incoming requests, can
// contain three types of parameters:
//
//  Syntax         Type
//  :name          named parameter
//  :name*         named with catch-all parameter
//  :name(regexp)  named with regexp parameter
//
// Named parameters are dynamic path segments. They match anything until the
// next '/' or the path end:
//
//  Path: /api/:type/:ID
//
//  Requests:
//   /api/user/123             match: type="user", ID="123"
//   /api/user                 no match
//   /api/user/123/comments    no match
//
// Named with catch-all parameters match anything until the path end, including the
// directory index (the '/' before the catch-all). Since they match anything
// until the end, catch-all parameters must always be the final path element.
//
//  Path: /files/:filepath*
//
//  Requests:
//   /files                              no match
//   /files/LICENSE                      match: filepath="LICENSE"
//   /files/templates/article.html       match: filepath="templates/article.html"
//
// Named with regexp parameters match anything using regexp until the
// next '/' or the path end:
//
//  Path: /api/:type/:ID(^\\d+$)
//
//  Requests:
//   /api/user/123             match: type="user", ID="123"
//   /api/user                 no match
//   /api/user/abc             no match
//   /api/user/123/comments    no match
//
// The value of parameters is saved on the gear.Context. Retrieve the value of a parameter by name:
//
//  type := ctx.Param("type")
//  id   := ctx.Param("ID")
//
type Router struct {
	root       string
	trie       *trie
	tsr        bool
	otherwise  []Middleware
	middleware []Middleware
}

// RouterOptions is options for Router
type RouterOptions struct {
	// Router's namespace. Gear supports multiple routers with different namespace.
	// Root string should start with "/", default to "/"
	Root string
	// Ignore case when matching URL path.
	IgnoreCase bool
	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	TrailingSlashRedirect bool
}

var defaultRouterOptions = RouterOptions{Root: "/", IgnoreCase: true, TrailingSlashRedirect: true}

// NewRouter returns a new Router instance with root path and ignoreCase option.
// Gear support multi-routers. For example:
//
//  // Create app
//  app := gear.New()
//
//  // Create views router
//  viewRouter := gear.NewRouter()
//  viewRouter.Get("/", Ctl.IndexView)
//  // add more ...
//
//  apiRouter := gear.NewRouter(RouterOptions{
//  	Root: "/api",
//  	IgnoreCase: true,
//  	TrailingSlashRedirect: true
//  })
//  // support one more middleware
//  apiRouter.Get("/user/:id", API.Auth, API.User)
//  // add more ..
//
//  app.UseHandler(apiRouter) // Must add apiRouter first.
//  app.UseHandler(viewRouter)
//  // Start app at 3000
//  app.Listen(":3000")
//
func NewRouter(routerOptions ...RouterOptions) *Router {
	opts := defaultRouterOptions
	if len(routerOptions) > 0 {
		opts = routerOptions[0]
	}
	if opts.Root == "" {
		opts.Root = "/"
	}

	return &Router{
		root:       opts.Root,
		trie:       newTrie(opts.IgnoreCase, opts.TrailingSlashRedirect),
		middleware: make([]Middleware, 0),
	}
}

// Use registers a new Middleware in the router, that will be called when router mathed.
func (r *Router) Use(handle Middleware) {
	r.middleware = append(r.middleware, handle)
}

// Handle registers a new Middleware handler with method and path in the router.
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Handle(method, pattern string, handlers ...Middleware) {
	if method == "" {
		panic(NewAppError("invalid method"))
	}
	if len(handlers) == 0 {
		panic(NewAppError("invalid middleware"))
	}
	r.trie.define(pattern).handle(strings.ToUpper(method), handlers)
}

// Get registers a new GET route for a path with matching handler in the router.
func (r *Router) Get(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodGet, pattern, handlers...)
}

// Head registers a new HEAD route for a path with matching handler in the router.
func (r *Router) Head(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodHead, pattern, handlers...)
}

// Post registers a new POST route for a path with matching handler in the router.
func (r *Router) Post(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodPost, pattern, handlers...)
}

// Put registers a new PUT route for a path with matching handler in the router.
func (r *Router) Put(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodPut, pattern, handlers...)
}

// Patch registers a new PATCH route for a path with matching handler in the router.
func (r *Router) Patch(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodPatch, pattern, handlers...)
}

// Delete registers a new DELETE route for a path with matching handler in the router.
func (r *Router) Delete(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodDelete, pattern, handlers...)
}

// Options registers a new OPTIONS route for a path with matching handler in the router.
func (r *Router) Options(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodOptions, pattern, handlers...)
}

// Otherwise registers a new Middleware handler in the router
// that will run if there is no other handler matching.
func (r *Router) Otherwise(handlers ...Middleware) {
	if len(handlers) == 0 {
		panic(NewAppError("invalid middleware"))
	}
	r.otherwise = handlers
}

// Serve implemented gear.Handler interface
func (r *Router) Serve(ctx *Context) error {
	path := ctx.Path
	method := ctx.Method
	var handlers []Middleware

	if !strings.HasPrefix(path, r.root) {
		return nil
	}

	res := r.trie.match(strings.TrimPrefix(path, r.root))
	// TrailingSlashRedirect
	if res.tsr && len(ctx.Path) > 1 {
		if ctx.Path[len(ctx.Path)-1] == '/' {
			ctx.Req.URL.Path = ctx.Path[:len(ctx.Path)-1]
		} else {
			ctx.Req.URL.Path = ctx.Path + "/"
		}

		code := 301
		if ctx.Method != "GET" {
			code = 307
		}
		return ctx.Redirect(code, ctx.Req.URL.String())
	}

	if res.node == nil {
		if r.otherwise == nil {
			return &Error{Code: 501, Msg: fmt.Sprintf(`"%s" not implemented`, path)}
		}
		handlers = r.otherwise
	} else {
		handlers = res.node.methods[method]
		if len(handlers) == 0 {
			// OPTIONS support
			if method == http.MethodOptions {
				ctx.Set(HeaderAllow, res.node.allowMethods)
				return ctx.End(204)
			}

			if r.otherwise == nil {
				// If no route handler is returned, it's a 405 error
				ctx.Set(HeaderAllow, res.node.allowMethods)
				return &Error{Code: 405, Msg: fmt.Sprintf(`"%s" not allowed in "%s"`, method, path)}
			}
			handlers = r.otherwise
		}
	}

	if res.params != nil {
		ctx.SetAny(paramsKey, res.params)
	}
	err := r.run(ctx, handlers)
	ctx.ended = true
	return err
}

func (r *Router) run(ctx *Context, handlers []Middleware) (err error) {
	for _, handle := range r.middleware {
		if err = handle(ctx); !isNil(err) {
			return
		}
		if ctx.ended {
			return // middleware and fn should not run if true
		}
	}

	for _, handle := range handlers {
		if err = handle(ctx); !isNil(err) {
			return
		}
		if ctx.ended {
			return // middleware and fn should not run if true
		}
	}
	return
}
