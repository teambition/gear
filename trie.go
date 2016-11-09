package gear

import (
	"fmt"
	"regexp"
	"strings"
)

var wordReg = regexp.MustCompile("^\\w+$")
var doubleColonReg = regexp.MustCompile("^::\\w*$")

// newTrie(ignoreCase, trailingSlashRedirect)
// newTrie(ignoreCase)
// newTrie()
func newTrie(args ...bool) *trie {
	// Ignore case when matching URL path.
	ignoreCase := true
	// Check if the current route can't be matched but a handler
	// for the path with (without) the trailing slash exists.
	trailingSlashRedirect := false
	if len(args) > 0 {
		ignoreCase = args[0]
	}
	if len(args) > 1 {
		trailingSlashRedirect = args[1]
	}
	return &trie{
		ignoreCase: ignoreCase,
		tsr:        trailingSlashRedirect,
		root: &trieNode{
			parentNode:      nil,
			literalChildren: map[string]*trieNode{},
			methods:         map[string][]Middleware{},
		},
	}
}

type trie struct {
	ignoreCase bool
	tsr        bool
	root       *trieNode
}

type trieNode struct {
	pattern      string
	allowMethods string
	methods      map[string][]Middleware

	name            string
	endpoint        bool
	regex           *regexp.Regexp
	parentNode      *trieNode
	wildcard        bool
	varyChild       *trieNode
	literalChildren map[string]*trieNode
}

func (n *trieNode) handle(method string, handlers []Middleware) {
	if n.methods[method] != nil {
		panic(NewAppError(fmt.Sprintf("the route in %s already defined", n.pattern)))
	}
	n.methods[method] = handlers
	if n.allowMethods == "" {
		n.allowMethods = method
	} else {
		n.allowMethods += ", " + method
	}
}

type trieMatched struct {
	node   *trieNode
	params map[string]string
	tsr    bool
}

func (t *trie) define(pattern string) *trieNode {
	if strings.Contains(pattern, "//") {
		panic(NewAppError(fmt.Sprintf("multi-slash exist: %s", pattern)))
	}

	_pattern := strings.TrimPrefix(pattern, "/")
	node := defineNode(t.root, strings.Split(_pattern, "/"), t.ignoreCase)

	if node.pattern == "" {
		node.pattern = pattern
	}
	return node
}

// path should not contains multi-slash
func (t *trie) match(path string) *trieMatched {
	parent := t.root
	frags := strings.Split(strings.TrimPrefix(path, "/"), "/")

	res := &trieMatched{}
	for i, frag := range frags {
		_frag := frag
		if t.ignoreCase {
			_frag = strings.ToLower(frag)
		}

		node, named := matchNode(parent, _frag)
		if node == nil {
			// TrailingSlashRedirect: /acb/efg/ -> /acb/efg
			if t.tsr && frag == "" && len(frags) == (i+1) && parent.endpoint {
				res.tsr = true
			}
			return res
		}
		parent = node

		if named {
			if res.params == nil {
				res.params = map[string]string{}
			}
			if node.wildcard {
				res.params[node.name] = strings.Join(frags[i:], "/")
				break
			} else {
				res.params[node.name] = frag
			}
		}
	}

	if parent.endpoint {
		res.node = parent
	} else if t.tsr && parent.literalChildren[""] != nil {
		// TrailingSlashRedirect: /acb/efg -> /acb/efg/
		res.tsr = true
	}
	return res
}

func defineNode(parent *trieNode, frags []string, ignoreCase bool) *trieNode {
	frag := frags[0]
	frags = frags[1:]
	child := parseNode(parent, frag, ignoreCase)

	if len(frags) == 0 {
		child.endpoint = true
		return child
	} else if child.wildcard {
		panic(NewAppError(fmt.Sprintf("can't define pattern after wildcard: %s", child.pattern)))
	}
	return defineNode(child, frags, ignoreCase)
}

func matchNode(parent *trieNode, frag string) (child *trieNode, named bool) {
	if child = parent.literalChildren[frag]; child != nil {
		return
	}

	if child = parent.varyChild; child != nil {
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

	_frag := frag
	if doubleColonReg.MatchString(frag) {
		_frag = frag[1:]
	}
	if ignoreCase {
		_frag = strings.ToLower(_frag)
	}

	if literalChildren[_frag] != nil {
		return literalChildren[_frag]
	}

	node := &trieNode{
		parentNode:      parent,
		literalChildren: map[string]*trieNode{},
		methods:         map[string][]Middleware{},
	}

	if frag == "" {
		literalChildren[frag] = node
	} else if doubleColonReg.MatchString(frag) {
		// pattern "/a/::" should match "/a/:"
		// pattern "/a/::bc" should match "/a/:bc"
		// pattern "/a/::/bc" should match "/a/:/bc"
		literalChildren[_frag] = node
	} else if frag[0] == ':' {
		var name, regex string
		name = frag[1:]
		trailing := name[len(name)-1]
		if trailing == ')' {
			if index := strings.IndexRune(name, '('); index > 0 {
				regex = name[index+1 : len(name)-1]
				if len(regex) > 0 {
					name = name[0:index]
					node.regex = regexp.MustCompile(regex)
				} else {
					panic(NewAppError(fmt.Sprintf("invalid pattern: %s", frag)))
				}
			}
		} else if trailing == '*' {
			name = name[0 : len(name)-1]
			node.wildcard = true
		}
		// name must be word characters `[0-9A-Za-z_]`
		if !wordReg.MatchString(name) {
			panic(NewAppError(fmt.Sprintf("invalid pattern: %s", frag)))
		}
		node.name = name
		if child := parent.varyChild; child != nil {
			if child.name != name || child.wildcard != node.wildcard {
				panic(NewAppError(fmt.Sprintf("invalid pattern: %s", frag)))
			}
			if child.regex != nil && child.regex.String() != regex {
				panic(NewAppError(fmt.Sprintf("invalid pattern: %s", frag)))
			}
			return child
		}

		parent.varyChild = node
	} else if frag[0] == '*' || frag[0] == '(' || frag[0] == ')' {
		panic(NewAppError(fmt.Sprintf("invalid pattern: %s", frag)))
	} else {
		literalChildren[_frag] = node
	}

	return node
}
