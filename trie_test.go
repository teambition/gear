package gear

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type te struct {
	x int
}

func EqualPtr(t *testing.T, a, b interface{}) {
	require.Equal(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func NotEqualPtr(t *testing.T, a, b interface{}) {
	require.NotEqual(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func TestGearTrie(t *testing.T) {
	t.Run("trie.define", func(t *testing.T) {
		t.Run("root pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			tr2 := newTrie(true)
			node := tr1.define("/")
			require.Equal(t, node.name, "")

			EqualPtr(t, node, tr1.define("/"))
			EqualPtr(t, node, tr1.define(""))
			NotEqualPtr(t, node, tr2.define("/"))
			NotEqualPtr(t, node, tr2.define(""))

			EqualPtr(t, node.parentNode, tr1.root)
		})

		t.Run("simple pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/a/b")
			require.Equal(t, node.name, "")

			EqualPtr(t, node, tr1.define("/a/b"))
			EqualPtr(t, node, tr1.define("a/b/"))
			EqualPtr(t, node, tr1.define("/a/b/"))
			require.Equal(t, node.pattern, "/a/b")

			parent := tr1.define("/a")
			EqualPtr(t, node.parentNode, parent)
			NotEqualPtr(t, parent.varyChild, node)
			EqualPtr(t, parent.literalChildren["b"], node)
			child := tr1.define("/a/b/c")
			EqualPtr(t, child.parentNode, node)
			EqualPtr(t, node.literalChildren["c"], child)

			require.Panics(t, func() {
				tr1.define("/a//b")
			})
		})

		t.Run("double colon pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/a/::b")
			require.Equal(t, node.name, "")
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
			tr1 := newTrie(true)

			require.Panics(t, func() {
				tr1.define("/a/:")
			})
			require.Panics(t, func() {
				tr1.define("/a/:/")
			})
			require.Panics(t, func() {
				tr1.define("/a/:abc$/")
			})
			node := tr1.define("/a/:b")
			require.Equal(t, node.name, "b")
			require.False(t, node.wildcard)
			require.Nil(t, node.varyChild)
			require.Equal(t, node.pattern, "/a/:b")
			require.Panics(t, func() {
				tr1.define("/a/:x")
			})

			parent := tr1.define("/a")
			require.Equal(t, parent.name, "")
			EqualPtr(t, parent.varyChild, node)
			EqualPtr(t, node.parentNode, parent)
			child := tr1.define("/a/:b/c")
			EqualPtr(t, child.parentNode, node)
			require.Panics(t, func() {
				tr1.define("/a/:x/c")
			})
		})

		t.Run("wildcard pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			require.Panics(t, func() {
				tr1.define("/a/*")
			})
			require.Panics(t, func() {
				tr1.define("/a/:*")
			})
			require.Panics(t, func() {
				tr1.define("/a/:#*")
			})
			require.Panics(t, func() {
				tr1.define("/a/:abc(*")
			})

			node := tr1.define("/a/:b*")
			require.Equal(t, node.name, "b")
			require.True(t, node.wildcard)
			require.Nil(t, node.varyChild)
			require.Equal(t, node.pattern, "/a/:b*")
			require.Panics(t, func() {
				tr1.define("/a/:x*")
			})

			parent := tr1.define("/a")
			require.Equal(t, parent.name, "")
			require.False(t, parent.wildcard)
			EqualPtr(t, parent.varyChild, node)
			EqualPtr(t, node.parentNode, parent)
			require.Panics(t, func() {
				tr1.define("/a/:b*/c")
			})
		})

		t.Run("regexp pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			require.Panics(t, func() {
				tr1.define("/a/(")
			})
			require.Panics(t, func() {
				tr1.define("/a/)")
			})
			require.Panics(t, func() {
				tr1.define("/a/:(")
			})
			require.Panics(t, func() {
				tr1.define("/a/:)")
			})
			require.Panics(t, func() {
				tr1.define("/a/:()")
			})
			require.Panics(t, func() {
				tr1.define("/a/:bc)")
			})
			require.Panics(t, func() {
				tr1.define("/a/:(bc)")
			})
			require.Panics(t, func() {
				tr1.define("/a/:#(bc)")
			})
			require.Panics(t, func() {
				tr1.define("/a/:b(c)*")
			})

			node := tr1.define("/a/:b(x|y|z)")
			require.Equal(t, node.name, "b")
			require.Equal(t, node.pattern, "/a/:b(x|y|z)")
			require.False(t, node.wildcard)
			require.Nil(t, node.varyChild)
			require.Equal(t, node, tr1.define("/a/:b(x|y|z)"))
			require.Panics(t, func() {
				tr1.define("/a/:x(x|y|z)")
			})

			parent := tr1.define("/a")
			require.Equal(t, parent.name, "")
			require.False(t, parent.wildcard)
			EqualPtr(t, parent.varyChild, node)
			EqualPtr(t, node.parentNode, parent)

			child := tr1.define("/a/:b(x|y|z)/c")
			EqualPtr(t, child.parentNode, node)
			require.Panics(t, func() {
				tr1.define("/a/:x(x|y|z)/c")
			})
		})
	})

	t.Run("trie.match", func(t *testing.T) {
		t.Run("root pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/")
			res := tr1.match("/")
			fmt.Println(res)
			require.Nil(t, res.params)
			EqualPtr(t, node, res.node)

			res2 := tr1.match("")
			EqualPtr(t, node, res2.node)
			NotEqualPtr(t, res, res2)

			require.Nil(t, tr1.match("/a").node)
		})

		t.Run("simple pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/a/b")
			res := tr1.match("/a/b")
			require.Nil(t, res.params)
			EqualPtr(t, node, res.node)

			require.Nil(t, tr1.match("/a").node)
			require.Nil(t, tr1.match("/a/b/c").node)
			require.Nil(t, tr1.match("/a/x/c").node)
		})

		t.Run("double colon pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/a/::b")
			res := tr1.match("/a/:b")
			require.Nil(t, res.params)
			EqualPtr(t, node, res.node)
			require.Nil(t, tr1.match("/a").node)
			require.Nil(t, tr1.match("/a/::b").node)

			node = tr1.define("/a/::b/c")
			res = tr1.match("/a/:b/c")
			require.Nil(t, res.params)
			EqualPtr(t, node, res.node)
			require.Nil(t, tr1.match("/a/::b/c").node)

			node = tr1.define("/a/::")
			res = tr1.match("/a/:")
			require.Nil(t, res.params)
			EqualPtr(t, node, res.node)
			require.Nil(t, tr1.match("/a/::").node)
		})

		t.Run("named pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/a/:b")
			res := tr1.match("/a/xyz汉")
			require.Equal(t, "xyz汉", res.params["b"])
			require.Equal(t, "", res.params["x"])
			EqualPtr(t, node, res.node)
			require.Nil(t, tr1.match("/a").node)
			require.Nil(t, tr1.match("/a/xyz汉/123").node)

			node2 := tr1.define("/:a/:b")
			res2 := tr1.match("/a/xyz汉")
			EqualPtr(t, node, res2.node)

			res2 = tr1.match("/ab/xyz汉")
			require.Equal(t, "xyz汉", res2.params["b"])
			require.Equal(t, "ab", res2.params["a"])
			EqualPtr(t, node2, res2.node)
			require.Nil(t, tr1.match("/ab").node)
			require.Nil(t, tr1.match("/ab/xyz汉/123").node)
		})

		t.Run("wildcard pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/a/:b*")
			res := tr1.match("/a/xyz汉")
			require.Equal(t, "xyz汉", res.params["b"])
			EqualPtr(t, node, res.node)
			require.Nil(t, tr1.match("/a").node)

			res = tr1.match("/a/xyz汉/123")
			require.Equal(t, "xyz汉/123", res.params["b"])
			EqualPtr(t, node, res.node)

			node = tr1.define("/:a*")
			require.Nil(t, tr1.match("/a").node) // TODO
			res = tr1.match("/123")
			require.Equal(t, "123", res.params["a"])
			EqualPtr(t, node, res.node)
			res = tr1.match("/123/xyz汉")
			require.Equal(t, "123/xyz汉", res.params["a"])
			EqualPtr(t, node, res.node)
		})

		t.Run("regexp pattern", func(t *testing.T) {
			tr1 := newTrie(true)
			node := tr1.define("/a/:b(^(x|y|z)$)")
			res := tr1.match("/a/x")
			require.Equal(t, "x", res.params["b"])
			EqualPtr(t, node, res.node)
			res = tr1.match("/a/y")
			require.Equal(t, "y", res.params["b"])
			EqualPtr(t, node, res.node)
			res = tr1.match("/a/z")
			require.Equal(t, "z", res.params["b"])
			EqualPtr(t, node, res.node)

			require.Nil(t, tr1.match("/a").node)
			require.Nil(t, tr1.match("/a/xy").node)
			require.Nil(t, tr1.match("/a/x/y").node)

			child := tr1.define("/a/:b(^(x|y|z)$)/c")
			res = tr1.match("/a/x/c")
			require.Equal(t, "x", res.params["b"])
			EqualPtr(t, child, res.node)
			res = tr1.match("/a/y/c")
			require.Equal(t, "y", res.params["b"])
			EqualPtr(t, child, res.node)
			res = tr1.match("/a/z/c")
			require.Equal(t, "z", res.params["b"])
			EqualPtr(t, child, res.node)
		})
	})
}
