package gear

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

func TestGearResponseCompress(t *testing.T) {
	gzipCompress := func(buf []byte) []byte {
		var data bytes.Buffer
		if gw, err := gzip.NewWriterLevel(&data, gzip.DefaultCompression); err == nil {
			gw.Write(buf)
			gw.Close()
		}
		return data.Bytes()
	}
	gzipUnCompress := func(buf []byte) []byte {
		var data []byte
		if gr, err := gzip.NewReader(bytes.NewBuffer(buf)); err == nil {
			data, _ = ioutil.ReadAll(gr)
			gr.Close()
		}
		return data
	}

	flateCompress := func(buf []byte) []byte {
		var data bytes.Buffer
		if fw, err := flate.NewWriter(&data, flate.DefaultCompression); err == nil {
			fw.Write(buf)
			fw.Close()
		}
		return data.Bytes()
	}
	flateUnCompress := func(buf []byte) []byte {
		var data []byte
		fr := flate.NewReader(bytes.NewBuffer(buf))
		data, _ = ioutil.ReadAll(fr)
		fr.Close()
		return data
	}

	t.Run("DefaultCompress", func(t *testing.T) {
		body := []byte(strings.Repeat("你好，Gear", 500))
		short := []byte(strings.Repeat("你好，Gear", 50))

		app := New()
		app.Set("AppCompress", &DefaultCompress{})

		r := NewRouter()
		r.Get("/full", func(ctx *Context) error {
			ctx.Type(MIMETextPlainCharsetUTF8)
			return ctx.End(http.StatusOK, body)
		})
		r.Get("/short", func(ctx *Context) error {
			ctx.Type(MIMETextPlainCharsetUTF8)
			return ctx.End(http.StatusOK, short)
		})
		app.UseHandler(r)

		srv := app.Start()
		defer srv.Close()

		host := "http://" + srv.Addr().String()

		t.Run("gzip compress", func(t *testing.T) {
			assert := assert.New(t)

			req := NewRequst()
			req.Headers["Accept-Encoding"] = "gzip, deflate"

			res, err := req.Get(host + "/full")
			assert.Nil(err)
			assert.True(res.OK())
			content := PickRes(ioutil.ReadAll(res.Body)).([]byte)

			buf := gzipCompress(body)
			assert.True(len(buf) < len(body))
			assert.True(len(buf) == len(content))
			assert.Equal("gzip", res.Header.Get(HeaderContentEncoding))
			assert.Equal(HeaderAcceptEncoding, res.Header.Get(HeaderVary))
			assert.Equal(strconv.FormatInt(int64(len(buf)), 10), res.Header.Get(HeaderContentLength))
			content = gzipUnCompress(content)
			assert.Equal(body, content)
		})

		t.Run("deflate compress", func(t *testing.T) {
			assert := assert.New(t)

			req := NewRequst()
			req.Headers["Accept-Encoding"] = "deflate,gzip"

			res, err := req.Get(host + "/full")
			assert.Nil(err)
			assert.True(res.OK())
			content := PickRes(ioutil.ReadAll(res.Body)).([]byte)

			buf := flateCompress(body)
			assert.True(len(buf) < len(body))
			assert.True(len(buf) == len(content))
			assert.Equal("deflate", res.Header.Get(HeaderContentEncoding))
			assert.Equal(HeaderAcceptEncoding, res.Header.Get(HeaderVary))
			assert.Equal(strconv.FormatInt(int64(len(buf)), 10), res.Header.Get(HeaderContentLength))
			content = flateUnCompress(content)
			assert.Equal(body, content)
		})

		t.Run("when no Accept-Encoding", func(t *testing.T) {
			assert := assert.New(t)

			req := NewRequst()
			req.Headers["Accept-Encoding"] = ""

			res, err := req.Get(host + "/full")
			assert.Nil(err)
			assert.True(res.OK())
			content := PickRes(res.Content()).([]byte)

			assert.Equal("", res.Header.Get(HeaderContentEncoding))
			assert.Equal("", res.Header.Get(HeaderVary))
			assert.Equal(strconv.FormatInt(int64(len(body)), 10), res.Header.Get(HeaderContentLength))
			assert.Equal(body, content)
		})

		t.Run("compress threshold", func(t *testing.T) {
			assert := assert.New(t)

			req := NewRequst()
			req.Headers["Accept-Encoding"] = "gzip, deflate"

			res, err := req.Get(host + "/short")
			assert.Nil(err)
			assert.True(res.OK())
			content := PickRes(res.Content()).([]byte)

			assert.Equal("", res.Header.Get(HeaderContentEncoding))
			assert.Equal("", res.Header.Get(HeaderVary))
			assert.Equal(strconv.FormatInt(int64(len(short)), 10), res.Header.Get(HeaderContentLength))
			assert.Equal(short, content)
		})
	})
}
