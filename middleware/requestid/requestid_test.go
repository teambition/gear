package requestid

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

func TestGearMiddlewareRequestID(t *testing.T) {
	t.Run("request without X-Request-ID", func(t *testing.T) {
		assert := assert.New(t)

		app := gear.New()
		app.Use(New())
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
		assert.Nil(err)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.NotEqual("", res.Header.Get(gear.HeaderXRequestID))
	})

	t.Run("request with X-Request-ID", func(t *testing.T) {
		assert := assert.New(t)

		app := gear.New()
		app.Use(New())
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		rid := generator()

		req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
		assert.Nil(err)
		req.Header.Set(gear.HeaderXRequestID, rid)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.Equal(rid, res.Header.Get(gear.HeaderXRequestID))
	})

	t.Run("custom id generator", func(t *testing.T) {
		assert := assert.New(t)

		app := gear.New()

		rid := "example rid"
		app.Use(New(Options{
			Generator: func() string {
				return rid
			},
		}))
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
		assert.Nil(err)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.Equal(rid, res.Header.Get(gear.HeaderXRequestID))
	})
}
