package gear

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGearServer(t *testing.T) {
	t.Run("app.Close immediately", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		res.Body.Close()
		assert.Nil(app.Close())
	})

	t.Run("app.Close gracefully", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		res.Body.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.Nil(app.Close(ctx))
	})

	t.Run("start with addr", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start("127.0.0.1:3324")
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		res.Body.Close()
	})

	t.Run("failed to listen", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start("127.0.0.1:3323")
		defer srv.Close()

		app2 := New()
		app2.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		assert.Panics(func() {
			app2.Start("127.0.0.1:3323")
		})

		app3 := New()
		app3.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		err := app3.Listen("127.0.0.1:3323")
		assert.NotNil(err)

		app4 := New()
		app4.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		err = app3.ListenTLS("127.0.0.1:3323", "", "")
		assert.NotNil(err)

		go func() {
			time.Sleep(time.Second)
			srv.Close()
		}()
		err = srv.Wait()
		assert.NotNil(err)
	})
}

func TestGearAppHello(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Equal("development", app.Env())
		app.Use(func(ctx *Context) error {
			return ctx.End(200, []byte("<h1>Hello!</h1>"))
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("<h1>Hello!</h1>", PickRes(res.Text()).(string))
		res.Body.Close()
	})
}

func TestGearError(t *testing.T) {
	t.Run("ErrorLog and OnError", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		assert.Panics(func() {
			app.Set(SetLogger, struct{}{})
		})
		assert.Panics(func() {
			app.Set(SetOnError, struct{}{})
		})
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Set(SetOnError, func(ctx *Context, err HTTPError) {
			ctx.Type(MIMETextHTMLCharsetUTF8)
		})

		app.Use(func(ctx *Context) error {
			return errors.New("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("application/json; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal(`{"error":"Internal Server Error","message":"Some error"}`, PickRes(res.Text()).(string))
		assert.True(strings.Contains(buf.String(),
			`TEST: Error{Code:500, Err:"Internal Server Error", Msg:"Some error", Data:<nil>, Stack:"\t`))
		res.Body.Close()
	})

	t.Run("return HTTPError as text", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Set(SetOnError, func(ctx *Context, err HTTPError) {
			ctx.Type(MIMETextPlainCharsetUTF8)
			ctx.End(err.Status(), []byte(err.Error()))
		})

		app.Use(func(ctx *Context) error {
			return errors.New("some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal("Internal Server Error: some error", PickRes(res.Text()).(string))
		assert.Equal("", buf.String())
		res.Body.Close()
	})

	t.Run("return router error as JSON", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		router := NewRouter()
		router.Get("/", func(ctx *Context) error {
			return errors.New("some error")
		})
		app.UseHandler(router)
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("application/json; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal(`{"error":"Internal Server Error","message":"some error"}`, PickRes(res.Text()).(string))
		assert.True(strings.Contains(buf.String(),
			`TEST: Error{Code:500, Err:"Internal Server Error", Msg:"some error", Data:<nil>, Stack:"\t`))
		res.Body.Close()
	})

	t.Run("panic recovered", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Use(func(ctx *Context) error {
			ctx.Status(400)
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"Internal Server Error","message":"Some error"}`, PickRes(res.Text()).(string))

		log := buf.String()
		assert.True(strings.Contains(log, "github.com/teambition/gear"))
		res.Body.Close()
	})
}

func TestGearSetTimeout(t *testing.T) {
	t.Run("respond 504 when timeout", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		assert.Panics(func() {
			app.Set(SetTimeout, struct{}{})
		})
		app.Set(SetTimeout, time.Millisecond*100)

		app.Use(func(ctx *Context) error {
			time.Sleep(time.Millisecond * 300)
			return ctx.HTML(200, "OK")
		})
		app.Use(func(ctx *Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(504, res.StatusCode)
		assert.Equal(`{"error":"Gateway Timeout","message":"context deadline exceeded"}`, PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("respond 500 when cancel", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set(SetTimeout, time.Millisecond*100)

		app.Use(func(ctx *Context) error {
			ctx.Cancel()
			time.Sleep(time.Millisecond)
			return ctx.End(200, []byte("some data"))
		})
		app.Use(func(ctx *Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		res.Body.Close()
	})

	t.Run("respond 200", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set(SetTimeout, time.Millisecond*100)

		app.Use(func(ctx *Context) error {
			time.Sleep(time.Millisecond * 10)
			return ctx.End(200, []byte("OK"))
		})
		app.Use(func(ctx *Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()

		time.Sleep(time.Millisecond * 500)
	})
}

func TestGearSetWithContext(t *testing.T) {
	t.Run("respond 200", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Panics(func() {
			app.Set(SetWithContext, func() {})
		})

		key := struct{}{}
		app.Set(SetWithContext, func(r *http.Request) context.Context {
			return context.WithValue(r.Context(), key, "Hello Context")
		})

		app.Use(func(ctx *Context) error {
			value := ctx.Value(key).(string)
			return ctx.End(200, []byte(value))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("Hello Context", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("should panic", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Set(SetWithContext, func(r *http.Request) context.Context {
			return context.WithValue(context.Background(), "key", "Hello Context")
		})
		count := 0
		app.Use(func(ctx *Context) error {
			count++
			return ctx.End(204)
		})

		srv := app.Start()
		defer srv.Close()

		_, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.NotNil(err)
		assert.Equal(0, count)
	})
}
