package gear

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/go-http-utils/cookie"
	"github.com/stretchr/testify/assert"
)

type Counter int32

func (c *Counter) Int() int {
	return int(atomic.LoadInt32((*int32)(c)))
}

func (c *Counter) Add() int {
	return int(atomic.AddInt32((*int32)(c), 1))
}

func CtxTest(app *App, method, url string, body io.Reader) *Context {
	req := httptest.NewRequest(method, url, body)
	res := httptest.NewRecorder()
	return NewContext(app, res, req)
}

func CtxResult(ctx *Context) *http.Response {
	res := ctx.Res.rw.(*httptest.ResponseRecorder)
	return res.Result()
}

func CtxBody(ctx *Context) (val string) {
	body, err := ioutil.ReadAll(CtxResult(ctx).Body)
	if err == nil {
		val = bytes.NewBuffer(body).String()
	}
	return
}

func TestGearContextContextInterface(t *testing.T) {
	assert := assert.New(t)

	app := New()
	ch := make(chan bool, 1)
	app.Use(func(ctx *Context) error {
		// ctx.Deadline
		_, ok := ctx.Deadline()
		assert.False(ok)
		// ctx.Err
		assert.Nil(ctx.Err())
		// ctx.Value
		s := ctx.Value(http.ServerContextKey)
		EqualPtr(t, s, app.Server)

		go func() {
			// ctx.Done
			<-ctx.Done()
			ch <- true
		}()

		return ctx.End(204)
	})
	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)
	assert.True(<-ch)
}

func TestGearContextWithContext(t *testing.T) {
	assert := assert.New(t)

	type contextKey string

	count := Counter(0)
	app := New()
	app.Use(func(ctx *Context) error {
		assert.Panics(func() {
			ctx.WithContext(context.WithValue(ctx, contextKey("key"), "val"))
		})

		assert.Panics(func() {
			ctx.WithContext(context.WithValue(context.Background(), contextKey("key"), "val"))
		})

		ctx.WithContext(context.WithValue(ctx.Context(), contextKey("key"), "val"))
		ctx.WithContext(ctx.WithValue(contextKey("key2"), "val2"))
		assert.Equal("val", ctx.Value(contextKey("key")))
		assert.Equal("val2", ctx.Value(contextKey("key2")))
		assert.Equal("val", ctx.Req.Context().Value(contextKey("key")))
		assert.Equal("val2", ctx.Req.Context().Value(contextKey("key2")))

		c := ctx.WithValue(contextKey("test"), "abc")
		assert.Equal("val", c.Value(contextKey("key")))
		assert.Equal("val2", c.Value(contextKey("key2")))
		assert.Equal("abc", c.Value(contextKey("test")))
		s := c.Value(http.ServerContextKey)
		EqualPtr(t, s, app.Server)

		c1, _ := ctx.WithCancel()
		c2, _ := ctx.WithDeadline(time.Now().Add(time.Second))
		c3, _ := ctx.WithTimeout(time.Second)

		go func() {
			<-c1.Done()
			assert.True(ctx.Res.ended.isTrue())
			count.Add()
		}()

		go func() {
			<-c2.Done()
			assert.True(ctx.Res.ended.isTrue())
			count.Add()
		}()

		go func() {
			<-c3.Done()
			assert.True(ctx.Res.ended.isTrue())
			count.Add()
		}()

		ctx.Status(404)
		ctx.Cancel()

		assert.True(ctx.Res.ended.isTrue())
		time.Sleep(time.Millisecond)
		return nil
	})
	app.Use(func(ctx *Context) error {
		panic("this middleware unreachable")
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(499, res.StatusCode)
	assert.Equal(3, count.Int())
}

func TestGearContextLogErr(t *testing.T) {
	t.Run("normal error and no flag", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "", 0))
		app.Use(func(ctx *Context) error {
			ctx.LogErr(errors.New("some error"))
			return ctx.End(204)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)

		// [2017-08-11T15:41:22.709Z] ERR {"code":500,"error":"InternalServerError","message":"some error","stack":"\\t/usr/local/go/src/testing/testing.go:697"}
		assert.True(strings.Contains(buf.String(), `Z] ERR {"`))
		assert.True(strings.Contains(buf.String(), `"message":"some error"`))
		assert.True(strings.Contains(buf.String(), `"stack":"\\t`))
	})
}

func TestGearContextTiming(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		x := false
		app.Use(func(ctx *Context) error {
			err := ctx.Timing(time.Millisecond*150, func(c context.Context) {
				go func() {
					<-c.Done()
					assert.Equal(context.Canceled, c.Err())
				}()
				time.Sleep(time.Millisecond * 100)
				x = true
			})
			assert.Nil(err)
			assert.True(x)
			return ctx.End(204)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
	})

	t.Run("when fn panic", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			err := ctx.Timing(time.Millisecond*15, func(c context.Context) {
				go func() {
					<-c.Done()
					assert.Equal(context.Canceled, c.Err())
				}()
				panic("some timing error")
			})
			assert.NotNil(err)
			return err
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"InternalServerError","message":"Timing panic: \"some timing error\""}`, PickRes(res.Text()).(string))
	})

	t.Run("when timeout", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			err := ctx.Timing(time.Millisecond*10, func(c context.Context) {
				go func() {
					<-c.Done()
					assert.Equal(context.DeadlineExceeded, c.Err())
				}()
				time.Sleep(time.Millisecond * 15)
			})
			assert.Equal(context.DeadlineExceeded, err)
			return ctx.Error(err)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"InternalServerError","message":"context deadline exceeded"}`, PickRes(res.Text()).(string))
	})

	t.Run("when context timeout", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set(SetTimeout, time.Millisecond*10)
		app.Use(func(ctx *Context) error {
			err := ctx.Timing(time.Millisecond*20, func(c context.Context) {
				go func() {
					<-c.Done()
					assert.Equal(context.DeadlineExceeded, c.Err())
				}()
				time.Sleep(time.Millisecond * 15)
			})
			assert.Equal(context.DeadlineExceeded, err)
			time.Sleep(time.Millisecond * 10)
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(504, res.StatusCode)
		assert.Equal(`{"error":"GatewayTimeout","message":"context deadline exceeded"}`, PickRes(res.Text()).(string))
		time.Sleep(time.Second)
	})
}

