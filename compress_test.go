package gear

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"

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
		assert.Panics(t, func() {
			app.Set(SetCompress, struct{}{})
		})
		app.Set(SetCompress, &DefaultCompress{})

		r := NewRouter()
		r.Get("/full", func(ctx *Context) error {
			EqualPtr(t, ctx.Res.Header(), ctx.Res.rw.Header())

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

			req, _ := NewRequst("GET", host+"/full")

			req.Header.Set("Accept-Encoding", "gzip, deflate")

			res, err := DefaultClientDo(req)
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

			req, _ := NewRequst("GET", host+"/full")
			req.Header.Set("Accept-Encoding", "deflate;q=1.0, br;q=0.8, *;q=0.1")
			res, err := DefaultClientDo(req)
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

		t.Run("when Non-support Accept-Encoding", func(t *testing.T) {
			assert := assert.New(t)

			req, _ := NewRequst("GET", host+"/full")
			req.Header.Set("Accept-Encoding", "compress, br")

			res, err := DefaultClientDo(req)
			assert.Nil(err)
			assert.True(res.OK())
			content := PickRes(res.Content()).([]byte)

			assert.Equal("", res.Header.Get(HeaderContentEncoding))
			assert.Equal("", res.Header.Get(HeaderVary))
			assert.Equal(body, content)
		})

		t.Run("compress threshold", func(t *testing.T) {
			assert := assert.New(t)

			req, _ := NewRequst("GET", host+"/short")
			req.Header.Set("Accept-Encoding", "gzip, deflate")
			res, err := DefaultClientDo(req)
			assert.Nil(err)
			assert.True(res.OK())
			content := PickRes(res.Content()).([]byte)

			assert.Equal("", res.Header.Get(HeaderContentEncoding))
			assert.Equal("", res.Header.Get(HeaderVary))
			assert.Equal(strconv.FormatInt(int64(len(short)), 10), res.Header.Get(HeaderContentLength))
			assert.Equal(short, content)
		})

		t.Run("when Content Type not set", func(t *testing.T) {
			assert := assert.New(t)

			body := []byte(strings.Repeat("你好，Gear", 500))

			app := New()
			app.Set(SetCompress, &DefaultCompress{})

			r := NewRouter()
			r.Get("/full", func(ctx *Context) error {
				return ctx.End(http.StatusOK, body)
			})
			app.UseHandler(r)

			srv := app.Start()
			defer srv.Close()

			host := "http://" + srv.Addr().String()

			req, _ := NewRequst("GET", host+"/full")

			res, err := DefaultClientDo(req)
			assert.Nil(err)
			assert.True(res.OK())
			content := PickRes(res.Content()).([]byte)

			assert.Equal("", res.Header.Get(HeaderContentEncoding))
			assert.Equal("", res.Header.Get(HeaderVary))
			assert.Equal(body, content)
		})

		t.Run("when status code should not compress", func(t *testing.T) {
			assert := assert.New(t)

			app := New()
			app.Set(SetCompress, &DefaultCompress{})

			r := NewRouter()
			r.Get("/204", func(ctx *Context) error {
				ctx.Type(MIMETextPlainCharsetUTF8)
				return ctx.End(204)
			})
			r.Get("/205", func(ctx *Context) error {
				ctx.Type(MIMETextPlainCharsetUTF8)
				return ctx.End(205)
			})
			r.Get("/304", func(ctx *Context) error {
				ctx.Type(MIMETextPlainCharsetUTF8)
				return ctx.End(304)
			})
			app.UseHandler(r)

			srv := app.Start()
			defer srv.Close()

			host := "http://" + srv.Addr().String()

			res, _ := RequestBy("GET", host+"/204")
			assert.Equal(204, res.StatusCode)
			assert.Equal("", res.Header.Get(HeaderContentEncoding))

			res, _ = RequestBy("GET", host+"/205")
			assert.Equal(205, res.StatusCode)
			assert.Equal("", res.Header.Get(HeaderContentEncoding))

			res, _ = RequestBy("GET", host+"/304")
			assert.Equal(304, res.StatusCode)
			assert.Equal("", res.Header.Get(HeaderContentEncoding))
		})
	})
}
