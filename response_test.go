package gear

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
)

func TestGearResponse(t *testing.T) {
	app := New()

	t.Run("Header", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		res := ctx.Res
		header := res.Header()

		res.Set("Set-Cookie", "foo=bar; Path=/; HttpOnly")
		assert.Equal(res.Get("Set-Cookie"), header.Get("Set-Cookie"))

		res.Add("Set-Cookie", "a=b; Path=/; HttpOnly")
		assert.Equal([]string{"foo=bar; Path=/; HttpOnly", "a=b; Path=/; HttpOnly"}, getHeaderValues(header, "Set-Cookie"))

		res.Del("Set-Cookie")
		assert.Equal("", res.Get("Set-Cookie"))

		assert.Equal("", res.Get("Vary"))
		res.Vary("Accept-Encoding")
		assert.Equal("Accept-Encoding", res.Get("Vary"))

		res.Vary("Accept-Language")
		assert.Equal("Accept-Encoding, Accept-Language", strings.Join(res.Header()["Vary"], ", "))

		res.Vary("*")
		assert.Equal("*", res.Get("Vary"))
		res.Vary("Accept-Language")
		assert.Equal("*", strings.Join(res.Header()["Vary"], ", "))
	})

	t.Run("ResetHeader", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		res := ctx.Res
		res.Set("accept", "text/plain")
		res.Set("allow", "GET")
		res.Set("X-Ratelimit-Limit", "10")
		res.Set("retry-after", "3 seconds")
		res.Set("warning", "some warning")
		res.Set("access-control-allow-origin", "*")
		res.Set("Set-Cookie", "Set-Cookie: UserID=JohnDoe; Max-Age=3600; Version=")

		res.ResetHeader()
		assert.Equal("text/plain", res.Get(HeaderAccept))
		assert.Equal("GET", res.Get(HeaderAllow))
		assert.Equal("10", res.Get("X-Ratelimit-Limit"))
		assert.Equal("3 seconds", res.Get(HeaderRetryAfter))
		assert.Equal("some warning", res.Get(HeaderWarning))
		assert.Equal("*", res.Get(HeaderAccessControlAllowOrigin))
		assert.Equal("", res.Get(HeaderSetCookie))

		res.ResetHeader(regexp.MustCompile(`^$`))
		assert.Equal(0, len(res.Header()))
	})

	t.Run("implicit WriteHeader call", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		res := ctx.Res

		assert.Equal(false, res.HeaderWrote())
		assert.Equal(0, res.status)

		res.status = http.StatusUnavailableForLegalReasons
		res.Write([]byte("Hello"))

		assert.Equal(true, res.HeaderWrote())
		assert.Equal(http.StatusUnavailableForLegalReasons, res.status)
		assert.Equal(http.StatusUnavailableForLegalReasons, CtxResult(ctx).StatusCode)
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("explicit WriteHeader call", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		res := ctx.Res

		assert.Equal(false, res.HeaderWrote())
		assert.Equal(0, res.status)

		res.WriteHeader(0)

		assert.Equal(true, res.HeaderWrote())
		assert.Equal(421, res.status)
		assert.Equal(421, CtxResult(ctx).StatusCode)
		assert.Equal("", CtxBody(ctx))

		ctx = CtxTest(app, "GET", "http://example.com/foo", nil)
		res = ctx.Res

		assert.Equal(false, res.HeaderWrote())
		assert.Equal(0, res.status)

		res.body = []byte("Hello")
		res.WriteHeader(0)
		res.Write([]byte("Hello"))

		assert.Equal(true, res.HeaderWrote())
		assert.Equal(200, res.status)
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Equal("Hello", CtxBody(ctx))

		ctx = CtxTest(app, "GET", "http://example.com/foo", nil)
		res = ctx.Res

		assert.Equal(false, res.HeaderWrote())
		assert.Equal(0, res.status)

		res.Write([]byte("Hello"))

		assert.Equal(true, res.HeaderWrote())
		assert.Equal(200, res.status)
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("respond", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Res.respond(200, []byte("Hello"))

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("respond with empty status", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Res.respond(204, []byte("Hello"))

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(204, ctx.Res.Status())
		assert.Equal(204, CtxResult(ctx).StatusCode)
		assert.Equal("", CtxBody(ctx))
	})

	t.Run("WriteHeader should only run once", func(t *testing.T) {
		assert := assert.New(t)

		count := 0
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.After(func() {
			count++
		})
		assert.Equal(false, ctx.Res.HeaderWrote())
		assert.Equal(0, ctx.Res.status)

		var wg sync.WaitGroup
		wg.Add(1000)
		for i := 0; i < 1000; i++ {
			go func() {
				defer wg.Done()
				ctx.Res.WriteHeader(204)
			}()
		}
		wg.Wait()

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(1, count)
		assert.Equal(204, ctx.Res.status)
		assert.Equal(204, CtxResult(ctx).StatusCode)
	})

	t.Run("Should support golang HandlerFunc", func(t *testing.T) {
		assert := assert.New(t)

		count := 0
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.After(func() {
			count++
		})

		assert.Equal(false, ctx.Res.HeaderWrote())
		assert.Equal(0, ctx.Res.status)
		http.NotFound(ctx.Res, ctx.Req)

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(1, count)
		assert.Equal(404, ctx.Res.status)
		assert.Equal(404, ctx.Res.status)
	})

	t.Run("Body", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.End(200, []byte{})

		assert.Equal(200, ctx.Res.Status())
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Equal([]byte{}, ctx.Res.Body())
		assert.Equal("", CtxBody(ctx))

		ctx = CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Res.respond(200, []byte("Hello"))

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(200, ctx.Res.Status())
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Equal([]byte("Hello"), ctx.Res.Body())
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("Body should be nil", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.End(200)

		assert.Equal(200, ctx.Res.Status())
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Nil(ctx.Res.Body())
		assert.Equal("", CtxBody(ctx))

		ctx = CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.End(204, []byte("Hello"))

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(204, ctx.Res.Status())
		assert.Equal(204, CtxResult(ctx).StatusCode)
		assert.Nil(ctx.Res.Body())
		assert.Equal("", CtxBody(ctx))
	})
}

func TestGearResponseFlusher(t *testing.T) {
	assert := assert.New(t)

	app := New()
	app.Use(func(ctx *Context) error {
		ctx.End(200, []byte("OK"))
		ctx.Res.Flush()
		return nil
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(200, res.StatusCode)
	res.Body.Close()
}

func TestGearResponseHijacker(t *testing.T) {
	assert := assert.New(t)

	app := New()
	app.Use(func(ctx *Context) error {
		ctx.End(204)

		conn, rw, err := ctx.Res.Hijack()
		assert.NotNil(conn)
		assert.NotNil(rw)
		assert.Nil(err)
		conn.Close()
		return nil
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)
	res.Body.Close()
}

func TestGearResponseCloseNotifier(t *testing.T) {
	assert := assert.New(t)

	app := New()
	app.Use(func(ctx *Context) error {
		ctx.End(204)
		ch := ctx.Res.CloseNotify()
		assert.NotNil(ch)
		return nil
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)
	res.Body.Close()
}

func TestGearResponsePusher(t *testing.T) {
	t.Run("Should return error if not http2", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			err := ctx.Res.Push("/test", &http.PushOptions{Method: "GET"})
			assert.Equal(err.Error(), "Error: http.Pusher not implemented")

			return ctx.End(200, []byte("OK"))
		})

		srv := app.Start()
		defer app.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		res.Body.Close()
	})

	t.Run("Should support http2 push", func(t *testing.T) {
		assert := assert.New(t)

		const htmlBody = `
<!DOCTYPE html>
<html>
  <head>
    <link href="/hello.css" rel="stylesheet" type="text/css">
  </head>
  <body>
    <h1>Hello, Gear!</h1>
  </body>
</html>`

		const pushBody = `
h1 {
  color: red;
}
`

		app := New()
		router := NewRouter()
		router.Get("/", func(ctx *Context) error {
			err := ctx.Res.Push("/hello.css", &http.PushOptions{Method: "GET"})
			assert.Equal("https", ctx.Protocol())
			assert.Equal(err, http.ErrNotSupported)
			return ctx.HTML(200, htmlBody)
		})
		router.Get("/hello.css", func(ctx *Context) error {
			ctx.Type("text/css")
			return ctx.End(200, []byte(pushBody))
		})
		app.UseHandler(router)

		cond := sync.NewCond(new(sync.Mutex))
		cond.L.Lock()
		go func() {
			cond.Signal()
			app.ListenTLS("127.0.0.1:13333", "./testdata/out/test.crt", "./testdata/out/test.key")
		}()
		defer app.Close()

		cond.Wait()
		time.Sleep(time.Millisecond * 100)
		tr, err := HTTP2Transport("./testdata/out/test.crt", "./testdata/out/test.key")
		assert.Nil(err)
		cli := &http.Client{Transport: tr}
		res, err := cli.Get("https://127.0.0.1:13333")
		fmt.Println(res, err)
		assert.Nil(err)
		assert.Equal("HTTP/2.0", res.Proto)
		res.Body.Close()
	})
}
