package gear

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// ----- Test Context.Any -----
type ctxAnyType struct{}
type ctxAnyResult struct {
	Host string
	Path string
}

var ctxAny = &ctxAnyType{}

func (t *ctxAnyType) New(ctx *Context) (interface{}, error) {
	if ctx.Method != "GET" {
		return nil, errors.New(ctx.Method)
	}
	return &ctxAnyResult{Host: ctx.Host, Path: ctx.Path}, nil
}

func TestGearContextAny(t *testing.T) {
	app := New()

	t.Run("type Any", func(t *testing.T) {
		t.Run("should get the same value with the same ctx", func(t *testing.T) {
			ctx := NewCtx(app, "GET", "http://example.com/foo", nil)
			val, err := ctx.Any(ctxAny)
			require.Nil(t, err)
			res := val.(*ctxAnyResult)
			require.Equal(t, ctx.Host, res.Host)
			require.Equal(t, ctx.Path, res.Path)

			val2, _ := ctx.Any(ctxAny)
			EqualPtr(t, val, val2)
		})

		t.Run("should get different value with different ctx", func(t *testing.T) {
			ctx := NewCtx(app, "GET", "http://example.com/foo", nil)
			val, err := ctx.Any(ctxAny)
			require.Nil(t, err)

			ctx2 := NewCtx(app, "GET", "http://example.com/foo", nil)
			val2, err2 := ctx2.Any(ctxAny)
			require.Nil(t, err2)
			NotEqualPtr(t, val, val2)
		})

		t.Run("should get error", func(t *testing.T) {
			ctx := NewCtx(app, "POST", "http://example.com/foo", nil)
			val, err := ctx.Any(ctxAny)
			require.Nil(t, val)
			require.NotNil(t, err)
			require.Equal(t, "POST", err.Error())
		})
	})

	t.Run("SetAny with interface{}", func(t *testing.T) {
		ctx := NewCtx(app, "POST", "http://example.com/foo", nil)
		val, err := ctx.Any(struct{}{})
		require.Nil(t, val)
		require.Equal(t, "non-existent key", err.Error())

		ctx.SetAny(struct{}{}, true)
		val, err = ctx.Any(struct{}{})
		require.Nil(t, err)
		require.True(t, val.(bool))
	})
}
