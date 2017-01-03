package gear

import (
	"net/http"
	"regexp"
	"sync"
	"testing"

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
	})

	t.Run("ResetHeader", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		res := ctx.Res
		res.Set("accept", "text/plain")
		res.Set("allow", "GET")
		res.Set("retry-after", "3 seconds")
		res.Set("warning", "some warning")
		res.Set("access-control-allow-origin", "*")
		res.Set("Set-Cookie", "Set-Cookie: UserID=JohnDoe; Max-Age=3600; Version=")

		res.ResetHeader()
		assert.Equal("text/plain", res.Get(HeaderAccept))
		assert.Equal("GET", res.Get(HeaderAllow))
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

		res.bodyLength = len([]byte("Hello"))
		res.WriteHeader(0)
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
		assert.Equal("5", CtxResult(ctx).Header.Get(HeaderContentLength))
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("respond with empty status", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Res.respond(204, []byte("Hello"))

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(204, ctx.Status())
		assert.Equal(204, CtxResult(ctx).StatusCode)
		assert.Equal("", CtxResult(ctx).Header.Get(HeaderContentLength))
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

func TestGearCheckStatus(t *testing.T) {
	assert := assert.New(t)
	assert.False(IsStatusCode(1))
	assert.True(IsStatusCode(100))

	assert.False(isRedirectStatus(200))
	assert.True(isRedirectStatus(301))

	assert.False(isEmptyStatus(200))
	assert.True(isEmptyStatus(204))
}
