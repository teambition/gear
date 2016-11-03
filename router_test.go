package gear

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGearRouter(t *testing.T) {
	newApp := func(router *Router) *ServerListener {
		app := New()
		app.UseHandler(router)
		return app.Start()
	}

	req := NewRequst()

	t.Run("router.Use, router.Handle", func(t *testing.T) {
		assert := assert.New(t)

		called := 0
		r := NewRouter("/api", false)
		r.Use(func(ctx *Context) error {
			assert.True(strings.HasPrefix(ctx.Path, "/api"))
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
		assert.Nil(err)
		assert.Equal(0, called)
		assert.Equal(500, res.StatusCode)
		assert.Equal("Internal Server Error", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/api")
		assert.Nil(err)
		assert.Equal(0, called)
		assert.Equal(501, res.StatusCode)
		assert.Equal("\"/api\" not implemented", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/api/users")
		assert.Nil(err)
		assert.Equal(1, called)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with http.Method", func(t *testing.T) {
		assert := assert.New(t)

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
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("GET", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Head(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Post(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("POST", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Put(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("PUT", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Patch(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("PATCH", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Delete(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("DELETE", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Options(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OPTIONS", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with 501", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter("/", false)
		r.Get("/abc", func(ctx *Context) error {
			ctx.End(204)
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(501, res.StatusCode)
		assert.Equal("\"/\" not implemented", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with 405", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter("/", false)
		r.Get("/abc", func(ctx *Context) error {
			ctx.End(204)
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Put(host + "/abc")
		assert.Nil(err)
		assert.Equal(405, res.StatusCode)
		assert.Equal("\"PUT\" not allowed in \"/abc\"", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with auto options respond", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter("/", false)
		r.Get("/abc", func(ctx *Context) error {
			ctx.End(204)
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Options(host + "/abc")
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		assert.Equal("", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with named pattern", func(t *testing.T) {
		assert := assert.New(t)

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
		assert.Nil(err)
		assert.Equal(1, count)
		assert.Equal(200, res.StatusCode)
		assert.Equal("user123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with double colon pattern", func(t *testing.T) {
		assert := assert.New(t)

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
		assert.Nil(err)
		assert.Equal(1, count)
		assert.Equal(200, res.StatusCode)
		assert.Equal("123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with wildcard pattern", func(t *testing.T) {
		assert := assert.New(t)

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
		assert.Nil(err)
		assert.Equal(1, count)
		assert.Equal(200, res.StatusCode)
		assert.Equal("user/123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with regexp pattern", func(t *testing.T) {
		assert := assert.New(t)

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
		assert.Nil(err)
		assert.Equal(0, count)
		assert.Equal(501, res.StatusCode)
		res.Body.Close()

		res, err = req.Get(host + "/api/user/123")
		assert.Nil(err)
		assert.Equal(1, count)
		assert.Equal(200, res.StatusCode)
		assert.Equal("user123", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with Otherwise", func(t *testing.T) {
		assert := assert.New(t)

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
		assert.Nil(err)
		assert.Equal(1, count)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/api/user/abc")
		assert.Nil(err)
		assert.Equal(2, count)
		assert.Equal(404, res.StatusCode)
		assert.Equal("GET /api/user/abc", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Put(host + "/api")
		assert.Nil(err)
		assert.Equal(3, count)
		assert.Equal(404, res.StatusCode)
		assert.Equal("PUT /api", PickRes(res.Text()).(string))
		res.Body.Close()
	})
}
