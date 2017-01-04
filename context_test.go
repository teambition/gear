package gear

import (
	"bytes"
	"context"
	"errors"
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

	"github.com/stretchr/testify/assert"
)

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
	var count int32

	app := New()
	app.Use(func(ctx *Context) error {
		// ctx.WithValue
		c := ctx.WithValue("test", "abc")
		assert.Equal("abc", c.Value("test").(string))
		s := c.Value(http.ServerContextKey)
		EqualPtr(t, s, app.Server)

		c1, _ := ctx.WithCancel()
		c2, _ := ctx.WithDeadline(time.Now().Add(time.Second))
		c3, _ := ctx.WithTimeout(time.Second)

		go func() {
			<-c1.Done()
			assert.True(ctx.ended.isTrue())
			atomic.AddInt32(&count, 1)
		}()

		go func() {
			<-c2.Done()
			assert.True(ctx.ended.isTrue())
			atomic.AddInt32(&count, 1)
		}()

		go func() {
			<-c3.Done()
			assert.True(ctx.ended.isTrue())
			atomic.AddInt32(&count, 1)
		}()

		ctx.Status(404)
		ctx.Cancel()

		assert.True(ctx.ended.isTrue())
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
	assert.Equal(504, res.StatusCode)
	assert.Equal(atomic.LoadInt32(&count), int32(3))
}

func TestGearContextTiming(t *testing.T) {
	data := []string{"hello"}

	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			res, err := ctx.Timing(time.Millisecond*15, func() interface{} {
				time.Sleep(time.Millisecond * 10)
				return data
			})
			assert.True(err == nil)
			assert.Equal(data, res.([]string))
			return ctx.JSON(200, res.([]string))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal(`["hello"]`, PickRes(res.Text()).(string))
	})

	t.Run("when timeout", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			res, err := ctx.Timing(time.Millisecond*10, func() interface{} {
				time.Sleep(time.Millisecond * 15)
				return data
			})
			assert.True(res == nil)
			assert.Equal(context.DeadlineExceeded, err)
			return ctx.Error(err)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("context deadline exceeded", PickRes(res.Text()).(string))
	})

	t.Run("when context timeout", func(t *testing.T) {
		assert := assert.New(t)

		app := New()

		app.Set("AppTimeout", time.Millisecond*10)
		app.Use(func(ctx *Context) error {
			res, err := ctx.Timing(time.Millisecond*20, func() interface{} {
				time.Sleep(time.Millisecond * 15)
				return data
			})
			assert.True(res == nil)
			assert.Equal(context.DeadlineExceeded, err)
			time.Sleep(time.Millisecond * 10)
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(504, res.StatusCode)
		assert.Equal("context deadline exceeded", PickRes(res.Text()).(string))
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
		app.Set("AppEnv", 123)
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
		assert.Equal("Gear: non-existent key", err.Error())

		ctx.SetAny(struct{}{}, true)
		val, err = ctx.Any(struct{}{})
		assert.Nil(err)
		assert.True(val.(bool))
	})

	t.Run("Setting", func(t *testing.T) {
		assert := assert.New(t)

		ctx := CtxTest(app, "POST", "http://example.com/foo", nil)
		assert.Equal("development", ctx.Setting("AppEnv").(string))
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
	assert := assert.New(t)

	app := New()
	r := NewRouter()
	r.Get("/XForwardedFor", func(ctx *Context) error {
		assert.Equal("127.0.0.10", ctx.IP().String())
		return ctx.End(http.StatusNoContent)
	})
	r.Get("/XRealIP", func(ctx *Context) error {
		assert.Equal("127.0.0.20", ctx.IP().String())
		return ctx.End(http.StatusNoContent)
	})
	r.Get("/", func(ctx *Context) error {
		assert.NotNil(ctx.IP())
		return ctx.End(http.StatusNoContent)
	})
	r.Get("/err", func(ctx *Context) error {
		assert.Nil(ctx.IP())
		return ctx.End(http.StatusNoContent)
	})
	app.UseHandler(r)

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	req, _ := NewRequst("GET", host+"/XForwardedFor")
	req.Header.Set("X-Forwarded-For", "127.0.0.10")

	res, err := DefaultClientDo(req)
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)

	req, _ = NewRequst("GET", host+"/XRealIP")
	req.Header.Set("X-Real-IP", "127.0.0.20")

	res, err = DefaultClientDo(req)
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)

	req, _ = NewRequst("GET", host)
	res, err = DefaultClientDo(req)
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)

	req, _ = NewRequst("GET", host+"/err")
	req.Header.Set("X-Real-IP", "1.2.3")

	res, err = DefaultClientDo(req)
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)
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
		assert.Equal([]string{"123"}, ctx.QueryValues("id"))
		assert.Equal("", ctx.Query("other"))
		return ctx.End(http.StatusNoContent)
	})
	r.Get("/view", func(ctx *Context) error {
		assert.Nil(ctx.QueryValues("other"))
		assert.Equal("123", ctx.Query("id"))
		assert.Equal([]string{"123", "abc"}, ctx.QueryValues("id"))
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

func TestGearContextCookie(t *testing.T) {
	assert := assert.New(t)

	app := New()
	r := NewRouter()
	r.Get("/", func(ctx *Context) error {
		c1, _ := ctx.Cookie("Gear")
		c2, _ := ctx.Cookie("Gear.sig")

		assert.Equal("test", c1.Value)
		assert.Equal("abc123", c2.Value)
		assert.Equal(2, len(ctx.Cookies()))

		c1.Value = "Hello"
		c1.Path = "/test"
		ctx.SetCookie(c1)
		return ctx.End(http.StatusNoContent)
	})
	app.UseHandler(r)

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()
	req, _ := NewRequst("GET", host)
	res, err := DefaultClientDoWithCookies(req, map[string]string{"Gear": "test", "Gear.sig": "abc123"})
	assert.Nil(err)
	assert.Equal(204, res.StatusCode)
	c := res.Cookies()[0]
	assert.Equal("Gear", c.Name)
	assert.Equal("Hello", c.Value)
	assert.Equal("/test", c.Path)
}

func TestGearContextGetSet(t *testing.T) {
	assert := assert.New(t)

	app := New()
	ctx := CtxTest(app, "GET", "http://example.com/foo", nil)

	assert.Equal("", ctx.Get(HeaderAccept))
	ctx.Set(HeaderWarning, "Some error")
	res := CtxResult(ctx)
	assert.Equal("Some error", res.Header.Get(HeaderWarning))
}

func TestGearContextStatus(t *testing.T) {
	assert := assert.New(t)

	app := New()
	ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
	assert.Equal(ctx.Status(), 0)
	assert.Equal(ctx.Status(1), 0)
	assert.Equal(ctx.Status(401), 401)
	assert.Equal(ctx.Status(1), 401)
}

func TestGearContextType(t *testing.T) {
	assert := assert.New(t)

	app := New()
	ctx := CtxTest(app, "GET", "http://example.com/foo", nil)
	assert.Equal("", ctx.Type())
	ctx.Type(MIMEApplicationJSONCharsetUTF8)
	assert.Equal(MIMEApplicationJSONCharsetUTF8, ctx.Res.Get(HeaderContentType))
	assert.Equal(MIMEApplicationJSONCharsetUTF8, ctx.Type())
}

func TestGearContextHTML(t *testing.T) {
	assert := assert.New(t)

	app := New()
	count := 0
	app.Use(func(ctx *Context) error {
		ctx.OnEnd(func() {
			count++
			assert.Equal(2, count)
		})
		ctx.After(func() {
			count++
			assert.Equal(1, count)
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
	assert.Equal(200, res.StatusCode)
	assert.Equal("Hello", PickRes(res.Text()).(string))
	assert.Equal(2, count)
}

func TestGearContextJSON(t *testing.T) {
	assert := assert.New(t)

	app := New()
	count := 0
	app.Use(func(ctx *Context) error {
		if ctx.Path == "/error" {
			ctx.OnEnd(func() {
				count++
				assert.Equal(3, count)
			})
			ctx.After(func() {
				panic("this hook unreachable")
			})
			return ctx.JSON(http.StatusOK, math.NaN())
		}

		ctx.OnEnd(func() {
			count++
			assert.Equal(2, count)
		})
		ctx.After(func() {
			count++
			assert.Equal(1, count)
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
	assert.Equal(200, res.StatusCode)
	assert.Equal(`["Hello"]`, PickRes(res.Text()).(string))
	assert.Equal(2, count)
	assert.Equal(MIMEApplicationJSONCharsetUTF8, res.Header.Get(HeaderContentType))

	res, err = RequestBy("GET", host+"/error")
	assert.Nil(err)
	assert.Equal(500, res.StatusCode)
	assert.True(strings.Contains(PickRes(res.Text()).(string), "json: unsupported value"))
	assert.Equal(3, count)
	assert.Equal(MIMETextPlainCharsetUTF8, res.Header.Get(HeaderContentType))
}

func TestGearContextJSONP(t *testing.T) {
	assert := assert.New(t)

	app := New()
	count := 0
	app.Use(func(ctx *Context) error {
		if ctx.Path == "/error" {
			ctx.OnEnd(func() {
				count++
				assert.Equal(3, count)
			})
			ctx.After(func() {
				panic("this hook unreachable")
			})
			return ctx.JSONP(http.StatusOK, "cb123", math.NaN())
		}

		ctx.OnEnd(func() {
			count++
			assert.Equal(2, count)
		})
		ctx.After(func() {
			count++
			assert.Equal(1, count)
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
	assert.Equal(200, res.StatusCode)
	assert.Equal(`/**/ typeof cb123 === "function" && cb123(["Hello"]);`, PickRes(res.Text()).(string))
	assert.Equal(2, count)
	assert.Equal("nosniff", res.Header.Get(HeaderXContentTypeOptions))
	assert.Equal(MIMEApplicationJavaScriptCharsetUTF8, res.Header.Get(HeaderContentType))

	res, err = RequestBy("GET", host+"/error")
	assert.Nil(err)
	assert.Equal(500, res.StatusCode)
	assert.True(strings.Contains(PickRes(res.Text()).(string), "json: unsupported value"))
	assert.Equal(3, count)
	assert.Equal(MIMETextPlainCharsetUTF8, res.Header.Get(HeaderContentType))
}

type XMLData struct {
	Type    string `xml:"type,attr,omitempty"`
	Comment string `xml:",comment"`
	Number  string `xml:",chardata"`
}

func TestGearContextXML(t *testing.T) {
	assert := assert.New(t)

	app := New()
	count := 0
	app.Use(func(ctx *Context) error {
		if ctx.Path == "/error" {
			ctx.OnEnd(func() {
				count++
				assert.Equal(3, count)
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
			count++
			assert.Equal(2, count)
		})
		ctx.After(func() {
			count++
			assert.Equal(1, count)
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
	assert.Equal(200, res.StatusCode)
	assert.Equal(`<XMLData type="test"><!--golang-->123</XMLData>`, PickRes(res.Text()).(string))
	assert.Equal(2, count)
	assert.Equal(MIMEApplicationXMLCharsetUTF8, res.Header.Get(HeaderContentType))

	res, err = RequestBy("GET", host+"/error")
	assert.Nil(err)
	assert.Equal(500, res.StatusCode)
	assert.True(strings.Contains(PickRes(res.Text()).(string), "xml: unsupported type"))
	assert.Equal(3, count)
	assert.Equal(MIMETextPlainCharsetUTF8, res.Header.Get(HeaderContentType))
}

type RenderTest struct {
	tpl *template.Template
}

func (t *RenderTest) Render(ctx *Context, w io.Writer, name string, data interface{}) (err error) {
	if err = t.tpl.ExecuteTemplate(w, name, data); err != nil {
		err = &Error{404, err.Error(), err}
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
		assert.True(strings.Contains(PickRes(res.Text()).(string), "Gear: renderer not registered"))
	})

	t.Run("should work", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		assert.Panics(func() {
			app.Set("AppRenderer", struct{}{})
		})
		app.Set("AppRenderer", &RenderTest{
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
		app.Set("AppRenderer", &RenderTest{
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
		assert.Equal(`html/template: "helloA" is undefined`, PickRes(res.Text()).(string))
	})
}

func TestGearContextStream(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/hello.html")
	if err != nil {
		panic(NewAppError(err.Error()))
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
		app.Set("AppLogger", log.New(&buf, "TEST: ", 0))
		app.Use(func(ctx *Context) error {
			ctx.End(204)

			file, err := os.Open("testdata/hello.html")
			if err != nil {
				panic(err)
			}
			ctx.Stream(200, MIMETextHTMLCharsetUTF8, file)
			assert.Equal(204, ctx.Status())
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
		panic(NewAppError(err.Error()))
	}

	t.Run("should work as attachment", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		app.Use(func(ctx *Context) error {
			file, err := os.Open("testdata/README.md")
			if err != nil {
				return err
			}
			return ctx.Attachment("README.md", file)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("attachment; filename=README.md", res.Header.Get(HeaderContentDisposition))
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
			return ctx.Attachment("README.md", file, true)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("inline; filename=README.md", res.Header.Get(HeaderContentDisposition))
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
					assert.Equal(ctx.Status(), 301)
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
					assert.Equal(ctx.Status(), 302)
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

		app := New()
		count := 0
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				count++
			})
			err := &Error{Code: 401, Msg: "some error"}
			return ctx.Error(err)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(0, count)
		assert.Equal(401, res.StatusCode)
		assert.Equal("some error", PickRes(res.Text()).(string))
	})

	t.Run("should work with error", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := 0
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				count++
			})
			return ctx.Error(errors.New("some error"))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(0, count)
		assert.Equal(500, res.StatusCode)
		assert.Equal("some error", PickRes(res.Text()).(string))
	})

	t.Run("with nil error", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := 0
		app.Use(func(ctx *Context) error {
			var err error
			ctx.After(func() {
				count++
			})
			return ctx.Error(err)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(0, count)
		assert.Equal(500, res.StatusCode)
		assert.Equal("Gear: nil-error", PickRes(res.Text()).(string))
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
			ctx.ended.setTrue()
			return ctx.End(200, []byte("OK"))
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(421, res.StatusCode)
		assert.Equal("", PickRes(res.Text()).(string))
	})
}

func TestGearContextAfter(t *testing.T) {
	t.Run("should work in LIFO order", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := 0
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				count++
				assert.Equal(4, count)
				ctx.Status(204)
			})
			ctx.After(func() {
				count++
				assert.Equal(3, count)
			})
			ctx.After(func() {
				count++
				assert.Equal(2, count)
			})
			count++
			assert.Equal(1, count)
			return ctx.End(400)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(4, count)
		assert.Equal(204, res.StatusCode)
	})

	t.Run("can't add hook if ctx ended", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := 0
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				assert.Panics(func() {
					ctx.After(func() {})
				})
				count++
				assert.Equal(2, count)
			})

			count++
			assert.Equal(1, count)
			ctx.Status(204)
			ctx.ended.setTrue()
			assert.Panics(func() {
				ctx.After(func() {})
			})
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(2, count)
		assert.Equal(204, res.StatusCode)
	})
}

func TestGearContextOnEnd(t *testing.T) {
	t.Run("should work in LIFO order", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := 0
		app.Use(func(ctx *Context) error {
			ctx.OnEnd(func() {
				count++
				assert.Equal(4, count)
				ctx.Status(500)
			})
			ctx.After(func() {
				count++
				assert.Equal(2, count)
				ctx.Status(204)
			})
			ctx.OnEnd(func() {
				count++
				assert.Equal(3, count)
			})
			count++
			assert.Equal(1, count)
			return ctx.End(400)
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(4, count)
		assert.Equal(204, res.StatusCode)
	})

	t.Run("can't add hook if ctx ended", func(t *testing.T) {
		assert := assert.New(t)

		app := New()
		count := 0
		app.Use(func(ctx *Context) error {
			ctx.After(func() {
				assert.Panics(func() {
					ctx.OnEnd(func() {})
				})
				count++
				assert.Equal(2, count)
			})

			ctx.OnEnd(func() {
				assert.Panics(func() {
					ctx.OnEnd(func() {})
				})
				count++
				assert.Equal(3, count)
			})

			count++
			assert.Equal(1, count)
			ctx.Status(204)
			ctx.ended.setTrue()
			assert.Panics(func() {
				ctx.OnEnd(func() {})
			})
			return nil
		})

		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(3, count)
		assert.Equal(204, res.StatusCode)
	})
}
