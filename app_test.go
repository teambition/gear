package gear

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/textproto"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

// ------Helpers for help test --------
var DefaultClient = &http.Client{}

type GearResponse struct {
	*http.Response
}

func RequestBy(method, url string) (*GearResponse, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func DefaultClientDo(req *http.Request) (*GearResponse, error) {
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func DefaultClientDoWithCookies(req *http.Request, cookies map[string]string) (*http.Response, error) {
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	return DefaultClient.Do(req)
}
func NewRequst(method, url string) (*http.Request, error) {
	return http.NewRequest(method, url, nil)
}

func (resp *GearResponse) OK() bool {
	return resp.StatusCode < 400
}
func (resp *GearResponse) Content() (val []byte, err error) {
	var b []byte
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		if reader, err = gzip.NewReader(resp.Body); err != nil {
			return nil, err
		}
	case "deflate":
		if reader, err = zlib.NewReader(resp.Body); err != nil {
			return nil, err
		}
	default:
		reader = resp.Body
	}

	defer reader.Close()
	if b, err = ioutil.ReadAll(reader); err != nil {
		return nil, err
	}
	return b, err
}

func (resp *GearResponse) Text() (val string, err error) {
	b, err := resp.Content()
	if err != nil {
		return "", err
	}
	return string(b), err
}

//--------- End ---------
func TestGearServer(t *testing.T) {
	t.Run("start with addr", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start("127.0.0.1:3324")
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		res.Body.Close()
	})

	t.Run("failed to listen", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start(":3323")
		defer srv.Close()

		app2 := New()
		app2.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		assert.Panics(func() {
			app2.Start(":3323")
		})

		app3 := New()
		app3.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		err := app3.Listen(":3323")
		assert.NotNil(err)

		app4 := New()
		app4.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		err = app3.ListenTLS(":3323", "", "")
		assert.NotNil(err)

		go func() {
			time.Sleep(time.Second)
			srv.Close()
		}()
		err = srv.Wait()
		assert.NotNil(err)
	})
}

func TestGearAppHello(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Panics(func() {
			app.toServeHandler()
		})

		app.Use(func(ctx *Context) error {
			return ctx.End(200, []byte("<h1>Hello!</h1>"))
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("<h1>Hello!</h1>", PickRes(res.Text()).(string))
		res.Body.Close()
	})
}

type testOnError struct{}

// OnError implemented OnError interface.
func (o *testOnError) OnError(ctx *Context, err error) *Error {
	ctx.Type(MIMETextHTMLCharsetUTF8)
	return ParseError(err, 503)
}

