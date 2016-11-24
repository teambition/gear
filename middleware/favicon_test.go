package middleware

import (
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

// ----- Test Helpers -----

func EqualPtr(t *testing.T, a, b interface{}) {
	assert.Equal(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func NotEqualPtr(t *testing.T, a, b interface{}) {
	assert.NotEqual(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func PickRes(res interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}
	return res
}

func PickError(res interface{}, err error) error {
	return err
}



func TestGearMiddlewareFavicon(t *testing.T) {
	assert.Panics(t, func() {
		NewFavicon("../testdata/favicon1.ico")
	})

	app := gear.New()
	app.Use(NewFavicon("../testdata/favicon.ico"))
	app.Use(func(ctx *gear.Context) error {
		return ctx.HTML(200, "OK")
	})
	srv := app.Start()
	defer srv.Close()

	t.Run("GET", func(t *testing.T) {
		assert := assert.New(t)

		res, err = RequestBy("GET", "http://" + srv.Addr().String()+ "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("HEAD", func(t *testing.T) {
		assert := assert.New(t)

		res, err = RequestBy("HEAD", "http://" + srv.Addr().String()+ "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("OPTIONS", func(t *testing.T) {
		assert := assert.New(t)

		res, err = RequestBy("OPTIONS", "http://" + srv.Addr().String()+ "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other method", func(t *testing.T) {
		assert := assert.New(t)

		res, err = RequestBy("PATCH", "http://" + srv.Addr().String()+"/favicon.ico")
		assert.Nil(err)
		assert.Equal(405, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other path", func(t *testing.T) {
		assert := assert.New(t)

		res, err = RequestBy("PATCH", "http://" + srv.Addr().String()++ "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		res.Body.Close()
	})
}
