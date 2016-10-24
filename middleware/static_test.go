package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/teambition/gear"
)

func TestGearMiddlewareStatic(t *testing.T) {
	require.Panics(t, func() {
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
		res, err := req.Get("http://" + srv.Addr().String() + "/hello.html")
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("HEAD", func(t *testing.T) {
		res, err := req.Head("http://" + srv.Addr().String() + "/hello.html")
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("OPTIONS", func(t *testing.T) {
		res, err := req.Options("http://" + srv.Addr().String() + "/hello.html")
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		require.Equal(t, "GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other method", func(t *testing.T) {
		res, err := req.Patch("http://" + srv.Addr().String() + "/hello.html")
		require.Nil(t, err)
		require.Equal(t, 405, res.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		require.Equal(t, "GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other file", func(t *testing.T) {
		res, err := req.Get("http://" + srv.Addr().String() + "/favicon.ico")
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("404", func(t *testing.T) {
		res, err := req.Get("http://" + srv.Addr().String() + "/none.html")
		require.Nil(t, err)
		require.Equal(t, 404, res.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})
}