func TestGearError(t *testing.T) {
	t.Run("ErrorLog and OnError", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		assert.Panics(func() {
			app.Set("AppLogger", struct{}{})
		})
		assert.Panics(func() {
			app.Set("AppOnError", struct{}{})
		})
		app.Set("AppLogger", log.New(&buf, "TEST: ", 0))
		app.Set("AppOnError", &testOnError{})

		app.Use(func(ctx *Context) error {
			return errors.New("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(503, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal("Some error", PickRes(res.Text()).(string))
		assert.Equal("TEST: {Code: 503, Msg: Some error, Meta: Some error}\n", buf.String())
		res.Body.Close()
	})

	t.Run("return nil HTTPError", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set("AppLogger", log.New(&buf, "TEST: ", 0))
		app.Set("AppOnError", &testOnError{})

		app.Use(func(ctx *Context) error {
			var err *Error
			ctx.Status(204)
			return err
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		assert.Equal("", res.Header.Get(HeaderContentType))
		assert.Equal("", PickRes(res.Text()).(string))
		assert.Equal("", buf.String())
		res.Body.Close()
	})

	t.Run("panic recovered", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set("AppLogger", log.New(&buf, "TEST: ", 0))
		app.Use(func(ctx *Context) error {
			ctx.Status(400)
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("panic recovered: Some error", PickRes(res.Text()).(string))

		log := buf.String()
		assert.True(strings.Contains(log, "panic recovered"))
		assert.True(strings.Contains(log, "github.com/teambition/gear"))
		res.Body.Close()
	})
}

type testHTTPError1 struct {
	c int
	m string
	x bool
}

func (e *testHTTPError1) Error() string {
	return e.m
}

func (e *testHTTPError1) Status() int {
	return e.c
}

type testHTTPError2 struct {
	c int
	m string
	x bool
}

func (e testHTTPError2) Error() string {
	return e.m
}

func (e testHTTPError2) Status() int {
	return e.c
}

func TestGearParseError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert := assert.New(t)

		var err0 error
		err := ParseError(err0)
		assert.True(err == nil)

		var err1 *testHTTPError1
		err = ParseError(err1)
		assert.True(err == nil)

		var err2 *testHTTPError2
		err = ParseError(err2)
		assert.True(err == nil)

		var err3 *textproto.Error
		err = ParseError(err3)
		assert.True(err == nil)

		var err4 *Error
		err = ParseError(err4)
		assert.True(err == nil)

		var err5 HTTPError
		err = ParseError(err5)
		assert.True(err == nil)

		err6 := func() error {
			var e *testHTTPError1
			return e
		}()
		// fmt.Println(err6, err6 == nil) // <nil> false
		err = ParseError(err6)
		assert.True(err == nil)

		err7 := func() *Error {
			var e *Error
			return e
		}()
		assert.True(err7 == nil)

		err8 := func() HTTPError {
			var e *testHTTPError1
			return e
		}()
		// fmt.Println(err8, err8 == nil) // <nil> false
		err = ParseError(err8)
		assert.True(err == nil)
	})

	t.Run("Error", func(t *testing.T) {
		err1 := &Error{Code: 400, Msg: "test"}
		err := ParseError(err1)
		EqualPtr(t, err1, err)

		err2 := func() error {
			return &Error{Code: 400, Msg: "test"}
		}()
		err = ParseError(err2)
		EqualPtr(t, err2, err)
	})

	t.Run("textproto.Error", func(t *testing.T) {
		err1 := &textproto.Error{Code: 400, Msg: "test"}
		err := ParseError(err1)
		EqualPtr(t, err1, err.Meta)

		err2 := func() error {
			return &textproto.Error{Code: 400, Msg: "test"}
		}()
		err = ParseError(err2)
		EqualPtr(t, err2, err.Meta)
	})

	t.Run("custom HTTPError", func(t *testing.T) {
		err1 := &testHTTPError1{c: 400, m: "test"}
		err := ParseError(err1)
		EqualPtr(t, err1, err.Meta)

		err2 := func() error {
			return &testHTTPError1{c: 400, m: "test"}
		}()
		err = ParseError(err2)
		EqualPtr(t, err2, err.Meta)

		err3 := &testHTTPError2{c: 400, m: "test"}
		err = ParseError(err3)
		EqualPtr(t, err3, err.Meta)

		err4 := func() error {
			return &testHTTPError2{c: 400, m: "test"}
		}()
		err = ParseError(err4)
		EqualPtr(t, err4, err.Meta)
	})

	t.Run("error", func(t *testing.T) {
		assert := assert.New(t)

		err1 := errors.New("test")
		err := ParseError(err1)
		EqualPtr(t, err1, err.Meta)
		assert.Equal(err.Code, 500)

		err2 := func() error {
			return errors.New("test")
		}()
		err = ParseError(err2, 0)
		EqualPtr(t, err2, err.Meta)
		assert.Equal(err.Code, 500)

		err3 := func() error {
			return errors.New("test")
		}()
		err = ParseError(err3, 400)
		EqualPtr(t, err3, err.Meta)
		assert.Equal(err.Code, 400)
	})
}

func TestGearAppTimeout(t *testing.T) {
	t.Run("respond 503 when timeout", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		assert.Panics(func() {
			app.Set("AppTimeout", struct{}{})
		})
		app.Set("AppTimeout", time.Millisecond*100)

		app.Use(func(ctx *Context) error {
			time.Sleep(time.Millisecond * 300)
			return ctx.HTML(200, "OK")
		})
		app.Use(func(ctx *Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(503, res.StatusCode)
		assert.Equal("context deadline exceeded", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("respond 503 when cancel", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set("AppTimeout", time.Millisecond*100)

		app.Use(func(ctx *Context) error {
			ctx.Cancel()
			time.Sleep(time.Millisecond)
			ctx.String(500, "some data")
			return nil
		})
		app.Use(func(ctx *Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(503, res.StatusCode)
		assert.Equal("context canceled", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("respond 200", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set("AppTimeout", time.Millisecond*100)

		app.Use(func(ctx *Context) error {
			time.Sleep(time.Millisecond * 10)
			return ctx.End(200, []byte("OK"))
		})
		app.Use(func(ctx *Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
		res.Body.Close()

		time.Sleep(time.Millisecond * 500)
	})
}

func TestGearWrapHandler(t *testing.T) {
	assert := assert.New(t)

	app := New()
	count := 0
	app.Use(func(ctx *Context) error {
		ctx.After(func() {
			count++
			assert.Equal(2, count)
		})
		ctx.OnEnd(func() {
			count++
			assert.Equal(3, count)
		})
		count++
		assert.Equal(1, count)
		ctx.String(400, "some error")
		return nil
	})

	app.Use(WrapHandler(http.NotFoundHandler()))
	app.Use(func(ctx *Context) error {
		panic("this middleware unreachable")
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(3, count)
	assert.Equal(404, res.StatusCode)
	res.Body.Close()
}

func TestGearWrapHandlerFunc(t *testing.T) {
	assert := assert.New(t)

	app := New()
	count := 0
	app.Use(func(ctx *Context) error {
		ctx.After(func() {
			count++
			assert.Equal(2, count)
		})
		ctx.OnEnd(func() {
			count++
			assert.Equal(3, count)
		})
		count++
		assert.Equal(1, count)
		ctx.String(400, "some error")
		return nil
	})

	app.Use(WrapHandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(req.URL.Path))
	}))
	app.Use(func(ctx *Context) error {
		panic("this middleware unreachable")
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(3, count)
	assert.Equal(404, res.StatusCode)
	res.Body.Close()
}

type WriterTest struct {
	res http.ResponseWriter
}

func (wt *WriterTest) WriteHeader(code int) {
	wt.res.WriteHeader(code)
}

func (wt *WriterTest) Header() http.Header {
	return wt.res.Header()
}

func (wt *WriterTest) Write(b []byte) (int, error) {
	return 0, errors.New("can't write")
}

func TestGearWrapResponseWriter(t *testing.T) {
	assert := assert.New(t)

	app := New()
	var buf bytes.Buffer
	app.Set("AppLogger", log.New(&buf, "TEST: ", 0))
	app.Use(func(ctx *Context) error {
		ctx.Res.res = &WriterTest{ctx.Res.res}
		ctx.String(200, "OK")
		return nil
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(200, res.StatusCode)
	res.Body.Close()

	log := buf.String()
	assert.True(strings.Contains(log, "can't write"))
}
