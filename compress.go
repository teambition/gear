package gear

import (
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
)

// Compressible interface is use to enable compress response content.
type Compressible interface {
	// Compressible checks the response Content-Type and Content-Length to
	// determine whether to compress.
	// `length == 0` means response body maybe stream, or will be writed later.
	Compressible(contentType string, contentLength int) bool
}

// DefaultCompress is defalut Compress implemented. Use it to enable compress:
//
//	app.Set(gear.SetCompress, &gear.DefaultCompress{})
type DefaultCompress struct{}

// Compressible implemented Compress interface.
func (d *DefaultCompress) Compressible(contentType string, contentLength int) bool {
	if contentLength > 0 && contentLength <= 1024 {
		return false
	}
	return contentType != ""
}

// ThresholdCompress is an impelementation with transhold. The transhold defines the // minimun content length to enable compressible check.
type ThresholdCompress int

// Compressible implemented Compress interface.
//
//	app := gear.New()
//	app.Set(gear.SetCompress, gear.ThresholdCompress(128))
//
//	// Add a static middleware
//	app.Use(static.New(static.Options{
//		Root:   "./",
//		Prefix: "/",
//	}))
//	app.Error(app.Listen(":3000")) // http://127.0.0.1:3000/
func (tc ThresholdCompress) Compressible(contentType string, contentLength int) bool {
	if contentLength < int(tc) {
		return false
	}

	return contentType != ""
}

// http.ResponseWriter wrapper
type compressWriter struct {
	compress Compressible
	encoding string
	writer   io.WriteCloser
	res      *Response
	rw       http.ResponseWriter // underlying http.ResponseWriter
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Encoding
func newCompress(res *Response, c Compressible, encoding string) *compressWriter {
	switch encoding {
	case "gzip", "deflate":
		return &compressWriter{
			compress: c,
			res:      res,
			rw:       res.rw,
			encoding: encoding,
		}
	default:
		return nil
	}
}

func (cw *compressWriter) WriteHeader(code int) {
	defer cw.rw.WriteHeader(code)

	if !isEmptyStatus(code) &&
		cw.compress.Compressible(cw.res.Get(HeaderContentType), len(cw.res.body)) {
		var w io.WriteCloser

		// http://www.gzip.org/zlib/zlib_faq.html#faq38
		switch cw.encoding {
		case "gzip": // recommend
			w = gzip.NewWriter(cw.rw)
		case "deflate": // should be zlib
			w = zlib.NewWriter(cw.rw)
		}

		if w != nil {
			cw.writer = w
			cw.res.Del(HeaderContentLength)
			cw.res.Set(HeaderContentEncoding, cw.encoding)
			cw.res.Vary(HeaderAcceptEncoding)
		}
	}
}

func (cw *compressWriter) Header() http.Header {
	return cw.rw.Header()
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	if cw.writer != nil {
		return cw.writer.Write(b)
	}
	return cw.rw.Write(b)
}

func (cw *compressWriter) Close() error {
	if cw.writer != nil {
		return cw.writer.Close()
	}
	return nil
}
