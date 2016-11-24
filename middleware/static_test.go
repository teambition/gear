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
	assert.NotPanics(t, func() {
		NewStatic(StaticOptions{
			Root:        "",
			Prefix:      "",
			StripPrefix: true,
		})
	})

	app := gear.New()
	app.Set("AppCompress", &gear.DefaultCompress{})

	app.Use(NewStatic(StaticOptions{
		Root:        "../testdata",
		Prefix:      "/static",
		StripPrefix: true,
	}))
	app.Use(NewStatic(StaticOptions{
		Root:        "../testdata",
		Prefix:      "/",
		StripPrefix: false,
	}))
	srv := app.Start()
	defer srv.Close()

	t.Run("GET", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("GET with StripPrefix", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/static/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("HEAD", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("HEAD", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("OPTIONS", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("OPTIONS", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other method", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("PATCH", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(405, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other file", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("Should compress", func(t *testing.T) {
		assert := assert.New(t)

		req, _ := NewRequst("GET", "http://"+srv.Addr().String()+"/README.md")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		res, err := DefaultClientDo(req)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("gzip", res.Header.Get(gear.HeaderContentEncoding))
		res.Body.Close()
	})

	t.Run("404", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/none.html")
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})
}
