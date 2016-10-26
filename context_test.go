package gear

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type ContextTest struct {
	Ctx *Context
	Res *httptest.ResponseRecorder
}

func (c *ContextTest) Result() *http.Response {
	return c.Res.Result()
}

func (c *ContextTest) Body() (val string) {
	body, err := ioutil.ReadAll(c.Res.Result().Body)
	if err == nil {
		val = bytes.NewBuffer(body).String()
	}
	return
}

func NewCtx(app *Gear, method, url string, body io.Reader) *ContextTest {
	req := httptest.NewRequest(method, url, body)
	res := httptest.NewRecorder()
	return &ContextTest{Ctx: NewContext(app, res, req), Res: res}
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

	t.Run("type Any", func(t *testing.T) {
		t.Run("should get the same value with the same ctx", func(t *testing.T) {
			ctx := NewCtx(app, "GET", "http://example.com/foo", nil)
			val, err := ctx.Ctx.Any(ctxAny)
			require.Nil(t, err)
			res := val.(*ctxAnyResult)
			require.Equal(t, ctx.Ctx.Host, res.Host)
			require.Equal(t, ctx.Ctx.Path, res.Path)

			val2, _ := ctx.Ctx.Any(ctxAny)
			EqualPtr(t, val, val2)
		})

		t.Run("should get different value with different ctx", func(t *testing.T) {
			ctx := NewCtx(app, "GET", "http://example.com/foo", nil)
			val, err := ctx.Ctx.Any(ctxAny)
			require.Nil(t, err)

			ctx2 := NewCtx(app, "GET", "http://example.com/foo", nil)
			val2, err2 := ctx2.Ctx.Any(ctxAny)
			require.Nil(t, err2)
			NotEqualPtr(t, val, val2)
		})

		t.Run("should get error", func(t *testing.T) {
			ctx := NewCtx(app, "POST", "http://example.com/foo", nil)
			val, err := ctx.Ctx.Any(ctxAny)
			require.Nil(t, val)
			require.NotNil(t, err)
			require.Equal(t, "POST", err.Error())
		})
	})

	t.Run("SetAny with interface{}", func(t *testing.T) {
		ctx := NewCtx(app, "POST", "http://example.com/foo", nil)
		val, err := ctx.Ctx.Any(struct{}{})
		require.Nil(t, val)
		require.Equal(t, "non-existent key", err.Error())

		ctx.Ctx.SetAny(struct{}{}, true)
		val, err = ctx.Ctx.Any(struct{}{})
		require.Nil(t, err)
		require.True(t, val.(bool))
	})
}
