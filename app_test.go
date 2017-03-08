package gear

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"


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
	t.Run("app.Close immediately", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		res.Body.Close()
		assert.Nil(app.Close())
	})

	t.Run("app.Close gracefully", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.End(204)
		})
		srv := app.Start()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		res.Body.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.Nil(app.Close(ctx))
	})

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
		assert.Equal("development", app.Env())
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

func TestGearError(t *testing.T) {
	t.Run("ErrorLog and OnError", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		assert.Panics(func() {
			app.Set(SetLogger, struct{}{})
		})
		assert.Panics(func() {
			app.Set(SetOnError, struct{}{})
		})
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Set(SetOnError, func(ctx *Context, err HTTPError) {
			ctx.Type(MIMETextHTMLCharsetUTF8)
		})

		app.Use(func(ctx *Context) error {
			return errors.New("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal("Some error", PickRes(res.Text()).(string))
		assert.True(strings.Contains(buf.String(),
			`TEST: Error{Code:500, Msg:"Some error", Meta:<nil>, Stack:"\t`))
		res.Body.Close()
	})

	t.Run("return HTTPError as JSON", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Set(SetOnError, func(ctx *Context, err HTTPError) {
			ctx.JSON(err.Status(), err)
		})

		app.Use(func(ctx *Context) error {
			return errors.New("some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("application/json; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal(`{"code":500,"error":"some error"}`, PickRes(res.Text()).(string))
		assert.Equal("", buf.String())
		res.Body.Close()
	})

	t.Run("return router error as JSON", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Set(SetOnError, func(ctx *Context, err HTTPError) {
			ctx.JSON(err.Status(), err)
		})
		router := NewRouter()
		router.Get("/", func(ctx *Context) error {
			return errors.New("some error")
		})
		app.UseHandler(router)
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("application/json; charset=utf-8", res.Header.Get(HeaderContentType))
		assert.Equal(`{"code":500,"error":"some error"}`, PickRes(res.Text()).(string))
		assert.Equal("", buf.String())
		res.Body.Close()
	})

	t.Run("panic recovered", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Use(func(ctx *Context) error {
			ctx.Status(400)
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("Some error", PickRes(res.Text()).(string))

		log := buf.String()
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
		assert := assert.New(t)
		assert.Equal(400, err1.Status())
		assert.Equal("test", err1.Error())

		err := ParseError(err1)
		EqualPtr(t, err1, err)

		err2 := func() error {
			return &Error{Code: 400, Msg: "test"}
		}()
		err = ParseError(err2)
		EqualPtr(t, err2, err)
	})

	t.Run("textproto.Error", func(t *testing.T) {
		assert := assert.New(t)

		err1 := &textproto.Error{Code: 400, Msg: "test"}
		err := ParseError(err1)
		assert.Equal(err.Status(), 400)

		err2 := func() error {
			return &textproto.Error{Code: 400, Msg: "test"}
		}()
		err = ParseError(err2)
		assert.Equal(err.Status(), 400)
	})

	t.Run("custom HTTPError", func(t *testing.T) {
		assert := assert.New(t)

		err1 := &testHTTPError1{c: 400, m: "test"}
		err := ParseError(err1)
		assert.Equal(err.Status(), 400)

		err2 := func() error {
			return &testHTTPError1{c: 400, m: "test"}
		}()
		err = ParseError(err2)
		assert.Equal(err.Status(), 400)

		err3 := &testHTTPError2{c: 400, m: "test"}
		err = ParseError(err3)
		assert.Equal(err.Status(), 400)

		err4 := func() error {
			return &testHTTPError2{c: 400, m: "test"}
		}()
		err = ParseError(err4)
		assert.Equal(err.Status(), 400)
	})

	t.Run("error", func(t *testing.T) {
		assert := assert.New(t)

		err1 := errors.New("test")
		err := ParseError(err1)
		assert.Equal(err.Status(), 500)

		err2 := func() error {
			return errors.New("test")
		}()
		err = ParseError(err2, 0)
		assert.Equal(err.Status(), 500)

		err3 := func() error {
			return errors.New("test")
		}()
		err = ParseError(err3, 400)
		assert.Equal(err.Status(), 400)
	})
}

func TestGearSetTimeout(t *testing.T) {
	t.Run("respond 504 when timeout", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		assert.Panics(func() {
			app.Set(SetTimeout, struct{}{})
		})
		app.Set(SetTimeout, time.Millisecond*100)

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
		assert.Equal(504, res.StatusCode)
		assert.Equal("context deadline exceeded", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("respond 504 when cancel", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set(SetTimeout, time.Millisecond*100)

		app.Use(func(ctx *Context) error {
			ctx.Cancel()
			time.Sleep(time.Millisecond)
			return ctx.End(500, []byte("some data"))
		})
		app.Use(func(ctx *Context) error {
			panic("this middleware unreachable")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(504, res.StatusCode)
		assert.Equal("context canceled", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("respond 200", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set(SetTimeout, time.Millisecond*100)

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

func TestGearSetWithContext(t *testing.T) {
	t.Run("respond 200", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Panics(func() {
			app.Set(SetWithContext, func() {})
		})

		key := struct{}{}
		app.Set(SetWithContext, func(r *http.Request) context.Context {
			return context.WithValue(r.Context(), key, "Hello Context")
		})

		app.Use(func(ctx *Context) error {
			value := ctx.Value(key).(string)
			return ctx.End(200, []byte(value))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("Hello Context", PickRes(res.Text()).(string))
		res.Body.Close()
	})

	t.Run("should panic", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Set(SetWithContext, func(r *http.Request) context.Context {
			return context.WithValue(context.Background(), "key", "Hello Context")
		})
		count := 0
		app.Use(func(ctx *Context) error {
			count++
			return ctx.End(204)
		})

		srv := app.Start()
		defer srv.Close()

		_, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.NotNil(err)
		assert.Equal(0, count)
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
		ctx.Status(400)
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
		ctx.Status(400)
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

func TestGearCompose(t *testing.T) {
	assert := assert.New(t)

	app := New()
	count := 0
	app.Use(Compose(
		func(ctx *Context) error {
			assert.Nil(Compose()(ctx))
			count++
			assert.Equal(1, count)
			return nil
		},
		func(ctx *Context) error {
			count++
			assert.Equal(2, count)
			return ctx.End(400)
		},
		func(ctx *Context) error {
			panic("this middleware unreachable")
		},
	))

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(2, count)
	assert.Equal(400, res.StatusCode)
	res.Body.Close()
}

type WriterTest struct {
	rw http.ResponseWriter
}

func (wt *WriterTest) WriteHeader(code int) {
	wt.rw.WriteHeader(code)
}

func (wt *WriterTest) Header() http.Header {
	return wt.rw.Header()
}

func (wt *WriterTest) Write(b []byte) (int, error) {
	return 0, errors.New("can't write")
}

func TestGearWrapResponseWriter(t *testing.T) {
	assert := assert.New(t)

	app := New()
	var buf bytes.Buffer
	app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
	app.Use(func(ctx *Context) error {
		ctx.Res.rw = &WriterTest{ctx.Res.rw}

		ch := ctx.Res.CloseNotify()
		assert.NotNil(ch)
		return ctx.End(200, []byte("OK"))
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

func TestErrorWithStack(t *testing.T) {
	t.Run("ErrorWithStack", func(t *testing.T) {
		assert := assert.New(t)

		var err error

		assert.Nil(ErrorWithStack(err))

		// *Error type test
		err = &Error{500, "hello", nil, ""}
		assert.NotZero(ErrorWithStack(err).Stack)
		// string type test
		str := "Some thing"
		assert.NotZero(ErrorWithStack(str).Stack)
		// other type
		v := struct {
			a string
		}{
			a: "Some thing",
		}
		assert.NotZero(ErrorWithStack(v).Stack)
		// test skip
		errSkip := &Error{500, "hello", nil, ""}
		assert.True(strings.Index(ErrorWithStack(errSkip, 0).Stack, "app.go") > 0)
	})

	t.Run("Error string", func(t *testing.T) {
		assert := assert.New(t)

		meta := []byte("服务异常")
		err := &Error{500, "Some error", meta, ""}
		assert.True(strings.Contains(err.String(), `, Meta:"服务异常",`))

		meta = meta[0 : len(meta)-1] // invalid utf8 bytes
		err = &Error{500, "Some error", meta, ""}
		assert.False(strings.Contains(err.String(), `, Meta:"服务`))
		assert.True(strings.Contains(err.String(), `, Meta:[]byte{`))
	})

	t.Run("pruneStack", func(t *testing.T) {
		assert := assert.New(t)

		buf := []byte("head line\n")
		for i := 0; i < 100; i++ {
			buf = append(buf, []byte(strconv.Itoa(i)+"\n")...)
		}

		assert.Equal(`1\n3\n5\n7\n9\n11\n13\n15\n17\n19`, pruneStack(buf, 0))
		assert.Equal(`3\n5\n7\n9\n11\n13\n15\n17\n19\n21`, pruneStack(buf, 1))
	})
}

func HTTP2Transport(cert, key string) (*http2.Transport, error) {
	transport := &http2.Transport{}
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}

	if cert != "" && key != "" {
		certificate, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{certificate}
	}

	transport.TLSClientConfig = tlsCfg
	return transport, nil
}
