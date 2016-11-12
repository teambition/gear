package gear

import (
	"errors"
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
		r := NewRouter(RouterOptions{Root: "/api"})
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
		assert.Equal(444, res.StatusCode)
		assert.Equal("", PickRes(res.Text()).(string))
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

	t.Run("router.Handle with one more middleware", func(t *testing.T) {
		assert := assert.New(t)

		called := 0
		r := NewRouter()

		assert.Panics(func() {
			r.Handle("GET", "/")
		})

		assert.Panics(func() {
			r.Handle("", "/", func(_ *Context) error {
				return nil
			})
		})

		r.Handle("GET", "/", func(ctx *Context) error {
			called++
			assert.Equal(1, called)
			return nil
		}, func(ctx *Context) error {
			called++
			assert.Equal(2, called)
			return nil
		}, func(ctx *Context) error {
			called++
			assert.Equal(3, called)
			ctx.String(200, "OK")
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(3, called)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with http.Method", func(t *testing.T) {
		assert := assert.New(t)

		middleware := func(ctx *Context) error {
			return ctx.HTML(200, ctx.Method)
		}
		r := NewRouter()
		r.Get("/", middleware)
		r.Head("/", middleware)
		r.Post("/", middleware)
		r.Put("/", middleware)
		r.Patch("/", middleware)
		r.Delete("/", middleware)
		r.Options("/", middleware)

		assert.Panics(func() {
			r.Get("", middleware)
		})

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

	t.Run("automatic handle `OPTIONS` method", func(t *testing.T) {
		assert := assert.New(t)

		middleware := func(ctx *Context) error {
			return ctx.HTML(200, ctx.Method)
		}
		r := NewRouter()
		r.Get("/", middleware)
		r.Head("/", middleware)
		r.Post("/", middleware)
		r.Put("/", middleware)

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Options(host)
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		assert.Equal("GET, HEAD, POST, PUT", res.Header.Get(HeaderAllow))
		res.Body.Close()
	})

	t.Run("router.Get with one more middleware", func(t *testing.T) {
		assert := assert.New(t)

		called := 0
		r := NewRouter()

		assert.Panics(func() {
			r.Get("/")
		})

		r.Get("/", func(ctx *Context) error {
			called++
			assert.Equal(1, called)
			return nil
		}, func(ctx *Context) error {
			called++
			assert.Equal(2, called)
			return nil
		}, func(ctx *Context) error {
			called++
			assert.Equal(3, called)
			return ctx.HTML(200, "OK")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(3, called)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with 501", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter()
		r.Get("/abc", func(ctx *Context) error {
			return ctx.End(204)
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

		r := NewRouter()
		r.Get("/abc", func(ctx *Context) error {
			return ctx.End(204)
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

	t.Run("router with named pattern", func(t *testing.T) {
		assert := assert.New(t)

		count := 0
		r := NewRouter()
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/:type/:ID", func(ctx *Context) error {
			return ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
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
		r := NewRouter()
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/::/:ID", func(ctx *Context) error {
			return ctx.HTML(200, ctx.Param("ID"))
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
		r := NewRouter()
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/:type*", func(ctx *Context) error {
			return ctx.HTML(200, ctx.Param("type"))
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
		r := NewRouter()
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api/:type/:ID(^\\d+$)", func(ctx *Context) error {
			return ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
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
		r := NewRouter()
		r.Use(func(ctx *Context) error {
			count++
			return nil
		})
		r.Get("/api", func(ctx *Context) error {
			return ctx.HTML(200, "OK")
		})
		r.Otherwise(func(ctx *Context) error {
			return ctx.HTML(404, ctx.Method+" "+ctx.Path)
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

	t.Run("router.Otherwise with one more middleware", func(t *testing.T) {
		assert := assert.New(t)

		called := 0
		r := NewRouter()

		assert.Panics(func() {
			r.Otherwise()
		})

		r.Otherwise(func(ctx *Context) error {
			called++
			assert.Equal(1, called)
			return nil
		}, func(ctx *Context) error {
			called++
			assert.Equal(2, called)
			return nil
		}, func(ctx *Context) error {
			called++
			assert.Equal(3, called)
			return ctx.HTML(200, "OK")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(3, called)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with IgnoreCase = true (defalut)", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter()

		r.Get("/Api/:type/:ID", func(ctx *Context) error {
			return ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host + "/api/user/123")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("user123", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/API/User/Abc")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("UserAbc", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with IgnoreCase = false", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter(RouterOptions{})

		r.Get("/Api/:type/:ID", func(ctx *Context) error {
			return ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host + "/api/user/123")
		assert.Nil(err)
		assert.Equal(501, res.StatusCode)
		res.Body.Close()

		res, err = req.Get(host + "/Api/User/Abc")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("UserAbc", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with TrailingSlashRedirect = true (defalut)", func(t *testing.T) {
		assert := assert.New(t)
		app := New()

		r := NewRouter()

		r.Get("/", func(ctx *Context) error {
			return ctx.HTML(200, "/")
		})

		r.Get("/abc/efg", func(ctx *Context) error {
			return ctx.HTML(200, "/abc/efg")
		})

		r.Put("/abc/xyz/", func(ctx *Context) error {
			return ctx.HTML(200, "/abc/xyz/")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/abc/efg")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/abc/efg/")
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		ctx := CtxTest(app, "GET", "/abc/efg/", nil)
		r.Serve(ctx)
		rt := CtxResult(ctx)
		assert.Equal(301, rt.StatusCode)
		assert.Equal("/abc/efg", rt.Header.Get("Location"))

		res, err = req.Put(host + "/abc/xyz/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/xyz/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Put(host + "/abc/xyz")
		assert.Equal(307, res.StatusCode)
		assert.Equal("/abc/xyz/", res.Header.Get("Location"))
		res.Body.Close()

		ctx = CtxTest(app, "PUT", "/abc/xyz", nil)
		r.Serve(ctx)
		rt = CtxResult(ctx)
		assert.Equal(307, rt.StatusCode)
		assert.Equal("/abc/xyz/", rt.Header.Get("Location"))
	})

	t.Run("router with TrailingSlashRedirect = false", func(t *testing.T) {
		assert := assert.New(t)
		app := New()

		r := NewRouter(RouterOptions{})

		r.Get("/", func(ctx *Context) error {
			return ctx.HTML(200, "/")
		})

		r.Get("/abc/efg", func(ctx *Context) error {
			return ctx.HTML(200, "/abc/efg")
		})

		r.Put("/abc/xyz/", func(ctx *Context) error {
			return ctx.HTML(200, "/abc/xyz/")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/abc/efg")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Get(host + "/abc/efg/")
		assert.Equal(501, res.StatusCode)
		res.Body.Close()

		ctx := CtxTest(app, "GET", "/abc/efg/", nil)
		err = r.Serve(ctx)
		assert.Equal(501, err.(HTTPError).Status())

		res, err = req.Put(host + "/abc/xyz/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/xyz/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = req.Put(host + "/abc/xyz")
		assert.Equal(501, res.StatusCode)
		res.Body.Close()

		ctx = CtxTest(app, "PUT", "/abc/xyz", nil)
		err = r.Serve(ctx)
		assert.Equal(501, err.(HTTPError).Status())
	})

	t.Run("when router middleware ended early", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter()
		r.Use(func(ctx *Context) error {
			return ctx.HTML(200, "OK")
		})
		r.Get("/", func(ctx *Context) error {
			panic("this middleware unreachable")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("when router middleware error", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter()
		r.Use(func(ctx *Context) error {
			return errors.New("some error")
		})
		r.Get("/", func(ctx *Context) error {
			panic("this middleware unreachable")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("some error", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("when handler middleware ended early", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter()
		r.Get("/", func(ctx *Context) error {
			return ctx.HTML(200, "OK")
		}, func(ctx *Context) error {
			panic("this middleware unreachable")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("when handler middleware error", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter()
		r.Get("/", func(ctx *Context) error {
			return errors.New("some error")
		}, func(ctx *Context) error {
			panic("this middleware unreachable")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := req.Get(host)
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("some error", PickRes(res.Text()).(string))
		res.Body.Close()
	})
}
