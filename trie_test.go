package gear

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGearTrie(t *testing.T) {
	t.Run("trie.define", func(t *testing.T) {
		t.Run("root pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			tr2 := newTrie()
			node := tr1.define("/")
			assert.Equal(node.name, "")

			EqualPtr(t, node, tr1.define("/"))
			EqualPtr(t, node, tr1.define(""))
			NotEqualPtr(t, node, tr2.define("/"))
			NotEqualPtr(t, node, tr2.define(""))

			EqualPtr(t, node.parentNode, tr1.root)
		})

		t.Run("simple pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/a/b")
			assert.Equal(node.name, "")

			EqualPtr(t, node, tr1.define("/a/b"))
			NotEqualPtr(t, node, tr1.define("a/b/"))
			NotEqualPtr(t, node, tr1.define("/a/b/"))
			EqualPtr(t, tr1.define("/a/b/"), tr1.define("a/b/"))
			assert.Equal(node.pattern, "/a/b")

			parent := tr1.define("/a")
			EqualPtr(t, node.parentNode, parent)
			NotEqualPtr(t, parent.varyChild, node)
			EqualPtr(t, parent.literalChildren["b"], node)
			child := tr1.define("/a/b/c")
			EqualPtr(t, child.parentNode, node)
			EqualPtr(t, node.literalChildren["c"], child)

			assert.Panics(func() {
				tr1.define("/a//b")
			})
		})

		t.Run("double colon pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/a/::b")
			assert.Equal(node.name, "")
			NotEqualPtr(t, node, tr1.define("/a/::"))
			NotEqualPtr(t, node, tr1.define("/a/::x"))

			parent := tr1.define("/a")
			EqualPtr(t, node.parentNode, parent)
			NotEqualPtr(t, parent.varyChild, node)
			EqualPtr(t, parent.literalChildren[":"], tr1.define("/a/::"))
			EqualPtr(t, parent.literalChildren[":b"], tr1.define("/a/::b"))
			EqualPtr(t, parent.literalChildren[":x"], tr1.define("/a/::x"))

			child := tr1.define("/a/::b/c")
			EqualPtr(t, child.parentNode, node)
			EqualPtr(t, node.literalChildren["c"], child)
		})

		t.Run("named pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()

			assert.Panics(func() {
				tr1.define("/a/:")
			})
			assert.Panics(func() {
				tr1.define("/a/:/")
			})
			assert.Panics(func() {
				tr1.define("/a/:abc$/")
			})
			node := tr1.define("/a/:b")
			assert.Equal(node.name, "b")
			assert.False(node.wildcard)
			assert.Nil(node.varyChild)
			assert.Equal(node.pattern, "/a/:b")
			assert.Panics(func() {
				tr1.define("/a/:x")
			})

			parent := tr1.define("/a")
			assert.Equal(parent.name, "")
			EqualPtr(t, parent.varyChild, node)
			EqualPtr(t, node.parentNode, parent)
			child := tr1.define("/a/:b/c")
			EqualPtr(t, child.parentNode, node)
			assert.Panics(func() {
				tr1.define("/a/:x/c")
			})
		})

		t.Run("wildcard pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			assert.Panics(func() {
				tr1.define("/a/*")
			})
			assert.Panics(func() {
				tr1.define("/a/:*")
			})
			assert.Panics(func() {
				tr1.define("/a/:#*")
			})
			assert.Panics(func() {
				tr1.define("/a/:abc(*")
			})

			node := tr1.define("/a/:b*")
			assert.Equal(node.name, "b")
			assert.True(node.wildcard)
			assert.Nil(node.varyChild)
			assert.Equal(node.pattern, "/a/:b*")
			assert.Panics(func() {
				tr1.define("/a/:x*")
			})

			parent := tr1.define("/a")
			assert.Equal(parent.name, "")
			assert.False(parent.wildcard)
			EqualPtr(t, parent.varyChild, node)
			EqualPtr(t, node.parentNode, parent)
			assert.Panics(func() {
				tr1.define("/a/:b*/c")
			})
		})

		t.Run("regexp pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			assert.Panics(func() {
				tr1.define("/a/(")
			})
			assert.Panics(func() {
				tr1.define("/a/)")
			})
			assert.Panics(func() {
				tr1.define("/a/:(")
			})
			assert.Panics(func() {
				tr1.define("/a/:)")
			})
			assert.Panics(func() {
				tr1.define("/a/:()")
			})
			assert.Panics(func() {
				tr1.define("/a/:bc)")
			})
			assert.Panics(func() {
				tr1.define("/a/:(bc)")
			})
			assert.Panics(func() {
				tr1.define("/a/:#(bc)")
			})
			assert.Panics(func() {
				tr1.define("/a/:b(c)*")
			})

			node := tr1.define("/a/:b(x|y|z)")
			assert.Equal(node.name, "b")
			assert.Equal(node.pattern, "/a/:b(x|y|z)")
			assert.False(node.wildcard)
			assert.Nil(node.varyChild)
			assert.Equal(node, tr1.define("/a/:b(x|y|z)"))
			assert.Panics(func() {
				tr1.define("/a/:x(x|y|z)")
			})

			parent := tr1.define("/a")
			assert.Equal(parent.name, "")
			assert.False(parent.wildcard)
			EqualPtr(t, parent.varyChild, node)
			EqualPtr(t, node.parentNode, parent)

			child := tr1.define("/a/:b(x|y|z)/c")
			EqualPtr(t, child.parentNode, node)
			assert.Panics(func() {
				tr1.define("/a/:x(x|y|z)/c")
			})
		})

		t.Run("ignoreCase option", func(t *testing.T) {
			tr := newTrie(true)
			node := tr.define("/A/b")
			EqualPtr(t, node, tr.define("/a/b"))
			EqualPtr(t, node, tr.define("/a/B"))

			node = tr.define("/::A/b")
			EqualPtr(t, node, tr.define("/::a/b"))

			tr = newTrie(false)
			node = tr.define("/A/b")
			NotEqualPtr(t, node, tr.define("/a/b"))
			NotEqualPtr(t, node, tr.define("/a/B"))

			node = tr.define("/::A/b")
			NotEqualPtr(t, node, tr.define("/::a/b"))
		})
	})

	t.Run("trie.match", func(t *testing.T) {
		t.Run("root pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/")
			res := tr1.match("/")
			assert.Nil(res.params)
			EqualPtr(t, node, res.node)

			res2 := tr1.match("")
			EqualPtr(t, node, res2.node)
			NotEqualPtr(t, res, res2)

			assert.Nil(tr1.match("/a").node)
		})

		t.Run("simple pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/a/b")
			res := tr1.match("/a/b")
			assert.Nil(res.params)
			EqualPtr(t, node, res.node)

			assert.Nil(tr1.match("/a").node)
			assert.Nil(tr1.match("/a/b/c").node)
			assert.Nil(tr1.match("/a/x/c").node)

			assert.Panics(func() {
				tr1.match("/a//b")
			})
		})

		t.Run("double colon pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/a/::b")
			res := tr1.match("/a/:b")
			assert.Nil(res.params)
			EqualPtr(t, node, res.node)
			assert.Nil(tr1.match("/a").node)
			assert.Nil(tr1.match("/a/::b").node)

			node = tr1.define("/a/::b/c")
			res = tr1.match("/a/:b/c")
			assert.Nil(res.params)
			EqualPtr(t, node, res.node)
			assert.Nil(tr1.match("/a/::b/c").node)

			node = tr1.define("/a/::")
			res = tr1.match("/a/:")
			assert.Nil(res.params)
			EqualPtr(t, node, res.node)
			assert.Nil(tr1.match("/a/::").node)
		})

		t.Run("named pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/a/:b")
			res := tr1.match("/a/xyz汉")
			assert.Equal("xyz汉", res.params["b"])
			assert.Equal("", res.params["x"])
			EqualPtr(t, node, res.node)
			assert.Nil(tr1.match("/a").node)
			assert.Nil(tr1.match("/a/xyz汉/123").node)

			node2 := tr1.define("/:a/:b")
			res2 := tr1.match("/a/xyz汉")
			EqualPtr(t, node, res2.node)

			res2 = tr1.match("/ab/xyz汉")
			assert.Equal("xyz汉", res2.params["b"])
			assert.Equal("ab", res2.params["a"])
			EqualPtr(t, node2, res2.node)
			assert.Nil(tr1.match("/ab").node)
			assert.Nil(tr1.match("/ab/xyz汉/123").node)
		})

		t.Run("wildcard pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/a/:b*")
			res := tr1.match("/a/xyz汉")
			assert.Equal("xyz汉", res.params["b"])
			EqualPtr(t, node, res.node)
			assert.Nil(tr1.match("/a").node)

			res = tr1.match("/a/xyz汉/123")
			assert.Equal("xyz汉/123", res.params["b"])
			EqualPtr(t, node, res.node)

			node = tr1.define("/:a*")
			assert.Nil(tr1.match("/a").node) // TODO
			res = tr1.match("/123")
			assert.Equal("123", res.params["a"])
			EqualPtr(t, node, res.node)
			res = tr1.match("/123/xyz汉")
			assert.Equal("123/xyz汉", res.params["a"])
			EqualPtr(t, node, res.node)
		})

		t.Run("regexp pattern", func(t *testing.T) {
			assert := assert.New(t)

			tr1 := newTrie()
			node := tr1.define("/a/:b(^(x|y|z)$)")
			res := tr1.match("/a/x")
			assert.Equal("x", res.params["b"])
			EqualPtr(t, node, res.node)
			res = tr1.match("/a/y")
			assert.Equal("y", res.params["b"])
			EqualPtr(t, node, res.node)
			res = tr1.match("/a/z")
			assert.Equal("z", res.params["b"])
			EqualPtr(t, node, res.node)

			assert.Nil(tr1.match("/a").node)
			assert.Nil(tr1.match("/a/xy").node)
			assert.Nil(tr1.match("/a/x/y").node)

			child := tr1.define("/a/:b(^(x|y|z)$)/c")
			res = tr1.match("/a/x/c")
			assert.Equal("x", res.params["b"])
			EqualPtr(t, child, res.node)
			res = tr1.match("/a/y/c")
			assert.Equal("y", res.params["b"])
			EqualPtr(t, child, res.node)
			res = tr1.match("/a/z/c")
			assert.Equal("z", res.params["b"])
			EqualPtr(t, child, res.node)
		})

		t.Run("ignoreCase option", func(t *testing.T) {
			assert := assert.New(t)

			// ignoreCase = true
			tr := newTrie(true)
			node := tr.define("/A/:Name")

			res := tr.match("/a/x")
			EqualPtr(t, node, res.node)
			assert.Equal("x", res.params["Name"])
			assert.Equal("", res.params["name"])

			res = tr.match("/A/X")
			EqualPtr(t, node, res.node)
			assert.Equal("X", res.params["Name"])
			assert.Equal("", res.params["name"])

			node = tr.define("/::A/:Name")

			res = tr.match("/:a/x")
			EqualPtr(t, node, res.node)
			assert.Equal("x", res.params["Name"])
			assert.Equal("", res.params["name"])

			res = tr.match("/:A/X")
			EqualPtr(t, node, res.node)
			assert.Equal("X", res.params["Name"])
			assert.Equal("", res.params["name"])

			// ignoreCase = false
			tr = newTrie(false)
			node = tr.define("/A/:Name")

			assert.Nil(tr.match("/a/x").node)
			res = tr.match("/A/X")
			EqualPtr(t, node, res.node)
			assert.Equal("X", res.params["Name"])

			node = tr.define("/::A/:Name")

			assert.Nil(tr.match("/:a/x").node)
			res = tr.match("/:A/X")
			EqualPtr(t, node, res.node)
			assert.Equal("X", res.params["Name"])
			assert.Equal("", res.params["name"])
		})

		t.Run("trailingSlashRedirect option", func(t *testing.T) {
			assert := assert.New(t)

			// trailingSlashRedirect = false
			tr := newTrie(true, false)
			node1 := tr.define("/abc/efg")
			node2 := tr.define("/abc/xyz/")

			EqualPtr(t, node1, tr.match("/abc/efg").node)
			assert.False(tr.match("/abc/efg").tsr)
			assert.Nil(tr.match("/abc/efg/").node)
			assert.False(tr.match("/abc/efg/").tsr)

			EqualPtr(t, node2, tr.match("/abc/xyz/").node)
			assert.False(tr.match("/abc/xyz/").tsr)
			assert.Nil(tr.match("/abc/xyz").node)
			assert.False(tr.match("/abc/xyz").tsr)

			// trailingSlashRedirect = true
			tr = newTrie(true, true)
			node1 = tr.define("/abc/efg")
			node2 = tr.define("/abc/xyz/")

			EqualPtr(t, node1, tr.match("/abc/efg").node)
			assert.False(tr.match("/abc/efg").tsr)
			assert.Nil(tr.match("/abc/efg/").node)
			assert.True(tr.match("/abc/efg/").tsr)

			EqualPtr(t, node2, tr.match("/abc/xyz/").node)
			assert.False(tr.match("/abc/xyz/").tsr)
			assert.Nil(tr.match("/abc/xyz").node)
			assert.True(tr.match("/abc/xyz").tsr)
		})
	})
}
