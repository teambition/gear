package gear

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGearResponseCompress(t *testing.T) {
	assert := assert.New(t)

	var gzipedData bytes.Buffer
	var deflatedData bytes.Buffer

	data := []byte("test")

	gw := gzip.NewWriter(&gzipedData)
	defer gw.Close()
	gw.Write(data)
	gw.Close()

	dw, err := flate.NewWriter(&deflatedData, flate.DefaultCompression)
	assert.Nil(err)
	defer dw.Close()
	dw.Write(data)
	dw.Close()

	r := NewRouter("", false)
	r.Handle("GET", "/", func(ctx *Context) error {
		return ctx.End(http.StatusOK, data)
	})

	app := New()
	app.UseHandler(r)
	app.compress = true

	srv := app.Start()
	defer srv.Close()

	host := "http://" + srv.Addr().String()

	t.Run("gzip compress", func(t *testing.T) {
		req := NewRequst()
		req.Headers = map[string]string{"Accept-Encoding": "gzip,deflate"}

		res, err := req.Get(host)
		assert.Nil(err)
		assert.True(res.OK())
		assert.Equal("gzip", res.Header.Get(HeaderContentEncoding))
		assert.Equal(HeaderAcceptEncoding, res.Header.Get(HeaderVary))

		content, err := ioutil.ReadAll(res.Body)
		assert.Nil(err)
		assert.Equal(gzipedData.Bytes(), content)
	})

	t.Run("deflate compress", func(t *testing.T) {
		req := NewRequst()
		req.Headers = map[string]string{"Accept-Encoding": "deflate"}

		res, err := req.Get(host)
		assert.Nil(err)
		assert.True(res.OK())
		assert.Equal("deflate", res.Header.Get(HeaderContentEncoding))
		assert.Equal(HeaderAcceptEncoding, res.Header.Get(HeaderVary))

		content, err := ioutil.ReadAll(res.Body)
		assert.Nil(err)
		assert.Equal(deflatedData.Bytes(), content)
	})

	t.Run("compress filter", func(t *testing.T) {
		app.compressFilter = func(_ string) bool { return false }

		req := NewRequst()
		req.Headers = map[string]string{"Accept-Encoding": "gzip,deflate"}

		res, err := req.Get(host)
		assert.Nil(err)
		assert.True(res.OK())
		assert.Equal("", res.Header.Get(HeaderContentEncoding))
		assert.Equal("", res.Header.Get(HeaderVary))

		content, err := ioutil.ReadAll(res.Body)
		assert.Nil(err)
		assert.Equal(data, content)
	})
}