// ----- Test Context.Any -----
type ctxAnyType struct{}
type ctxAnyResult struct {
	Host string
	Path string
}

var ctxAny = &ctxAnyType{}

func (t *ctxAnyType) New(ctx *Context) (interface{}, error) {
	if ctx.Method != "GET" {
		return nil, errors.New(ctx.Method)
	}
	return &ctxAnyResult{Host: ctx.Host, Path: ctx.Path}, nil
}

func TestGearContextAny(t *testing.T) {
	app := New()
	assert.Panics(t, func() {
		app.Set(SetEnv, 123)
	})

	t.Run("type Any", func(t *testing.T) {
		t.Run("should get the same value with the same ctx", func(t *testing.T) {
			assert := assert.New(t)

			ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
			val, err := ctx.Any(ctxAny)
			assert.Nil(err)
			res := val.(*ctxAnyResult)
			assert.Equal(ctx.Host, res.Host)
			assert.Equal(ctx.Path, res.Path)

			val2, _ := ctx.Any(ctxAny)
			EqualPtr(t, val, val2)

			val3 := ctx.MustAny(ctxAny)
			EqualPtr(t, val, val3)
		})

		t.Run("should get different value with different ctx", func(t *testing.T) {
			assert := assert.New(t)

			ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
			val, err := ctx.Any(ctxAny)
			assert.Nil(err)

			ctx2 := CtxTest(app, "GET", "http://example.com/foo", nil)
			val2, err2 := ctx2.Any(ctxAny)
			assert.Nil(err2)
			NotEqualPtr(t, val, val2)
		})

		t.Run("should get error", func(t *testing.T) {
			assert := assert.New(t)

			ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
			val, err := ctx.Any(ctxAny)
			assert.Nil(val)
			assert.NotNil(err)
			assert.Equal("POST", err.Error())
		})
	})

	t.Run("SetAny with interface{}", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		val, err := ctx.Any(struct{}{})
		assert.Nil(val)
		assert.Equal("Error: non-existent key", err.Error())

		ctx.SetAny(struct{}{}, true)
		val, err = ctx.Any(struct{}{})
		assert.Nil(err)
		assert.True(val.(bool))
	})

	t.Run("Setting", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		assert.Equal("development", ctx.Setting(SetEnv).(string))

		app.Set(SetEnv, "test")
		ctx = CtxTest(app, "POST", "http://example.com/foo", nil)
		assert.Equal("test", ctx.Setting(SetEnv).(string))
	})
}

func TestGearContextSetting(t *testing.T) {
	assert := assert.New(t)
	val := map[string]int{"abc": 123}

	app := New()
	app.Set("someKey", val)
	ctx := CtxTest(app, "POST", "http://example.com/foo", nil)

	assert.Nil(ctx.Setting("key"))
	assert.Equal(val, ctx.Setting("someKey").(map[string]int))
}

func TestGearContextIP(t *testing.T) {
	t.Run("Default Setting", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		ctx.Req.Header.Set("X-Forwarded-For", "188.188.188.188, 192.168.0.99")
		assert.Equal("127.0.0.1", ctx.IP().String())

		ctx = CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		ctx.Req.Header.Set("X-Real-IP", "192.168.0.99")
		assert.Equal("127.0.0.1", ctx.IP().String())
	})

	t.Run("when set true", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Set(SetTrustedProxy, true)
		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		assert.Equal("127.0.0.1", ctx.IP().String())

		ctx.Req.Header.Set("X-Forwarded-For", "188.188.188.189")
		assert.Equal("188.188.188.189", ctx.IP().String())

		ctx.Req.Header.Set("X-Forwarded-For", "188.188.188.188, 192.168.0.99")
		assert.Equal("188.188.188.188", ctx.IP().String())

		ctx.Req.Header.Set("X-Real-IP", "188.188.188.187")
		assert.Equal("188.188.188.187", ctx.IP().String())
	})
}

func TestGearContextScheme(t *testing.T) {
	t.Run("Default Setting", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		ctx.Req.Header.Set("X-Forwarded-Proto", "https")
		assert.Equal("http", ctx.Scheme())

		ctx = CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		ctx.Req.Header.Set("X-Real-Scheme", "https")
		assert.Equal("http", ctx.Scheme())
	})

	t.Run("when set true", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Set(SetTrustedProxy, true)
		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		ctx.Req.Header.Set("X-Forwarded-Proto", "https")
		assert.Equal("https", ctx.Scheme())

		ctx = CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		ctx.Req.Header.Set("X-Forwarded-Scheme", "https")
		assert.Equal("https", ctx.Scheme())

		ctx = CtxTest(app, "POST", "http://example.com/foo", nil)
		ctx.Req.RemoteAddr = "127.0.0.1:65432"
		ctx.Req.Header.Set("X-Forwarded-Proto", "https")
		ctx.Req.Header.Set("X-Forwarded-Scheme", "https")
		ctx.Req.Header.Set("X-Real-Scheme", "http")
		assert.Equal("http", ctx.Scheme())
	})
}

func TestGearContextAccept(t *testing.T) {
	t.Run("ctx.AcceptType", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Req.Header.Set(HeaderAccept, "application/*;q=0.2, image/jpeg;q=0.8, text/html, text/plain")
		assert.Equal("text/html", ctx.AcceptType())
		assert.Equal("text/plain", ctx.AcceptType("text/plain", "application/json"))
		assert.Equal("", ctx.AcceptType("image/png", "image/tiff"))
	})

	t.Run("ctx.AcceptLanguage", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Req.Header.Set(HeaderAcceptLanguage, "en;q=0.8, es, pt")
		assert.Equal("es", ctx.AcceptLanguage())
		assert.Equal("pt", ctx.AcceptLanguage("en", "pt"))
	})

	t.Run("ctx.AcceptEncoding", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Req.Header.Set(HeaderAcceptEncoding, "gzip, compress;q=0.2")
		assert.Equal("gzip", ctx.AcceptEncoding())
		assert.Equal("compress", ctx.AcceptEncoding("deflate", "compress"))
	})

	t.Run("ctx.AcceptCharset", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.Req.Header.Set(HeaderAcceptCharset, "utf-8, iso-8859-1;q=0.2, utf-7;q=0.5")
		assert.Equal("utf-8", ctx.AcceptCharset())
		assert.Equal("utf-8", ctx.AcceptCharset("iso-8859-1", "utf-8"))
	})
}

