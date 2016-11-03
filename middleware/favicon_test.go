package middleware

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/mozillazg/request"
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

func NewRequst() *request.Request {
	c := &http.Client{}
	return request.NewRequest(c)
}

func TestGearMiddlewareFavicon(t *testing.T) {
	assert.Panics(t, func() {
		NewFavicon("../testdata/favicon1.ico")
	})

	app := gear.New()
	app.Use(NewFavicon("../testdata/favicon.ico"))
	srv := app.Start()
	defer srv.Close()

	req := NewRequst()

	t.Run("GET", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Get("http://" + srv.Addr().String() + "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("HEAD", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Head("http://" + srv.Addr().String() + "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("OPTIONS", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Options("http://" + srv.Addr().String() + "/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other method", func(t *testing.T) {
		assert := assert.New(t)

		res, err := req.Patch("http://" + srv.Addr().String() + "/favicon.ico")
		assert.Nil(err)
		assert.Equal(405, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})
}
