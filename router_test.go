package gear

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGearRouter(t *testing.T) {
	app := New()
	reqCtx := func(method, url string, body io.Reader) *Context {
		ctx := NewContext(app)
		req := httptest.NewRequest(method, url, body)
		res := httptest.NewRecorder()
		ctx.Reset(res, req)
		return ctx
	}

	t.Run("router.Use, router.Handle", func(t *testing.T) {
		// apiCount := 0
		r := NewRouter("/api", false)
		r.Use(func(ctx *Context) error {
			require.True(t, strings.HasPrefix(ctx.Path, "/api"))
			ctx.SetValue("middleware", true)
			return nil
		})
		r.Handle("GET", "/users", func(ctx *Context) error {
			return ctx.HTML(200, "ok")
		})

		ctx := reqCtx("GET", "/", nil)
		err := r.Serve(ctx)
		require.Nil(t, err)
		require.Nil(t, ctx.Value("middleware"))

		ctx = reqCtx("GET", "/api", nil)
		err = r.Serve(ctx)
		require.NotNil(t, err)
		require.Equal(t, 501, ctx.Res.Status)
		require.Nil(t, ctx.Value("middleware"))

		ctx = reqCtx("GET", "/api/users", nil)
		err = r.Serve(ctx)
		require.Nil(t, err)
		require.Equal(t, 200, ctx.Res.Status)
		require.True(t, ctx.Value("middleware").(bool))
	})
}