func TestGearContextParam(t *testing.T) {
	assert := assert.New(t)

	app := New()
	r := NewRouter()
	r.Get("/api/:type/:id", func(ctx *Context) error {
		assert.Equal("user", ctx.Param("type"))
		assert.Equal("123", ctx.Param("id"))
		assert.Equal("", ctx.Param("other"))
		return ctx.End(http.StatusNoContent)
	})
	r.Get("/view/:all*", func(ctx *Context) error {
		assert.Equal("user/123", ctx.Param("all"))
		return ctx.End(http.StatusNoContent)
	})
	app.UseHandler(r)

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	res, err := RequestBy("GET", host+"/api/user/123")
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)

	res, err = RequestBy("GET", host+"/view/user/123")

	assert.Nil(err)
	assert.Equal(204, res.StatusCode)
}

func TestGearContextQuery(t *testing.T) {
	assert := assert.New(t)

	app := New()
	r := NewRouter()
	r.Get("/api", func(ctx *Context) error {
		assert.Equal("user", ctx.Query("type"))
		assert.Equal("123", ctx.Query("id"))
		assert.Equal([]string{"123"}, ctx.QueryAll("id"))
		assert.Equal("", ctx.Query("other"))
		return ctx.End(http.StatusNoContent)
	})
	r.Get("/view", func(ctx *Context) error {
		assert.Nil(ctx.QueryAll("other"))
		assert.Equal("123", ctx.Query("id"))
		assert.Equal([]string{"123", "abc"}, ctx.QueryAll("id"))
		return ctx.End(http.StatusNoContent)
	})
	app.UseHandler(r)

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	res, err := RequestBy("GET", host+"/api?type=user&id=123")
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)

	res, err = RequestBy("GET", host+"/view?id=123&id=abc")
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)
}

func TestGearContextCookies(t *testing.T) {
	t.Run("without keys", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			val, err := ctx.Cookies.Get("Gear")
			assert.Nil(err)
			assert.Equal("test", val)

			ctx.Cookies.Set("Gear", "Hello")
			return ctx.End(http.StatusNoContent)
		})

		srv := app.Start()
		defer srv.Close()

		host := "http://" + srv.Addr().String()
		req, _ := NewRequst("GET", host)
		res, err := DefaultClientDoWithCookies(req, map[string]string{"Gear": "test"})
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		c := res.Cookies()[0]
		assert.Equal("Gear", c.Name)
		assert.Equal("Hello", c.Value)
	})

	t.Run("with keys", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Panics(func() {
			app.Set(SetKeys, "some key")
		})
		app.Set(SetKeys, []string{"some key"})
		app.Use(func(ctx *Context) error {
			val, err := ctx.Cookies.Get("cookieKey", true)
			assert.Nil(err)
			assert.Equal("cookie value", val)

			ctx.Cookies.Set("Gear", "Hello", &cookie.Options{Signed: true})
			return ctx.End(http.StatusNoContent)
		})

		srv := app.Start()
		defer srv.Close()

		host := "http://" + srv.Addr().String()
		req, _ := NewRequst("GET", host)
		res, err := DefaultClientDoWithCookies(req, map[string]string{
			"cookieKey":     "cookie value",
			"cookieKey.sig": "JROAKAAIUzC3_akvMb7PKF4l5h4",
		})
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		c := res.Cookies()[0]
		assert.Equal("Gear", c.Name)
		assert.Equal("Hello", c.Value)
		sig := res.Cookies()[1]
		assert.Equal("Gear.sig", sig.Name)
	})
}

type jsonBodyTemplate struct {
	ID   string `json:"id" form:"id"`
	Pass string `json:"pass" form:"pass"`
}

func (b *jsonBodyTemplate) Validate() error {
	if len(b.ID) < 3 || len(b.Pass) < 6 {
		return ErrBadRequest.WithMsg("invalid id or pass")
	}
	return nil
}

type xmlBodyTemplate struct {
	ID   string `xml:"id,attr"`
	Pass string `xml:"pass,attr"`
}

func (b *xmlBodyTemplate) Validate() error {
	if len(b.ID) < 3 || len(b.Pass) < 6 {
		return ErrBadRequest.WithMsg("invalid id or pass")
	}
	return nil
}

type mapTemplate map[string]*bson.ObjectId

func (m mapTemplate) Validate() error {
	if !m["id"].Valid() {
		return ErrBadRequest.WithMsg("invalid id")
	}
	return nil
}

