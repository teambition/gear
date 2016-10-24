package middleware

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/teambition/gear"
)

func TestGearMiddlewareTimeout(t *testing.T) {
	app := gear.New()
	app.Use(NewTimeout(time.Second, func(ctx *gear.Context) {
		ctx.Status(504)
		ctx.String("Service timeout")
	}))
	app.Use(func(ctx *gear.Context) error {
		ts := time.Now()
		c, _ := ctx.WithTimeout(time.Second * 2)
		select {
		case <-ctx.Done(): // this case will always reached
		case <-c.Done(): // this case maybe reached... but elapsed time should be 1 sec.
		}
		require.True(t, (time.Now().Sub(ts)-time.Second) < time.Millisecond*200)
		return nil
	})
	app.Use(func(ctx *gear.Context) error {
		panic("this middleware unreachable")
	})
	srv := app.Start()
	defer srv.Close()

	url := "http://" + srv.Addr().String()
	res, err := http.Get(url)
	require.Nil(t, err)
	require.Equal(t, res.StatusCode, 504)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	require.Nil(t, err)
	require.Equal(t, body, []byte("Service timeout"))
}
