package gear

import (
	"fmt"
	"net/http"
	"regexp"
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
	trie := &trie{
		ignoreCase: ignoreCase,
		root: &trieNode{
			parentNode:      nil,
			literalChildren: map[string]*trieNode{},
			methods:         map[string]Middleware{},
		},
	}
	if root == "" {
		root = "/"
	}
	return &Router{
		HandleOPTIONS: true,
		IsEndpoint:    true,
		root:          root,
		trie:          trie,
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

// Middleware implemented gear.Handler interface
func (r *Router) Middleware(ctx Context) error {
	path := ctx.Path()
	method := ctx.Method()

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
		ctx.End(204, "")
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

	ctx.SetValue(GearParamsKey, res.params)
	err := r.run(ctx, handle)
	if err == nil && r.IsEndpoint {
		ctx.End(0, "")
	}
	return err
}

func (r *Router) run(ctx Context, h Middleware) (err error) {
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

type trie struct {
	ignoreCase bool
	root       *trieNode
}

type trieNode struct {
	pattern      string
	allowMethods string
	methods      map[string]Middleware

	name            string
	endpoint        bool
	regex           *regexp.Regexp
	parentNode      *trieNode
	matchRemains    bool
	regexpChild     *trieNode
	literalChildren map[string]*trieNode
}

func (n *trieNode) handle(method string, handle Middleware) {
	if n.methods[method] != nil {
		panic("The route in \"" + n.pattern + "\" already defined")
	}
	n.methods[method] = handle
	if n.allowMethods == "" {
		n.allowMethods = method
	} else {
		n.allowMethods += ", " + method
	}
}

type trieMatched struct {
	node   *trieNode
	params map[string]string
}

func (t *trie) define(pattern string) *trieNode {
	if strings.Contains(pattern, "//") {
		panic("Multi-slash exist.")
	}

	_pattern := strings.Trim(pattern, "/")
	node := defineNode(t.root, strings.Split(_pattern, "/"), t.ignoreCase)

	if node.pattern == "" {
		node.pattern = pattern
	}
	return node
}

func (t *trie) match(path string) trieMatched {
	node := t.root
	frags := strings.Split(strings.Trim(path, "/"), "/")

	res := trieMatched{params: map[string]string{}}
	for i, frag := range frags {
		_frag := frag
		if t.ignoreCase && _frag != "" {
			_frag = strings.ToLower(frag)
		}
		named := false
		node, named = matchNode(node, _frag, res.params)
		if node == nil {
			return res
		}
		if named {
			if node.matchRemains {
				res.params[node.name] = strings.Join(frags[i:], "/")
				break
			} else {
				res.params[node.name] = frag
			}
		}
	}

	if node.endpoint {
		res.node = node
	}
	return res
}

func normalizePath(path string) string {
	if !strings.Contains(path, "//") {
		return path
	}
	return normalizePath(strings.Replace(path, "//", "/", -1))
}

func defineNode(parent *trieNode, frags []string, ignoreCase bool) *trieNode {
	frag := frags[0]
	frags = frags[1:]
	child := parseNode(parent, frag, ignoreCase)

	if len(frags) == 0 {
		child.endpoint = true
		return child
	}
	return defineNode(child, frags, ignoreCase)
}

func matchNode(parent *trieNode, frag string, params map[string]string) (child *trieNode, named bool) {
	if child = parent.literalChildren[frag]; child != nil {
		return
	}

	if child = parent.regexpChild; child != nil {
		if child.regex != nil && !child.regex.MatchString(frag) {
			child = nil
		} else {
			named = true
		}
	}
	return
}

func parseNode(parent *trieNode, frag string, ignoreCase bool) *trieNode {
	literalChildren := parent.literalChildren

	if literalChildren[frag] != nil {
		return literalChildren[frag]
	}

	node := &trieNode{
		parentNode:      parent,
		literalChildren: map[string]*trieNode{},
		methods:         map[string]Middleware{},
	}

	if frag != "" && frag[0] == ':' {
		var name, regex string
		name = frag[1:]
		trailing := name[len(name)-1]
		if trailing == ')' {
			if index := strings.IndexRune(name, '('); index > 0 {
				regex = name[index+1 : len(name)-1]
				if len(regex) > 0 {
					name = name[0:index]
					node.regex = regexp.MustCompile(regex)
				}
			}
		} else if trailing == '*' {
			name = name[0 : len(name)-1]
			node.matchRemains = true
		}
		if len(name) == 0 {
			panic(frag + "is invalid")
		}
		node.name = name
		if child := parent.regexpChild; child != nil {
			if child.name != name || child.matchRemains != node.matchRemains {
				panic(frag + "is invalid")
			}
			if child.regex != nil && child.regex.String() != regex {
				panic(frag + "is invalid")
			}
			return child
		}

		parent.regexpChild = node
	} else {
		literalChildren[frag] = node
	}

	return node
}