func TestGearContextParseBody(t *testing.T) {
	app := New()
	assert.Panics(t, func() {
		app.Set(SetBodyParser, 123)
	})

	t.Run("should parse JSON content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"admin","pass":"password"}`)))
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)

		body := jsonBodyTemplate{}
		assert.Nil(ctx.ParseBody(&body))
		assert.Equal("admin", body.ID)
		assert.Equal("password", body.Pass)
	})

	t.Run("should parse JSON type like content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"subject","pass":"password"}`)))
		ctx.Req.Header.Set(HeaderContentType, "application/jrd+json")

		body := jsonBodyTemplate{}
		assert.Nil(ctx.ParseBody(&body))
		assert.Equal("subject", body.ID)
	})

	t.Run("should parse JSON fail if invalid field type", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"subject","pass":12345678}`)))
		ctx.Req.Header.Set(HeaderContentType, "application/json")

		body := jsonBodyTemplate{}

		err := ctx.ParseBody(&body)
		assert.Equal(400, err.(*Error).Code)
		// "BadRequest: Unmarshal type error: field=pass, expected=string, got=number, offset=31"
		// go1.11: "BadRequest: Unmarshal type error: expected=string, got=number, offset=31"
		assert.Contains(err.Error(), "expected=string, got=number, offset=31")
	})

	t.Run("should parse JSON fail if invalid json", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{abc}`)))
		ctx.Req.Header.Set(HeaderContentType, "application/json")

		body := jsonBodyTemplate{}

		err := ctx.ParseBody(&body)
		assert.Equal(400, err.(*Error).Code)
		assert.Equal("BadRequest: Syntax error: offset=2, error=invalid character 'a' looking for beginning of object key string", err.Error())
	})

	t.Run("should parse XML content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`<body id="admin" pass="password"></body>`)))
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationXML)

		body := xmlBodyTemplate{}
		assert.Nil(ctx.ParseBody(&body))
		assert.Equal("admin", body.ID)
		assert.Equal("password", body.Pass)
	})

	t.Run("should support mapTemplate", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"000000000000000000000000"}`)))
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)

		body := mapTemplate{}
		assert.Nil(ctx.ParseBody(&body))
		assert.Equal(*body["id"], bson.ObjectIdHex("000000000000000000000000"))
	})

	t.Run("should 400 error when validate error", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"admin","pass":"pass"}`)))
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)

		body := jsonBodyTemplate{}
		err := ctx.ParseBody(&body)
		assert.Equal(400, err.(*Error).Code)
	})

	t.Run("should 415 error with invalid content type", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"admin","pass":"password"}`)))
		ctx.Req.Header.Set(HeaderContentType, "invalid type")

		body := jsonBodyTemplate{}
		err := ctx.ParseBody(&body)
		assert.Equal(415, err.(*Error).Code)
	})

	t.Run("should 415 error with empty content type", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"admin","pass":"password"}`)))

		body := jsonBodyTemplate{}
		err := ctx.ParseBody(&body)
		assert.Equal(415, err.(*Error).Code)
	})

	t.Run("should 400 error with empty content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		body := jsonBodyTemplate{}
		err := ctx.ParseBody(&body)
		assert.Equal(400, err.(*Error).Code)
	})

	t.Run("should 413 error when content too large", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Set(SetBodyParser, DefaultBodyParser(100))

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBufferString(strings.Repeat("t", 101)))
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)
		body := jsonBodyTemplate{}
		err := ctx.ParseBody(&body)
		assert.Equal(413, err.(*Error).Code)
	})

	t.Run("should error when bodyParser not exists", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.bodyParser = nil

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"admin","pass":"pass"}`)))
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)
		body := jsonBodyTemplate{}
		err := ctx.ParseBody(&body)
		assert.Equal("Error: bodyParser not registered", err.Error())
	})

	t.Run("should error when req.Body not exists", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		ctx := CtxTest(app, "POST", "http://example.com/foo",
			bytes.NewBuffer([]byte(`{"id":"admin","pass":"pass"}`)))
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)
		ctx.Req.Body = nil
		body := jsonBodyTemplate{}
		err := ctx.ParseBody(&body)
		assert.Equal("Error: missing request body", err.Error())
	})

	t.Run("should support content with gzip encoding", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		pass := strings.Repeat("你好，Gear", 500)
		data := []byte(fmt.Sprintf(`{"id":"admin","pass":"%s"}`, pass))

		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		gw.Write(data)
		gw.Close()
		assert.True(len(buf.Bytes()) < len(data))

		ctx := CtxTest(app, "POST", "http://example.com/foo", &buf)
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)
		ctx.Req.Header.Set(HeaderContentEncoding, "gzip")

		body := jsonBodyTemplate{}
		assert.Nil(ctx.ParseBody(&body))
		assert.Equal("admin", body.ID)
		assert.Equal(pass, body.Pass)
	})

	t.Run("should support content with deflate encoding", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		pass := strings.Repeat("你好，Gear", 500)
		data := []byte(fmt.Sprintf(`{"id":"admin","pass":"%s"}`, pass))

		var buf bytes.Buffer
		gw := zlib.NewWriter(&buf)
		gw.Write(data)
		gw.Close()
		assert.True(len(buf.Bytes()) < len(data))

		ctx := CtxTest(app, "POST", "http://example.com/foo", &buf)
		ctx.Req.Header.Set(HeaderContentType, MIMEApplicationJSON)
		ctx.Req.Header.Set(HeaderContentEncoding, "deflate")

		body := jsonBodyTemplate{}
		assert.Nil(ctx.ParseBody(&body))
		assert.Equal("admin", body.ID)
		assert.Equal(pass, body.Pass)
	})
}

type PaginationEmbedTemplate struct {
	PageSize  int    `json:"page_size" query:"page_size"`
	PageToken string `json:"page_token" query:"page_token"`
}

type jsonQueryTemplate struct {
	ID   string `json:"id" query:"id"`
	Pass string `json:"pass" query:"pass"`
	PaginationEmbedTemplate
}

func (b *jsonQueryTemplate) Validate() error {
	if len(b.ID) < 3 || len(b.Pass) < 6 {
		return ErrBadRequest.WithMsg("invalid id or pass")
	}
	return nil
}

type jsonParamTemplate struct {
	ID   string `json:"id" param:"id"`
	Pass string `json:"pass" param:"pass"`
}

func (b *jsonParamTemplate) Validate() error {
	if len(b.ID) < 3 || len(b.Pass) < 6 {
		return ErrBadRequest.WithMsg("invalid id or pass")
	}
	return nil
}

type jsonParamQueryTemplate struct {
	ID   string `json:"id" query:"id" param:"id"`
	Pass string `json:"pass" query:"pass" param:"pass"`
	Size int    `json:"size" query:"size" param:"size"`
}

func (b *jsonParamQueryTemplate) Validate() error {
	if len(b.ID) < 3 || len(b.Pass) < 6 {
		return ErrBadRequest.WithMsg("invalid id or pass")
	}
	return nil
}

type invalidQueryTemplate struct {
	ID   string    `json:"id" query:"id"`
	Pass string    `json:"pass" query:"pass"`
	Time time.Time `json:"time" query:"time"`
}

func (b *invalidQueryTemplate) Validate() error {
	return nil
}

type invalidParamTemplate struct {
	ID   string    `json:"id" param:"id"`
	Pass string    `json:"pass" param:"pass"`
	Time time.Time `json:"time" param:"time"`
}

func (b *invalidParamTemplate) Validate() error {
	return nil
}

type jsonPointerQueryTemplate struct {
	ID   *string `json:"id" query:"id"`
	Pass *string `json:"pass" query:"pass"`
}

func (b *jsonPointerQueryTemplate) Validate() error {
	return nil
}

func TestGearContextParseURL(t *testing.T) {
	app := New()
	assert.Panics(t, func() {
		app.Set(SetURLParser, 123)
	})

	t.Run("should error when urlParser not exists", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.urlParser = nil

		ctx := CtxTest(app, "GET", "http://example.com/foo?pass=123456789&id=foobar", nil)
		body := jsonQueryTemplate{}
		err := ctx.ParseURL(&body)
		assert.Equal("Error: urlParser not registered", err.Error())
	})

	t.Run("should 400 error when validate error", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo?pass=12&id=admin", nil)

		body := jsonQueryTemplate{}
		err := ctx.ParseURL(&body)
		assert.NotNil(err)
		assert.Equal(400, err.(*Error).Code)
	})

	t.Run("should parse query content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo?pass=password&id=admin&page_token=xxx&page_size=2", nil)

		body := jsonQueryTemplate{}
		err := ctx.ParseURL(&body)
		assert.Nil(err)
		assert.Equal("admin", body.ID)
		assert.Equal("password", body.Pass)
		assert.Equal(2, body.PageSize)
		assert.Equal("xxx", body.PageToken)
	})

	t.Run("should parse query error with invalid data type", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo?pass=password&id=admin&name=admin&time=vdfvdf", nil)
		body := invalidQueryTemplate{}
		err := ctx.ParseURL(&body)
		assert.NotNil(err)
		assert.Equal(400, err.(*Error).Code)
	})

	t.Run("should parse param error with invalid data type", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo?pass=password&id=admin&name=admin", nil)
		ctx.SetAny(paramsKey, map[string]string{
			"pass": "1234567",
			"id":   "admin_id",
			"time": "vdfvdf",
		})
		body := invalidParamTemplate{}
		err := ctx.ParseURL(&body)
		assert.NotNil(err)
		assert.Equal(400, err.(*Error).Code)
	})

	t.Run("should parse url params", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		ctx.SetAny(paramsKey, map[string]string{
			"pass": "1234567",
			"id":   "admin_id",
		})
		body := jsonParamTemplate{}
		err := ctx.ParseURL(&body)
		assert.Nil(err)
		assert.Equal("admin_id", body.ID)
		assert.Equal("1234567", body.Pass)
	})

	t.Run("should parse url params and query content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo?id=admin&size=", nil)
		ctx.SetAny(paramsKey, map[string]string{
			"pass": "1234567",
		})
		body := jsonParamQueryTemplate{}
		err := ctx.ParseURL(&body)
		assert.Nil(err)
		assert.Equal("admin", body.ID)
		assert.Equal("1234567", body.Pass)
		assert.Equal(0, body.Size)
	})

	t.Run("should parse url params take precedence over query content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo?pass=password&id=admin", nil)
		ctx.SetAny(paramsKey, map[string]string{
			"pass": "1234567",
			"id":   "admin_id",
		})
		body := jsonParamQueryTemplate{}
		err := ctx.ParseURL(&body)
		assert.Nil(err)
		assert.Equal("admin_id", body.ID)
		assert.Equal("1234567", body.Pass)
	})

	t.Run("shoud parse pointer query content", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo?id=admin&pass=password", nil)
		body := jsonPointerQueryTemplate{}
		err := ctx.ParseURL(&body)
		assert.Nil(err)
		assert.Equal("admin", *body.ID)
		assert.Equal("password", *body.Pass)
	})
}

func TestGearContextGetSet(t *testing.T) {
	app := New()

	t.Run("shoud GetHeader and SetHeader", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		assert.Equal("", ctx.GetHeader(HeaderAccept))
		ctx.SetHeader(HeaderWarning, "Some error")
		res := CtxResult(ctx)
		assert.Equal("Some error", res.Header.Get(HeaderWarning))
	})

	t.Run("shoud support Referer header", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		assert.Equal("", ctx.GetHeader(HeaderReferer))
		ctx.Req.Header.Set(HeaderReferer, "http://test.com")
		assert.Equal("http://test.com", ctx.GetHeader(HeaderReferer))
		assert.Equal("http://test.com", ctx.GetHeader("referer"))
		assert.Equal("http://test.com", ctx.GetHeader("Referrer"))
		assert.Equal("http://test.com", ctx.GetHeader("referrer"))
	})

	t.Run("shoud support Referrer header", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		assert.Equal("", ctx.GetHeader(HeaderReferer))
		ctx.Req.Header.Set("Referrer", "http://test.com")
		assert.Equal("http://test.com", ctx.GetHeader(HeaderReferer))
		assert.Equal("http://test.com", ctx.GetHeader("referer"))
		assert.Equal("http://test.com", ctx.GetHeader("Referrer"))
		assert.Equal("http://test.com", ctx.GetHeader("referrer"))
	})

	t.Run("GetHeaders should work", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
		assert.Equal(0, len(ctx.GetHeaders(HeaderXCanary)))
		ctx.Req.Header.Set(HeaderXCanary, "label=stable")
		ctx.Req.Header.Add(HeaderXCanary, "product=gear")
		assert.Equal([]string{"label=stable", "product=gear"}, ctx.GetHeaders(HeaderXCanary))
	})
}

func TestGearContextStatus(t *testing.T) {
	assert := assert.New(t)

	app := New()
	ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
	assert.Equal(ctx.Res.Status(), 0)
	ctx.Status(401)
	assert.Equal(ctx.Res.Status(), 401)
}

func TestGearContextType(t *testing.T) {
	assert := assert.New(t)

	app := New()
	ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
	assert.Equal("", ctx.Res.Type())
	ctx.Type(MIMEApplicationJSONCharsetUTF8)
	assert.Equal(MIMEApplicationJSONCharsetUTF8, ctx.Res.Get(HeaderContentType))
	assert.Equal(MIMEApplicationJSONCharsetUTF8, ctx.Res.Type())
}

func TestGearContextHTML(t *testing.T) {
	assert := assert.New(t)

	count := Counter(0)
	app := New()
	app.Use(func(ctx *Context) error {
		ctx.OnEnd(func() {
			assert.Equal(2, count.Add())
		})
		ctx.After(func() {
			assert.Equal(1, count.Add())
		})
		return ctx.HTML(http.StatusOK, "Hello")
	})
	app.Use(func(ctx *Context) error {
		panic("this middleware unreachable")
	})

	srv := app.Start()
	defer srv.Close()

	res, err := RequestBy("GET", "http://"+srv.Addr().String())
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(200, res.StatusCode)
	assert.Equal("Hello", PickRes(res.Text()).(string))
	assert.Equal(2, count.Int())
}

func TestGearContextJSON(t *testing.T) {
	assert := assert.New(t)

	count := Counter(0)
	app := New()
	app.Use(func(ctx *Context) error {
		if ctx.Path == "/error" {
			ctx.OnEnd(func() {
				assert.Equal(3, count.Add())
			})
			ctx.After(func() {
				panic("this hook unreachable")
			})
			return ctx.JSON(http.StatusOK, math.NaN())
		}

		ctx.OnEnd(func() {
			assert.Equal(2, count.Add())
		})
		ctx.After(func() {
			assert.Equal(1, count.Add())
		})
		return ctx.JSON(http.StatusOK, []string{"Hello"})
	})
	app.Use(func(ctx *Context) error {
		panic("this middleware unreachable")
	})

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	res, err := RequestBy("GET", host)
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(200, res.StatusCode)
	assert.Equal(`["Hello"]`, PickRes(res.Text()).(string))
	assert.Equal(2, count.Int())
	assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))

	res, err = RequestBy("GET", host+"/error")
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(500, res.StatusCode)
	assert.True(strings.Contains(PickRes(res.Text()).(string), "json: unsupported value"))
	assert.Equal(3, count.Int())
	assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))
}

func TestGearContextOkJSON(t *testing.T) {
	assert := assert.New(t)

	app := New()
	app.Use(func(ctx *Context) error {
		return ctx.OkJSON(struct{}{})
	})

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	res, err := RequestBy("GET", host)
	assert.Nil(err)

	assert.Equal(200, res.StatusCode)
	assert.Equal(`{}`, PickRes(res.Text()).(string))
	assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))
}

func TestGearContextJSONP(t *testing.T) {
	assert := assert.New(t)

	count := Counter(0)
	app := New()
	app.Use(func(ctx *Context) error {
		if ctx.Path == "/error" {
			ctx.OnEnd(func() {
				assert.Equal(3, count.Add())
			})
			ctx.After(func() {
				panic("this hook unreachable")
			})
			return ctx.JSONP(http.StatusOK, "cb123", math.NaN())
		}

		ctx.OnEnd(func() {
			assert.Equal(2, count.Add())
		})
		ctx.After(func() {
			assert.Equal(1, count.Add())
		})
		return ctx.JSONP(http.StatusOK, "cb123", []string{"Hello"})
	})
	app.Use(func(ctx *Context) error {
		panic("this middleware unreachable")
	})

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	res, err := RequestBy("GET", host)
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(200, res.StatusCode)
	assert.Equal(`/**/ typeof cb123 === "function" && cb123(["Hello"]);`, PickRes(res.Text()).(string))
	assert.Equal(2, count.Int())
	assert.Equal("nosniff", res.Header.Get(HeaderXContentTypeOptions))
	assert.Equal(MIMEApplicationJavaScriptCharsetUTF8, res.Header.Get(HeaderContentType))

	res, err = RequestBy("GET", host+"/error")
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(500, res.StatusCode)
	assert.True(strings.Contains(PickRes(res.Text()).(string), "json: unsupported value"))
	assert.Equal(3, count.Int())
	assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))
}

type XMLData struct {
	Type    string `xml:"type,attr,omitempty"`
	Comment string `xml:",comment"`
	Number  string `xml:",chardata"`
}

func TestGearContextXML(t *testing.T) {
	assert := assert.New(t)

	count := Counter(0)
	app := New()
	app.Use(func(ctx *Context) error {
		if ctx.Path == "/error" {
			ctx.OnEnd(func() {
				assert.Equal(3, count.Add())
			})
			ctx.After(func() {
				panic("this hook unreachable")
			})

			return ctx.XML(http.StatusOK, struct {
				Value interface{}
				Err   string
				Kind  reflect.Kind
			}{
				Value: make(chan bool),
				Err:   "xml: unsupported type: chan bool",
				Kind:  reflect.Chan,
			})
		}

		ctx.OnEnd(func() {
			assert.Equal(2, count.Add())
		})
		ctx.After(func() {
			assert.Equal(1, count.Add())
		})
		return ctx.XML(http.StatusOK, XMLData{"test", "golang", "123"})
	})
	app.Use(func(ctx *Context) error {
		panic("this middleware unreachable")
	})

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	res, err := RequestBy("GET", host)
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(200, res.StatusCode)
	assert.Equal(`<XMLData type="test"><!--golang-->123</XMLData>`, PickRes(res.Text()).(string))
	assert.Equal(2, count.Int())
	assert.Equal(MIMEApplicationXMLCharsetUTF8, res.Header.Get(HeaderContentType))

	res, err = RequestBy("GET", host+"/error")
	assert.Nil(err)

	time.Sleep(time.Millisecond)
	assert.Equal(500, res.StatusCode)
	assert.True(strings.Contains(PickRes(res.Text()).(string), "xml: unsupported type"))
	assert.Equal(3, count.Int())
	assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))
}

type SenderTest struct{}

func (s *SenderTest) Send(ctx *Context, code int, data interface{}) error {
	switch v := data.(type) {
	case []byte:
		ctx.Type(MIMETextPlainCharsetUTF8)
		return ctx.End(code, v)
	case string:
		return ctx.HTML(code, v)
	case error:
		return ctx.Error(v)
	default:
		return ctx.JSON(code, data)
	}
}

func TestGearContextSend(t *testing.T) {
	t.Run("should panic when sender not registered", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.Send(http.StatusOK, "data")
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"Error","message":"sender not registered"}`, PickRes(res.Text()).(string))
	})

	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Panics(func() {
			app.Set(SetSender, struct{}{})
		})
		app.Set(SetSender, &SenderTest{})
		app.Use(func(ctx *Context) error {
			switch ctx.Path {
			case "/text":
				return ctx.Send(http.StatusOK, []byte("Hello, Gear!"))
			case "/html":
				return ctx.Send(http.StatusOK, "<h1>Hello, Gear!</h1>")
			case "/error":
				return ctx.Send(http.StatusOK, Err.WithMsg("some error"))
			default:
				return ctx.Send(http.StatusOK, map[string]string{"value": "Hello, Gear!"})
			}
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/text")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal(MIMETextPlainCharsetUTF8, res.Header.Get(HeaderContentType))
		assert.Equal("Hello, Gear!", PickRes(res.Text()).(string))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal(MIMETextHTMLCharsetUTF8, res.Header.Get(HeaderContentType))
		assert.Equal("<h1>Hello, Gear!</h1>", PickRes(res.Text()).(string))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/error")
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))
		assert.Equal(`{"error":"Error","message":"some error"}`, PickRes(res.Text()).(string))

		res, err = RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))
		assert.Equal(`{"value":"Hello, Gear!"}`, PickRes(res.Text()).(string))
	})
}

