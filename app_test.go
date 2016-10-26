package gear

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/mozillazg/request"
	"github.com/stretchr/testify/require"
)

// ----- Test Helpers -----

func EqualPtr(t *testing.T, a, b interface{}) {
	require.Equal(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func NotEqualPtr(t *testing.T, a, b interface{}) {
	require.NotEqual(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
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

// ----- Test App -----

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
	require.Equal(t, 200, res.StatusCode)
	require.Equal(t, "<h1>Hello!</h1>", PickRes(res.Text()).(string))
	res.Body.Close()
}

func TestGearError(t *testing.T) {
	t.Run("ErrorLog and OnError", func(t *testing.T) {
		var buf bytes.Buffer

		app := New()
		app.ErrorLog = log.New(&buf, "TEST: ", 0)
		app.OnError = func(ctx *Context, err error) *Error {
			ctx.Type("html")
			return ParseError(err, 501)
		}

		app.Use(func(ctx *Context) error {
			return errors.New("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		req := NewRequst()
		res, err := req.Get("http://" + srv.Addr().String())
		require.Nil(t, err)
		require.Equal(t, 501, res.StatusCode)
		require.Equal(t, "Some error\n", PickRes(res.Text()).(string))
		require.Equal(t, "TEST: Some error\n", buf.String())
		res.Body.Close()
	})

	t.Run("panic recovered", func(t *testing.T) {
		var buf bytes.Buffer

		app := New()
		app.ErrorLog = log.New(&buf, "TEST: ", 0)
		app.Use(func(ctx *Context) error {
			ctx.Status(400)
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		req := NewRequst()
		res, err := req.Get("http://" + srv.Addr().String())
		require.Nil(t, err)
		require.Equal(t, 500, res.StatusCode)
		require.Equal(t, "Internal Server Error\n", PickRes(res.Text()).(string))

		log := buf.String()
		require.True(t, strings.Contains(log, "TEST: panic recovered")) // recovered title
		require.True(t, strings.Contains(log, "GET /"))                 // http request content
		require.True(t, strings.Contains(log, "Some error"))            // panic content
		res.Body.Close()
	})
}
