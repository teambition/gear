package gear

import (
	"fmt"
	"regexp"
	"strings"
)

var wordReg = regexp.MustCompile("^\\w+$")
var doubleColonReg = regexp.MustCompile("^::\\w*$")

func newTrie(ignoreCase bool) *trie {
	return &trie{
		ignoreCase: ignoreCase,
		root: &trieNode{
			parentNode:      nil,
			literalChildren: map[string]*trieNode{},
			methods:         map[string]Middleware{},
		},
	}
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
	wildcard        bool
	varyChild       *trieNode
	literalChildren map[string]*trieNode
}

func (n *trieNode) handle(method string, handle Middleware) {
	if n.methods[method] != nil {
		panic(NewAppError(fmt.Sprintf("the route in %s already defined", n.pattern)))
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
		panic(NewAppError(fmt.Sprintf("multi-slash exist: %s", pattern)))
	}

	_pattern := strings.Trim(pattern, "/")
	node := defineNode(t.root, strings.Split(_pattern, "/"), t.ignoreCase)

	if node.pattern == "" {
		node.pattern = pattern
	}
	return node
}

func (t *trie) match(path string) *trieMatched {
	node := t.root
	frags := strings.Split(strings.Trim(path, "/"), "/")

	res := &trieMatched{}
	for i, frag := range frags {
		_frag := frag
		if t.ignoreCase && _frag != "" {
			_frag = strings.ToLower(frag)
		}
		named := false
		node, named = matchNode(node, _frag)
		if node == nil {
			return res
		}
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

	if node.endpoint {
		res.node = node
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

	if literalChildren[_frag] != nil {
		return literalChildren[_frag]
	}

	node := &trieNode{
		parentNode:      parent,
		literalChildren: map[string]*trieNode{},
		methods:         map[string]Middleware{},
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
		literalChildren[frag] = node
	}

	return node
}
