package gear

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGearRouter(t *testing.T) {
	app := New()

	t.Run("router.Use, router.Handle", func(t *testing.T) {
		// apiCount := 0
		r := NewRouter("/api", false)
		r.Use(func(ctx *Context) error {
			require.True(t, strings.HasPrefix(ctx.Path, "/api"))
			ctx.SetAny("middleware", true)
			return nil
		})
		r.Handle("GET", "/users", func(ctx *Context) error {
			return ctx.HTML(200, "ok")
		})

		ctx := NewCtx(app, "GET", "/", nil)
		err := r.Serve(ctx)
		require.Nil(t, err)
		any, _ := ctx.Any("middleware")
		require.Nil(t, any)

		ctx = NewCtx(app, "GET", "/api", nil)
		err = r.Serve(ctx)
		require.NotNil(t, err)
		require.Equal(t, 501, ctx.Res.Status)
		any, _ = ctx.Any("middleware")
		require.Nil(t, any)

		ctx = NewCtx(app, "GET", "/api/users", nil)
		err = r.Serve(ctx)
		require.Nil(t, err)
		require.Equal(t, 200, ctx.Res.Status)
		any, _ = ctx.Any("middleware")
		require.True(t, any.(bool))
	})
}
