package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

func TestGearMiddlewareStatic(t *testing.T) {
	assert.Panics(t, func() {
		NewStatic(StaticOptions{
			Root:        "../testdata1",
			Prefix:      "/",
			StripPrefix: false,
		})
	})

	app := gear.New()
	app.Use(NewStatic(StaticOptions{
		Root:        "../testdata",
		Prefix:      "/",
		StripPrefix: false,
	}))
	srv := app.Start()
	defer srv.Close()

	req := NewRequst()

	t.Run("GET", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Get("http://" + srv.Addr().String() + "/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("HEAD", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Head("http://" + srv.Addr().String() + "/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("OPTIONS", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Options("http://" + srv.Addr().String() + "/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other method", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Patch("http://" + srv.Addr().String() + "/hello.html")
		assert.Nil(err)
		assert.Equal(405, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other file", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Get("http://" + srv.Addr().String() + "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("404", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Get("http://" + srv.Addr().String() + "/none.html")
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})
}
