package gear

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGearRouter(t *testing.T) {
	newApp := func(router *Router) *ServerListener {
		app := New()
		app.UseHandler(router)
		return app.Start()
	}

	req := NewRequst()

	t.Run("router.Use, router.Handle", func(t *testing.T) {
		called := 0
		r := NewRouter("/api", false)
		r.Use(func(ctx *Context) error {
			require.True(t, strings.HasPrefix(ctx.Path, "/api"))
			called++
			return nil
		})
		r.Handle("GET", "/users", func(ctx *Context) error {
			return ctx.HTML(200, "OK")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		require.Nil(t, err)
		require.Equal(t, 0, called)
		require.Equal(t, 500, res.StatusCode)
		require.Equal(t, "Internal Server Error", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/api")
		require.Nil(t, err)
		require.Equal(t, 0, called)
		require.Equal(t, 501, res.StatusCode)
		require.Equal(t, "\"/api\" not implemented\n", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/api/users")
		require.Nil(t, err)
		require.Equal(t, 1, called)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with http.Method", func(t *testing.T) {
		middleware := func(ctx *Context) error {
			return ctx.HTML(200, ctx.Method)
		}
		r := NewRouter("/", false)
		r.Get("/", middleware)
		r.Head("/", middleware)
		r.Post("/", middleware)
		r.Put("/", middleware)
		r.Patch("/", middleware)
		r.Delete("/", middleware)
		r.Options("/", middleware)

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "GET", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Head(host)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Post(host)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "POST", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Put(host)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "PUT", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Patch(host)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "PATCH", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Delete(host)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "DELETE", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Options(host)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "OPTIONS", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with 501", func(t *testing.T) {
		r := NewRouter("/", false)
		r.Get("/abc", func(ctx *Context) error {
			ctx.End(204)
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		require.Nil(t, err)
		require.Equal(t, 501, res.StatusCode)
		require.Equal(t, "\"/\" not implemented\n", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with 405", func(t *testing.T) {
		r := NewRouter("/", false)
		r.Get("/abc", func(ctx *Context) error {
			ctx.End(204)
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Put(host + "/abc")
		require.Nil(t, err)
		require.Equal(t, 405, res.StatusCode)
		require.Equal(t, "\"PUT\" not allowed in \"/abc\"\n", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with auto options respond", func(t *testing.T) {
		r := NewRouter("/", false)
		r.Get("/abc", func(ctx *Context) error {
			ctx.End(204)
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Options(host + "/abc")
		require.Nil(t, err)
		require.Equal(t, 204, res.StatusCode)
		require.Equal(t, "", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with named pattern", func(t *testing.T) {
		count := 0
		r := NewRouter("/", false)
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/:type/:ID", func(ctx *Context) error {
			ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host + "/api/user/123")
		require.Nil(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "user123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with double colon pattern", func(t *testing.T) {
		count := 0
		r := NewRouter("/", false)
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/::/:ID", func(ctx *Context) error {
			ctx.HTML(200, ctx.Param("ID"))
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host + "/api/:/123")
		require.Nil(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with wildcard pattern", func(t *testing.T) {
		count := 0
		r := NewRouter("/", false)
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/:type*", func(ctx *Context) error {
			ctx.HTML(200, ctx.Param("type"))
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host + "/api/user/123")
		require.Nil(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "user/123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with regexp pattern", func(t *testing.T) {
		count := 0
		r := NewRouter("/", false)
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/:type/:ID(^\\d+$)", func(ctx *Context) error {
			ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host + "/api/user/abc")
		require.Nil(t, err)
		require.Equal(t, 0, count)
		require.Equal(t, 501, res.StatusCode)
		res.Body.Close()

		res, err = req.Get(host + "/api/user/123")
		require.Nil(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "user123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with Otherwise", func(t *testing.T) {
		count := 0
		r := NewRouter("/", false)
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api", func(ctx *Context) error {
			ctx.HTML(200, "OK")
			return nil
		})
		r.Otherwise(func(ctx *Context) error {
			ctx.HTML(404, ctx.Method+" "+ctx.Path)
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host + "/api")
		require.Nil(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "OK", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/api/user/abc")
		require.Nil(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 404, res.StatusCode)
		require.Equal(t, "GET /api/user/abc", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Put(host + "/api")
		require.Nil(t, err)
		require.Equal(t, 3, count)
		require.Equal(t, 404, res.StatusCode)
		require.Equal(t, "PUT /api", PickRes(res.Text()).(string))
		res.Body.Close()
	})
}
