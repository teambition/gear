package gear

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGearAppHello(t *testing.T) {
	app := New()
	app.Use(func(ctx *Context) error {
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
	app := New()
	app.OnError = func(ctx *Context, err error) *textproto.Error {
		ctx.Type("html")
		return NewError(err, 501)
	}

	app.Use(func(ctx *Context) error {
		return errors.New("Some error")
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
	require.Equal(t, body, []byte("501 Some error"))
}