type RenderTest struct {
	tpl *template.Template
}

func (t *RenderTest) Render(ctx *Context, w io.Writer, name string, data interface{}) (err error) {
	if err = t.tpl.ExecuteTemplate(w, name, data); err != nil {
		err = ErrNotFound.From(err)
	}
	return
}

func TestGearContextRender(t *testing.T) {
	t.Run("should panic when renderer not registered", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.Render(http.StatusOK, "index", []string{})
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"Error","message":"renderer not registered"}`, PickRes(res.Text()).(string))
	})

	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Panics(func() {
			app.Set(SetRenderer, struct{}{})
		})
		app.Set(SetRenderer, &RenderTest{
			tpl: template.Must(template.New("hello").Parse("Hello, {{.}}!")),
		})
		app.Use(func(ctx *Context) error {
			return ctx.Render(http.StatusOK, "hello", "Gear")
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("Hello, Gear!", PickRes(res.Text()).(string))
	})

	t.Run("when return error", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Set(SetRenderer, &RenderTest{
			tpl: template.Must(template.New("hello").Parse("Hello, {{.}}!")),
		})
		app.Use(func(ctx *Context) error {
			return ctx.Render(http.StatusOK, "helloA", "Gear")
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		assert.Equal(`{"error":"NotFound","message":"html/template: \"helloA\" is undefined"}`, PickRes(res.Text()).(string))
	})
}

func TestGearContextStream(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/hello.html")
	if err != nil {
		panic(Err.From(err))
	}

	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			file, err := os.Open("testdata/hello.html")
			if err != nil {
				return err
			}
			return ctx.Stream(http.StatusOK, MIMETextHTMLCharsetUTF8, file)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal(MIMETextHTMLCharsetUTF8, res.Header.Get(HeaderContentType))
		assert.Equal(string(data), PickRes(res.Text()).(string))
	})

	t.Run("should not change if context ended", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := New()
		app.Set(SetLogger, log.New(&buf, "TEST: ", 0))
		app.Use(func(ctx *Context) error {
			ctx.End(204)

			file, err := os.Open("testdata/hello.html")
			if err != nil {
				panic(err)
			}
			ctx.Stream(200, MIMETextHTMLCharsetUTF8, file)
			assert.Equal(204, ctx.Res.Status())
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		assert.Equal("", buf.String())
	})
}

func TestGearContextAttachment(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/README.md")
	if err != nil {
		panic(Err.From(err))
	}

	t.Run("should work as attachment", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			file, err := os.Open("testdata/README.md")
			if err != nil {
				return err
			}
			return ctx.Attachment("Gear 设计说明.md", time.Time{}, file)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal(`attachment; filename="Gear+%E8%AE%BE%E8%AE%A1%E8%AF%B4%E6%98%8E.md"; filename*=UTF-8''Gear%20%E8%AE%BE%E8%AE%A1%E8%AF%B4%E6%98%8E.md`,
			res.Header.Get(HeaderContentDisposition))
		assert.Equal(MIMETextPlainCharsetUTF8, res.Header.Get(HeaderContentType))
		assert.Equal(string(data), PickRes(res.Text()).(string))
	})

	t.Run("should work as inline", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			file, err := os.Open("testdata/README.md")
			if err != nil {
				return err
			}
			return ctx.Attachment("README.md", time.Time{}, file, true)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal(`inline; filename="README.md"`, res.Header.Get(HeaderContentDisposition))
		assert.Equal(MIMETextPlainCharsetUTF8, res.Header.Get(HeaderContentType))
		assert.Equal(string(data), PickRes(res.Text()).(string))
	})
}

func TestGearContextRedirect(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		redirected := false
		app.Use(func(ctx *Context) error {
			if ctx.Path != "/ok" {
				ctx.OnEnd(func() {
					assert.Equal(ctx.Res.Status(), 301)
				})
				redirected = true
				ctx.Status(301)
				return ctx.Redirect("/ok")
			}
			return ctx.HTML(200, "OK")
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.True(redirected)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
	})

	t.Run("should correct status code", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		redirected := false
		app.Use(func(ctx *Context) error {
			if ctx.Path != "/ok" {
				ctx.OnEnd(func() {
					assert.Equal(ctx.Res.Status(), 302)
				})
				redirected = true
				return ctx.Redirect("/ok")
			}
			return ctx.HTML(200, "OK")
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.True(redirected)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
	})
}

func TestGearContextError(t *testing.T) {
	t.Run("should work with *Error", func(t *testing.T) {
		assert := assert.New(t)

		count := Counter(0)
		app := New()
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				count.Add()
			})
			err := ErrUnauthorized.WithMsg("some error")
			return ctx.Error(err)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(0, count.Int())
		assert.Equal(401, res.StatusCode)
		assert.Equal(`{"error":"Unauthorized","message":"some error"}`, PickRes(res.Text()).(string))
	})

	t.Run("should work with error", func(t *testing.T) {
		assert := assert.New(t)

		count := Counter(0)
		app := New()
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				count.Add()
			})
			return ctx.Error(errors.New("some error"))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(0, count.Int())
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"InternalServerError","message":"some error"}`, PickRes(res.Text()).(string))
	})

	t.Run("with nil error", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			var err error
			ctx.After(func() {
				count.Add()
			})
			return ctx.Error(err)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(0, count.Int())
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"InternalServerError","message":"nil error"}`, PickRes(res.Text()).(string))
	})

	t.Run("should transform with app.onerror", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Set(SetOnError, func(ctx *Context, err HTTPError) {
			ctx.JSON(err.Status(), err)
		})
		app.Use(func(ctx *Context) error {
			return ctx.Error(&MyHTTPError{400, "some error"})
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(400, res.StatusCode)
		assert.Equal(`{"code":400,"error":"some error"}`, PickRes(res.Text()).(string))
		assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))
	})
}

type MyHTTPError struct {
	Code int    `json:"code"`
	Msg  string `json:"error"`
}

func (e *MyHTTPError) Status() int   { return e.Code }
func (e *MyHTTPError) Error() string { return e.Msg }

func TestGearContextErrorStatus(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				count.Add()
			})
			return ctx.ErrorStatus(401)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(0, count.Int())
		assert.Equal(401, res.StatusCode)
		assert.Equal(`{"error":"Unauthorized","message":""}`, PickRes(res.Text()).(string))
	})

	t.Run("should 500 with invalid status", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			return ctx.ErrorStatus(301)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal(`{"error":"InternalServerError","message":"invalid error status"}`, PickRes(res.Text()).(string))
	})
}

func TestGearContextEnd(t *testing.T) {
	t.Run("should work with code 0", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			ctx.Status(204)
			return ctx.End(0)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
	})

	t.Run("should work with code", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			ctx.Status(500)
			return ctx.End(200, []byte("OK"))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
	})

	t.Run("should work with two args", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			ctx.Status(400)
			return ctx.End(200, []byte("OK"))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
	})

	t.Run("should not change if ctx ended", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			ctx.End(200, []byte("OK"))
			return ctx.End(400, []byte("Err"))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("OK", PickRes(res.Text()).(string))
	})
}

func TestGearContextAfter(t *testing.T) {
	t.Run("should work in LIFO order", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				assert.Equal(4, count.Add())
				ctx.Status(204)
			})
			ctx.After(func() {
				assert.Equal(3, count.Add())
			})
			ctx.After(func() {
				assert.Equal(2, count.Add())
			})
			assert.Equal(1, count.Add())
			return ctx.End(400)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(4, count.Int())
		assert.Equal(204, res.StatusCode)
	})

	t.Run("add hook when ctx ended", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				assert.Equal(3, count.Add())
			})

			count.Add()
			assert.Equal(1, count.Int())
			ctx.Status(204)
			ctx.Res.ended.setTrue()
			ctx.After(func() {
				assert.Equal(2, count.Add())
			})
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(3, count.Int())
		assert.Equal(204, res.StatusCode)
	})

	t.Run("can't add hook if header wrote", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				assert.Panics(func() {
					ctx.After(func() {})
				})
				assert.Equal(2, count.Add())
			})

			count.Add()
			assert.Equal(1, count.Int())
			ctx.End(204)
			assert.Panics(func() {
				ctx.After(func() {})
			})
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(2, count.Int())
		assert.Equal(204, res.StatusCode)
	})
}

func TestGearContextOnEnd(t *testing.T) {
	t.Run("should work in LIFO order", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			ctx.OnEnd(func() {
				assert.Equal(4, count.Add())
				ctx.Status(500)
			})
			ctx.After(func() {
				assert.Equal(2, count.Add())
				ctx.Status(204)
			})
			ctx.OnEnd(func() {
				assert.Equal(3, count.Add())
			})
			assert.Equal(1, count.Add())
			return ctx.End(400)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(4, count.Int())
		assert.Equal(204, res.StatusCode)
	})

	t.Run("add hook when ctx ended", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			ctx.OnEnd(func() {
				assert.Equal(3, count.Add())
			})

			count.Add()
			assert.Equal(1, count.Int())
			ctx.Status(204)
			ctx.Res.ended.setTrue()
			ctx.OnEnd(func() {
				assert.Equal(2, count.Add())
			})
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(3, count.Int())
		assert.Equal(204, res.StatusCode)
	})

	t.Run("can't add hook if header wrote", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := Counter(0)
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				assert.Panics(func() {
					ctx.OnEnd(func() {})
				})
				assert.Equal(2, count.Add())
			})

			ctx.OnEnd(func() {
				assert.Panics(func() {
					ctx.OnEnd(func() {})
				})
				assert.Equal(3, count.Add())
			})

			assert.Equal(1, count.Add())
			ctx.End(204)
			assert.Panics(func() {
				ctx.OnEnd(func() {})
			})
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)

		time.Sleep(time.Millisecond)
		assert.Equal(3, count.Int())
		assert.Equal(204, res.StatusCode)
	})
}
