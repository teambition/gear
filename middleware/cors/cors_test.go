package cors

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

var DefaultClient = &http.Client{}

func TestGearMiddlewareCORS(t *testing.T) {
	app := gear.New()
	app.Use(New(Options{
		AllowOrigins:  []string{"test.org"},
		AllowMethods:  []string{http.MethodGet, http.MethodPut},
		AllowHeaders:  []string{"CORS-Test-Allow-Header"},
		ExposeHeaders: []string{"CORS-Test-Expose-Header"},
		MaxAge:        10 * time.Second,
		Credentials:   true,
	}))
	app.Use(func(ctx *gear.Context) error {
		return ctx.HTML(200, "OK")
	})
	srv := app.Start()
	defer srv.Close()
	url := "http://" + srv.Addr().String()

	t.Run("Should not set Access-Control-Allow-Origin when request Origin header missing", func(t *testing.T) {
		assert := assert.New(t)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		assert.Nil(err)
		res, err := DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal("Origin", res.Header.Get(gear.HeaderVary))
		assert.Equal("", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
		assert.Equal(http.StatusOK, res.StatusCode)
	})

	t.Run("Should set Access-Control-Allow-Origin when request Origin header is not qualified", func(t *testing.T) {
		assert := assert.New(t)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		assert.Nil(err)
		req.Header.Set(gear.HeaderOrigin, "not-allowed.org")
		res, err := DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.Equal("Origin", res.Header.Get(gear.HeaderVary))
		assert.Equal("", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
	})

	t.Run("Should set Access-Control-Allow-Origin when request Origin header is qualified", func(t *testing.T) {
		assert := assert.New(t)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		assert.Nil(err)
		req.Header.Set(gear.HeaderOrigin, "test.org")
		res, err := DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.Equal("Origin", res.Header.Get(gear.HeaderVary))
		assert.Equal("test.org", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
	})

	t.Run("Should not set Access-Control-Allow-Origin when is invalid prefilghted request", func(t *testing.T) {
		assert := assert.New(t)

		req, err := http.NewRequest(http.MethodOptions, url, nil)
		assert.Nil(err)
		req.Header.Set(gear.HeaderOrigin, "test.org")
		res, err := DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.Equal([]string{"0"}, res.Header["Content-Length"])
		assert.Equal([]string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"}, res.Header["Vary"])
		assert.Equal("", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
	})

	t.Run("Should set headers as specfied in options when prefilghted request is valid", func(t *testing.T) {
		assert := assert.New(t)

		req, err := http.NewRequest(http.MethodOptions, url, nil)
		assert.Nil(err)
		req.Header.Set(gear.HeaderOrigin, "test.org")
		req.Header.Set(gear.HeaderAccessControlRequestMethod, http.MethodPut)
		res, err := DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.Equal([]string{"0"}, res.Header["Content-Length"])
		assert.Equal([]string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"}, res.Header["Vary"])
		assert.Equal("10", res.Header.Get(gear.HeaderAccessControlMaxAge))
		assert.Equal("test.org", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
		assert.Equal("GET, PUT", res.Header.Get(gear.HeaderAccessControlAllowMethods))
		assert.Equal("true", res.Header.Get(gear.HeaderAccessControlAllowCredentials))
		assert.Equal("CORS-Test-Allow-Header", res.Header.Get(gear.HeaderAccessControlAllowHeaders))
	})

	t.Run("Should set headers as specfied in options when is simple request", func(t *testing.T) {
		assert := assert.New(t)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		assert.Nil(err)
		req.Header.Set(gear.HeaderOrigin, "test.org")
		res, err := DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(http.StatusOK, res.StatusCode)
		assert.Equal("Origin", res.Header.Get(gear.HeaderVary))
		assert.Equal("test.org", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
		assert.Equal("true", res.Header.Get(gear.HeaderAccessControlAllowCredentials))
		assert.Equal("CORS-Test-Expose-Header", res.Header.Get(gear.HeaderAccessControlExposeHeaders))
	})

	t.Run("default options", func(t *testing.T) {
		app = gear.New()
		app.Use(New(Options{Credentials: true}))
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "OK")
		})
		srv = app.Start()
		defer srv.Close()
		url = "http://" + srv.Addr().String()

		t.Run("Should set default allowed methods and headers", func(t *testing.T) {
			assert := assert.New(t)

			req, err := http.NewRequest(http.MethodOptions, url, nil)
			assert.Nil(err)
			req.Header.Set(gear.HeaderOrigin, "test.org")
			req.Header.Set(gear.HeaderAccessControlRequestMethod, http.MethodPut)
			res, err := DefaultClient.Do(req)

			assert.Nil(err)
			assert.Equal(http.StatusOK, res.StatusCode)
			assert.Equal([]string{"0"}, res.Header["Content-Length"])
			assert.Equal([]string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"}, res.Header["Vary"])
			assert.Equal("test.org", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
			assert.Equal("true", res.Header.Get(gear.HeaderAccessControlAllowCredentials))
			assert.Equal(strings.Join(defaultAllowMethods, ", "),
				res.Header.Get(gear.HeaderAccessControlAllowMethods))
		})
	})

	t.Run("Custom AllowOriginsValidator", func(t *testing.T) {
		app = gear.New()
		app.Use(New(Options{
			AllowOriginsValidator: func(origin string, _ *gear.Context) string {
				if origin == "not-allow-origin.com" {
					return ""
				}
				return "test-origin.com"
			},
		}))
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "OK")
		})
		srv = app.Start()
		defer srv.Close()
		url = "http://" + srv.Addr().String()

		t.Run("Should returns the custom allowed origin returned by validator", func(t *testing.T) {
			assert := assert.New(t)

			req, err := http.NewRequest(http.MethodGet, url, nil)
			assert.Nil(err)
			req.Header.Set(gear.HeaderOrigin, "test.com")
			res, err := DefaultClient.Do(req)

			assert.Nil(err)
			assert.Equal(http.StatusOK, res.StatusCode)
			assert.Equal("Origin", res.Header.Get(gear.HeaderVary))
			assert.Equal("test-origin.com", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
		})

		t.Run("Should not set Access-Control-Allow-Origin when not pass the validator", func(t *testing.T) {
			assert := assert.New(t)

			req, err := http.NewRequest(http.MethodGet, url, nil)
			assert.Nil(err)
			req.Header.Set(gear.HeaderOrigin, "not-allow-origin.com")
			res, err := DefaultClient.Do(req)

			assert.Nil(err)
			assert.Equal(http.StatusOK, res.StatusCode)
			assert.Equal("Origin", res.Header.Get(gear.HeaderVary))
			assert.Equal("", res.Header.Get(gear.HeaderAccessControlAllowOrigin))
		})
	})
}
