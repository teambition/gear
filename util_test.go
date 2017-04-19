package gear

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/textproto"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

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

//--------- End ---------

func TestGearError(t *testing.T) {
	t.Run("Predefined errors", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(500, Err.Code)
		assert.Equal("Error", Err.Err)
		assert.Equal("", Err.Msg)
		assert.Equal(500, Err.Status())
		assert.Equal("Error: ", Err.Error())
		assert.Equal(`Error{Code:500, Err:"Error", Msg:"", Data:<nil>, Stack:""}`, Err.String())
	})

	t.Run("Error.WithMsg", func(t *testing.T) {
		assert := assert.New(t)

		err := Err.WithMsg()
		assert.Equal(500, err.Code)
		assert.Equal("Error", err.Err)
		assert.Equal("", err.Msg)
		err.Msg = "Hello"
		assert.Equal("", Err.Msg)

		err = Err.WithMsg("Hello")
		assert.Equal(500, err.Code)
		assert.Equal("Error", err.Err)
		assert.Equal("Hello", err.Msg)
		assert.Equal("Error: Hello", err.Error())

		err = Err.WithMsg("Hello", "world")
		assert.Equal(500, err.Code)
		assert.Equal("Error", err.Err)
		assert.Equal("Hello, world", err.Msg)
		assert.Equal("Error: Hello, world", err.Error())
	})

	t.Run("Error.WithCode", func(t *testing.T) {
		assert := assert.New(t)

		err := Err.WithCode(800)
		assert.Equal(800, err.Code)
		assert.Equal("Error", err.Err)
		assert.Equal("", err.Msg)
		err.Msg = "Hello"
		assert.Equal("", Err.Msg)
		assert.Equal(500, Err.Code)

		err = Err.WithCode(400)
		assert.Equal(400, err.Code)
		assert.Equal("Bad Request", err.Err)
		assert.Equal("", err.Msg)
		err.Msg = "Some error"
		assert.Equal("", Err.Msg)
	})

	t.Run("Error.From", func(t *testing.T) {
		assert := assert.New(t)

		var err error
		assert.Nil(Err.From(err))
		err = errors.New("some error")

		err1 := Err.From(err)
		assert.Equal("some error", err1.Msg)
		assert.Equal("Error: some error", err1.Error())

		err2 := Err.From(err1)
		EqualPtr(t, err1, err2)

		err2 = Err.From(&testHTTPError1{c: 400, m: "testHTTPError1"})
		assert.Equal(400, err2.Status())
		assert.Equal("Error: testHTTPError1", err2.Error())
		NotEqualPtr(t, err1, err2)

		err2 = Err.From(&textproto.Error{Code: 400, Msg: "textproto.Error"})
		assert.Equal(400, err2.Status())
		assert.Equal("Error: textproto.Error", err2.Error())
		NotEqualPtr(t, err1, err2)

		err1 = &Error{}
		err2 = err1.From(&textproto.Error{Code: 400, Msg: "textproto.Error"})
		assert.Equal(400, err2.Status())
		assert.Equal("Bad Request: textproto.Error", err2.Error())
		NotEqualPtr(t, err1, err2)

		err2 = err1.From(err1)
		EqualPtr(t, err1, err2)
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
		assert.Equal(": test", err1.Error())

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
		err = ErrInternalServerError.WithMsg("hello")
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
		errSkip := ErrInternalServerError.WithMsg("hello")
		assert.True(strings.Index(ErrorWithStack(errSkip, 0).Stack, "util.go") > 0)
	})

	t.Run("Error string", func(t *testing.T) {
		assert := assert.New(t)

		data := []byte("服务异常")
		err := ErrInternalServerError.WithMsg("Some error")
		err.Data = data
		assert.True(strings.Contains(err.String(), `, Data:"服务异常",`))

		data = data[0 : len(data)-1] // invalid utf8 bytes
		err.Data = data
		assert.False(strings.Contains(err.String(), `, Data:"服务`))
		assert.True(strings.Contains(err.String(), `, Data:[]byte{`))
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

func TestGearCheckStatus(t *testing.T) {
	assert := assert.New(t)
	assert.False(IsStatusCode(1))
	assert.True(IsStatusCode(100))

	assert.False(isRedirectStatus(200))
	assert.True(isRedirectStatus(301))

	assert.False(isEmptyStatus(200))
	assert.True(isEmptyStatus(204))
}

func TestGearContentDisposition(t *testing.T) {
	t.Run("Should format", func(t *testing.T) {
		assert := assert.New(t)

		assert.Equal("attachment", ContentDisposition("", ""))
		assert.Equal("inline", ContentDisposition("", "inline"))
		assert.Equal(`inline; filename="abc.txt"`, ContentDisposition(`abc.txt`, "inline"))
		assert.Equal(`attachment; filename="\"abc\".txt"; filename*=UTF-8''%22abc%22.txt`,
			ContentDisposition(`"abc".txt`, ""))
		assert.Equal(`attachment; filename="统计数据.txt"; filename*=UTF-8''%E7%BB%9F%E8%AE%A1%E6%95%B0%E6%8D%AE.txt`,
			ContentDisposition(`统计数据.txt`, ""))
		assert.Equal(`inline; filename="€ rates.txt"; filename*=UTF-8''%E2%82%AC%20rates.txt`,
			ContentDisposition(`€ rates.txt`, "inline"))

		mType, params, _ := mime.ParseMediaType(ContentDisposition(`统计数据.txt`, ""))
		assert.Equal("attachment", mType)
		assert.Equal("统计数据.txt", params["filename"])

		mType, params, _ = mime.ParseMediaType(ContentDisposition(`€ rates.txt`, "inline"))
		assert.Equal("inline", mType)
		assert.Equal("€ rates.txt", params["filename"])

		mType, params, _ = mime.ParseMediaType(ContentDisposition(`"abc".txt`, ""))
		assert.Equal("attachment", mType)
		assert.Equal(`"abc".txt`, params["filename"])
	})
}

type formStruct struct {
	String  string   `form:"string"`
	Bool    bool     `form:"bool"`
	Int     int      `form:"int"`
	Int8    int8     `form:"int8"`
	Int16   int16    `form:"int16"`
	Int32   int32    `form:"int32"`
	Int64   int64    `form:"int64"`
	Uint    uint     `form:"uint"`
	Uint8   uint8    `form:"uint8"`
	Uint16  uint16   `form:"uint16"`
	Uint32  uint32   `form:"uint32"`
	Uint64  uint64   `form:"uint64"`
	Float32 float32  `form:"float32"`
	Float64 float64  `form:"float64"`
	Slice1  []string `form:"slice1"`
	Slice2  []int    `form:"slice2"`
	Slice3  []int    `form:"slice3"`
	Hide    string   `json:"hide"`
}

func TestGearFormToStruct(t *testing.T) {
	data := url.Values{
		"string":  {"string"},
		"bool":    {"true"},
		"int":     {"-1"},
		"int8":    {"-1"},
		"int16":   {"-1"},
		"int32":   {"-1"},
		"int64":   {"-1"},
		"uint":    {"1"},
		"uint8":   {"1"},
		"uint16":  {"1"},
		"uint32":  {"1"},
		"uint64":  {"1"},
		"float32": {"1.1"},
		"float64": {"1.1"},
		"slice1":  {"slice1"},
		"slice2":  {"1"},
		"slice3":  {},
	}

	t.Run("Should error", func(t *testing.T) {
		assert := assert.New(t)

		assert.NotNil(FormToStruct(nil, nil))
		assert.NotNil(FormToStruct(data, nil))

		var v1 formStruct
		var v2 *formStruct
		assert.NotNil(FormToStruct(data, v1))
		assert.NotNil(FormToStruct(data, v2))

		v1 = formStruct{}
		assert.NotNil(FormToStruct(data, v1))

		v3 := struct {
			String interface{} `form:"string"`
		}{}
		assert.NotNil(FormToStruct(data, &v3))

		v4 := struct {
			Slice []int `form:"slice"`
		}{}
		assert.NotNil(FormToStruct(url.Values{"slice": {"a"}}, &v4))
	})

	t.Run("Should work", func(t *testing.T) {
		assert := assert.New(t)

		s := formStruct{}
		assert.Nil(FormToStruct(data, &s))
		assert.Equal("string", s.String)
		assert.Equal(true, s.Bool)
		assert.Equal(int(-1), s.Int)
		assert.Equal(int8(-1), s.Int8)
		assert.Equal(int16(-1), s.Int16)
		assert.Equal(int32(-1), s.Int32)
		assert.Equal(int64(-1), s.Int64)
		assert.Equal(uint(1), s.Uint)
		assert.Equal(uint8(1), s.Uint8)
		assert.Equal(uint16(1), s.Uint16)
		assert.Equal(uint32(1), s.Uint32)
		assert.Equal(uint64(1), s.Uint64)
		assert.Equal(float32(1.1), s.Float32)
		assert.Equal(float64(1.1), s.Float64)
		assert.Equal([]string{"slice1"}, s.Slice1)
		assert.Equal([]int{1}, s.Slice2)
		assert.Equal([]int{}, s.Slice3)
	})
}
