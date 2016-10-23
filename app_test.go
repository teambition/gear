package gear_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/teambition/gear"
)

type Response struct {
	Header http.Header
	Body   []byte
}

func TestGearAppHello(t *testing.T) {
	app := gear.New()
	app.Use(func(ctx *gear.Context) error {
		ctx.End(200, []byte("<h1>Hello!</h1>"))
		return nil
	})
	srv := app.Start()
	defer srv.Close()

	url := "http://" + srv.Addr().String()
	res, err := http.Get(url)
	require.Nil(t, err)
	require.Equal(t, res.StatusCode, 200)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	require.Nil(t, err)
	require.Equal(t, body, []byte("<h1>Hello!</h1>"))
}

func TestGearError(t *testing.T) {
	app := gear.New()
	app.OnError = func(ctx *gear.Context, err error) gear.HTTPError {
		ctx.Type("html")
		return gear.NewError(err, 501)
	}

	app.Use(func(ctx *gear.Context) error {
		return errors.New("Some 501 error")
	})
	srv := app.Start()
	defer srv.Close()

	url := "http://" + srv.Addr().String()
	res, err := http.Get(url)
	require.Nil(t, err)
	require.Equal(t, res.StatusCode, 501)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	require.Nil(t, err)
	require.Equal(t, body, []byte("Some 501 error"))
}
