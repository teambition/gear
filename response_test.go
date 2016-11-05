package gear

import (
	"net/http"
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

		res.Add("Link", "<http://localhost/>")
		res.Add("Link", "<http://localhost:3000/>")
		assert.Equal(res.Get("link"), "<http://localhost/>")
		assert.Equal(res.Get("Link"), header.Get("Link"))

		res.Set("Set-Cookie", "foo=bar; Path=/; HttpOnly")
		assert.Equal(res.Get("Set-Cookie"), header.Get("Set-Cookie"))

		res.Del("Link")
		assert.Equal("", res.Get("link"))
	})

	t.Run("implicit WriteHeader call", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		res := ctx.Res

		assert.Equal(false, res.HeaderWrote())
		assert.Equal(0, res.Status)

		res.Status = http.StatusUnavailableForLegalReasons
		res.Write([]byte("Hello"))

		assert.Equal(true, res.HeaderWrote())
		assert.Equal(http.StatusUnavailableForLegalReasons, res.Status)
		assert.Equal(http.StatusUnavailableForLegalReasons, CtxResult(ctx).StatusCode)
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("explicit WriteHeader call", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		res := ctx.Res

		assert.Equal(false, res.HeaderWrote())
		assert.Equal(0, res.Status)

		res.WriteHeader(0)

		assert.Equal(true, res.HeaderWrote())
		assert.Equal(444, res.Status)
		assert.Equal(444, CtxResult(ctx).StatusCode)
		assert.Equal("", CtxBody(ctx))

		ctx = CtxTest(app, "GET", "http://example.com/foo", nil)
		res = ctx.Res

		assert.Equal(false, res.HeaderWrote())
		assert.Equal(0, res.Status)

		res.Body = []byte("Hello")
		res.WriteHeader(0)
		res.Write(res.Body)

		assert.Equal(true, res.HeaderWrote())
		assert.Equal(200, res.Status)
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("respond", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)

		ctx.String("Hello")
		ctx.Res.respond()

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(200, ctx.Res.Status)
		assert.Equal(200, CtxResult(ctx).StatusCode)
		assert.Equal("Hello", CtxBody(ctx))
	})

	t.Run("WriteHeader should only run once", func(t *testing.T) {
		assert := assert.New(t)

		count := 0
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.After(func(_ *Context) {
			count++
		})
		assert.Equal(false, ctx.Res.HeaderWrote())
		assert.Equal(0, ctx.Res.Status)

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
		assert.Equal(204, ctx.Res.Status)
		assert.Equal(204, CtxResult(ctx).StatusCode)
	})

	t.Run("Should support golang HandlerFunc", func(t *testing.T) {
		assert := assert.New(t)

		count := 0
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.After(func(_ *Context) {
			count++
		})

		assert.Equal(false, ctx.Res.HeaderWrote())
		assert.Equal(0, ctx.Res.Status)
		http.NotFound(ctx.Res, ctx.Req)

		assert.Equal(true, ctx.Res.HeaderWrote())
		assert.Equal(1, count)
		assert.Equal(404, ctx.Res.Status)
		assert.Equal(404, ctx.Res.Status)
	})
}
