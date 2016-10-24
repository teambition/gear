package gear

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/mozillazg/request"
	"github.com/stretchr/testify/require"
)

func NewRequst() *request.Request {
	c := &http.Client{}
	return request.NewRequest(c)
}

func TestGearAppHello(t *testing.T) {
	app := New()
	app.Use(func(ctx *Context) error {
		ctx.End(200, []byte("<h1>Hello!</h1>"))
		return nil
	})
	srv := app.Start()
	defer srv.Close()

	req := NewRequst()
	res, err := req.Get("http://" + srv.Addr().String())
	require.Nil(t, err)
	require.Equal(t, res.StatusCode, 200)
	body, err := res.Text()
	res.Body.Close()
	require.Nil(t, err)
	require.Equal(t, body, "<h1>Hello!</h1>")
}

func TestGearError(t *testing.T) {
	t.Run("ErrorLog and OnError", func(t *testing.T) {
		var buf bytes.Buffer

		app := New()
		app.ErrorLog = log.New(&buf, "TEST: ", 0)
		app.OnError = func(ctx *Context, err error) *textproto.Error {
			ctx.Type("html")
			return NewError(err, 501)
		}

		app.Use(func(ctx *Context) error {
			return errors.New("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		req := NewRequst()
		res, err := req.Get("http://" + srv.Addr().String())
		require.Nil(t, err)
		require.Equal(t, res.StatusCode, 501)
		body, err := res.Text()
		res.Body.Close()
		require.Nil(t, err)
		require.Equal(t, "501 Some error\n", body)
		require.Equal(t, "TEST: Some error\n", buf.String())
	})

	t.Run("panic recovered", func(t *testing.T) {
		var buf bytes.Buffer

		app := New()
		app.ErrorLog = log.New(&buf, "TEST: ", 0)
		app.Use(func(ctx *Context) error {
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		req := NewRequst()
		res, err := req.Get("http://" + srv.Addr().String())
		require.Nil(t, err)
		require.Equal(t, res.StatusCode, 500)
		body, err := res.Text()
		res.Body.Close()
		require.Nil(t, err)

		require.Equal(t, "500 Internal Server Error\n", body)
		log := buf.String()
		require.True(t, strings.Contains(log, "TEST: panic recovered")) // recovered title
		require.True(t, strings.Contains(log, "GET /"))                 // http request content
		require.True(t, strings.Contains(log, "Some error"))            // panic content
	})
}
