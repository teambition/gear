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
			assert.Equal("/users", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/api/users", GetRouterPatternFromCtx(ctx))
			return ctx.HTML(200, "OK")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(0, called)
		assert.Equal(421, res.StatusCode)
		assert.Equal(string(misdirectedResponseBody), PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/api")
		assert.Nil(err)
		assert.Equal(0, called)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/api/users")
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
			ctx.End(200, []byte("OK"))
			return nil
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host)
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

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("GET", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("HEAD", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("POST", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("POST", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("PUT", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("PUT", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("PATCH", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("PATCH", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("DELETE", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("DELETE", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("OPTIONS", host)
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

		res, err := RequestBy("OPTIONS", host)
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

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(3, called)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with 421", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter()
		r.Get("/abc", func(ctx *Context) error {
			assert.Equal("/abc", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/abc", GetRouterPatternFromCtx(ctx))
			return ctx.End(204)
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
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

		res, err := RequestBy("PUT", host+"/abc")
		assert.Nil(err)
		assert.Equal(405, res.StatusCode)
		assert.Equal("GET", res.Header.Get(HeaderAllow))
		assert.Equal("nosniff", res.Header.Get(HeaderXContentTypeOptions))
		assert.Equal("application/json; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal(`{"error":"MethodNotAllowed","message":"\"PUT\" is not allowed in \"/abc\""}`, PickRes(res.Text()).(string))
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
			assert.Equal("/api/:type/:ID", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/api/:type/:ID", GetRouterPatternFromCtx(ctx))
			return ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host+"/api/user/123")
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
			assert.Equal("/api/::/:ID", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/api/::/:ID", GetRouterPatternFromCtx(ctx))
			return ctx.HTML(200, ctx.Param("ID"))
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host+"/api/:/123")
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
			assert.Equal("/api/:type*", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/api/:type*", GetRouterPatternFromCtx(ctx))
			return ctx.HTML(200, ctx.Param("type"))
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host+"/api/user/123")
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
			assert.Equal(`/api/:type/:ID(^\d+$)`, GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal(`/api/:type/:ID(^\d+$)`, GetRouterPatternFromCtx(ctx))
			return nil
		})
		r.Get(`/api/:type/:ID(^\d+$)`, func(ctx *Context) error {
			assert.Equal(`/api/:type/:ID(^\d+$)`, GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal(`/api/:type/:ID(^\d+$)`, GetRouterPatternFromCtx(ctx))
			return ctx.HTML(200, ctx.Param("type")+ctx.Param("ID"))
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host+"/api/user/abc")
		assert.Nil(err)
		assert.Equal(0, count)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/api/user/123")
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
			switch ctx.Method {
			case "GET":
				assert.Nil(GetRouterNodeFromCtx(ctx))
				assert.Equal("", GetRouterPatternFromCtx(ctx))
			case "PUT":
				assert.Equal("/api", GetRouterNodeFromCtx(ctx).GetPattern())
				assert.Equal("/api", GetRouterPatternFromCtx(ctx))
			}
			return ctx.HTML(404, ctx.Method+" "+ctx.Path)
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host+"/api")
		assert.Nil(err)
		assert.Equal(1, count)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/api/user/abc")
		assert.Nil(err)
		assert.Equal(2, count)
		assert.Equal(404, res.StatusCode)
		assert.Equal("GET /api/user/abc", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("PUT", host+"/api")
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

		res, err := RequestBy("GET", host)
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

		res, err := RequestBy("GET", host+"/api/user/123")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("user123", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/API/User/Abc")
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

		res, err := RequestBy("GET", host+"/api/user/123")
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/Api/User/Abc")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("UserAbc", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with FixedPathRedirect = true (defalut)", func(t *testing.T) {
		assert := assert.New(t)
		app := New()

		r := NewRouter()

		r.Get("/", func(ctx *Context) error {
			return ctx.HTML(200, "/")
		})

		r.Get("/abc/efg", func(ctx *Context) error {
			return ctx.HTML(200, "/abc/efg")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/efg")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc//efg")
		assert.NoError(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		ctx := CtxTest(app, "GET", "/abc//efg", nil)
		r.Serve(ctx)
		rt := CtxResult(ctx)
		assert.Equal(301, rt.StatusCode)
		assert.Equal("/abc/efg", rt.Header.Get("Location"))
	})

	t.Run("router with FixedPathRedirect = false", func(t *testing.T) {
		assert := assert.New(t)
		app := New()

		r := NewRouter(RouterOptions{})

		r.Get("/", func(ctx *Context) error {
			return ctx.HTML(200, "/")
		})

		r.Get("/abc/efg", func(ctx *Context) error {
			return ctx.HTML(200, "/abc/efg")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/efg")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc//efg")
		assert.NoError(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		ctx := CtxTest(app, "GET", "/abc//efg", nil)
		err = r.Serve(ctx)
		assert.Nil(err)
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

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/efg")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/efg/")
		assert.NoError(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		ctx := CtxTest(app, "GET", "/abc/efg/", nil)
		r.Serve(ctx)
		rt := CtxResult(ctx)
		assert.Equal(301, rt.StatusCode)
		assert.Equal("/abc/efg", rt.Header.Get("Location"))

		res, err = RequestBy("PUT", host+"/abc/xyz/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/xyz/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("PUT", host+"/abc/xyz")
		assert.NoError(err)
		assert.Equal(200, res.StatusCode)
		res.Body.Close()

		ctx = CtxTest(app, "PUT", "/abc/xyz", nil)
		r.Serve(ctx)
		rt = CtxResult(ctx)
		assert.Equal(307, rt.StatusCode)
		assert.Equal("/abc/xyz/", rt.Header.Get("Location"))
	})

	t.Run("router with root", func(t *testing.T) {
		assert := assert.New(t)

		r := NewRouter(RouterOptions{Root: "/api"})
		r.Get("/", func(ctx *Context) error {
			return ctx.End(200, []byte("hello"))
		})

		app := New()
		app.UseHandler(r)

		srv := app.Start()
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/api")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("hello", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("router with root and FixedPathRedirect", func(t *testing.T) {
		assert := assert.New(t)
		app := New()

		r := NewRouter(RouterOptions{Root: "/api", FixedPathRedirect: true, TrailingSlashRedirect: true})

		r.Get("/abc/efg", func(ctx *Context) error {
			return ctx.HTML(200, "/api/abc/efg")
		})

		srv := newApp(r)
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host+"/api//abc///efg//")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/api/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		ctx := CtxTest(app, "GET", "/api//abc///efg//", nil)
		r.Serve(ctx)
		rt := CtxResult(ctx)
		assert.Equal(301, rt.StatusCode)
		assert.Equal("/api/abc/efg", rt.Header.Get("Location"))
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

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/efg")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/efg", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/efg/")
		assert.NoError(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		ctx := CtxTest(app, "GET", "/abc/efg/", nil)
		err = r.Serve(ctx)
		assert.NoError(err)
		assert.Nil(r.Serve(ctx))

		res, err = RequestBy("PUT", host+"/abc/xyz/")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("/abc/xyz/", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("PUT", host+"/abc/xyz")
		assert.NoError(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		ctx = CtxTest(app, "PUT", "/abc/xyz", nil)
		assert.Nil(r.Serve(ctx))
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

		res, err := RequestBy("GET", host)
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

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"InternalServerError","message":"some error"}`, PickRes(res.Text()).(string))
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

		res, err := RequestBy("GET", host)
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

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"InternalServerError","message":"some error"}`, PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("multi-routers", func(t *testing.T) {
		assert := assert.New(t)

		r0 := NewRouter()
		r0.Get("/", func(ctx *Context) error {
			assert.Equal("/", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/", GetRouterPatternFromCtx(ctx))
			return ctx.End(200, []byte("ok"))
		})
		r0.Get("/xyz", func(ctx *Context) error {
			assert.Equal("/xyz", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/xyz", GetRouterPatternFromCtx(ctx))
			return ctx.End(200, []byte("xyz"))
		})

		r1 := NewRouter(RouterOptions{Root: "/abc"})
		r1.Get("/:name", func(ctx *Context) error {
			assert.Equal("/:name", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/abc/:name", GetRouterPatternFromCtx(ctx))
			return ctx.End(200, []byte(ctx.Param("name")))
		})

		r2 := NewRouter(RouterOptions{Root: "/abcd"})
		r2.Get("/:name", func(ctx *Context) error {
			assert.Equal("/:name", GetRouterNodeFromCtx(ctx).GetPattern())
			assert.Equal("/abcd/:name", GetRouterPatternFromCtx(ctx))
			return ctx.End(200, []byte(ctx.Param("name")))
		})

		app := New()
		app.UseHandler(r0)
		app.UseHandler(r1)
		app.UseHandler(r2)

		srv := app.Start()
		defer srv.Close()
		host := "http://" + srv.Addr().String()

		res, err := RequestBy("GET", host)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("ok", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/xyz")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("xyz", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/xyzz")
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc")
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/")
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abc/123")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("123", PickRes(res.Text()).(string))
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abcd")
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abcd/")
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		res.Body.Close()

		res, err = RequestBy("GET", host+"/abcd/123")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("123", PickRes(res.Text()).(string))
		res.Body.Close()
	})
}
