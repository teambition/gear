package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

func TestGearMiddlewareTimeout(t *testing.T) {
	t.Run("hook should run when timeout", func(t *testing.T) {
		assert := assert.New(t)

		count := 0
		app := gear.New()
		req := NewRequst()
		app.Use(NewTimeout(time.Millisecond*100, func(ctx *gear.Context) {
			count++
			ctx.Status(504)
			ctx.String("Service timeout")
		}))
		app.Use(func(ctx *gear.Context) error {
			ts := time.Now()
			c, _ := ctx.WithTimeout(time.Second * 10)
			select {
			case <-ctx.Done(): // this case will always reached
			case <-c.Done(): // this case maybe reached... but elapsed time should be 1 sec.
			}
			assert.True(time.Now().Sub(ts) > time.Millisecond*100)
			return nil
		})
		app.Use(func(ctx *gear.Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := req.Get("http://" + srv.Addr().String())
		assert.Nil(err)
		assert.Equal(1, count)
		assert.Equal(504, res.StatusCode)
		assert.Equal("Service timeout", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("hook should not run when no timeout", func(t *testing.T) {
		assert := assert.New(t)

		count := 0
		app := gear.New()
		req := NewRequst()
		app.Use(NewTimeout(time.Millisecond*100, func(ctx *gear.Context) {
			count++
			ctx.Status(504)
			ctx.String("Service timeout")
		}))
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "Hello")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := req.Get("http://" + srv.Addr().String())
		assert.Nil(err)
		assert.Equal(0, count)
		assert.Equal(200, res.StatusCode)
		assert.Equal("Hello", PickRes(res.Text()).(string))
		res.Body.Close()

		time.Sleep(time.Millisecond * 200)
		assert.Equal(0, count)
	})
}
