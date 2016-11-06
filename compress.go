package gear

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Compress interface is use to enable compress response context.
type Compress interface {
	// Compressible checks the response content type to decide whether to compress.
	Compressible(contentType string) bool
	// Threshold return the minimun response size in bytes to compress.
	// Default value is 1024 (1 kb).
	Threshold() int
}

// DefaultCompress is defalut Compress implemented. Use it to enable compress:
//
//  app.Set("AppCompress", &gear.DefaultCompress{})
//
type DefaultCompress struct{}

// Compressible implemented Compress interface.
func (d *DefaultCompress) Compressible(contentType string) bool {
	// Should use mime database https://github.com/GitbookIO/mimedb to find
	// which contentType is compressible
	if contentType == "" {
		return false
	}
	return true
}

// Threshold implemented Compress interface.
func (d *DefaultCompress) Threshold() int {
	return 1024
}

type compressWriter struct {
	body     *[]byte
	compress Compress
	encoding string
	writer   io.WriteCloser
	res      http.ResponseWriter
}

func newCompress(res *Response, c Compress, acceptEncoding string) *compressWriter {
	var encoding string
	encodings := strings.Split(acceptEncoding, ",")

loop:
	for _, encoding = range encodings {
		encoding = strings.TrimSpace(encoding)
		switch encoding {
		case "gzip", "deflate":
			break loop
		default:
			return nil
		}
	}

	return &compressWriter{
		body:     &res.Body,
		compress: c,
		encoding: encoding,
		res:      res.res,
	}
}

func (cw *compressWriter) WriteHeader(code int) {
	defer cw.res.WriteHeader(code)

	if code == http.StatusNoContent ||
		code == http.StatusResetContent ||
		code == http.StatusNotModified {
		return
	}

	length := len(*cw.body)
	if length > 0 && length < cw.compress.Threshold() {
		return
	}

	header := cw.res.Header()
	if cw.compress.Compressible(header.Get(HeaderContentType)) {
		var w io.WriteCloser

		switch cw.encoding {
		case "gzip":
			w, _ = gzip.NewWriterLevel(cw.res, gzip.DefaultCompression)
		case "deflate":
			w, _ = flate.NewWriter(cw.res, flate.DefaultCompression)
		}

		if w != nil {
			cw.writer = w
			header.Set(HeaderVary, HeaderAcceptEncoding)
			header.Set(HeaderContentEncoding, cw.encoding)
			header.Del(HeaderContentLength)
		}
	}
}

func (cw *compressWriter) Header() http.Header {
	return cw.res.Header()
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	if cw.writer != nil {
		return cw.writer.Write(b)
	}
	return cw.res.Write(b)
}

func (cw *compressWriter) Close() error {
	if cw.writer != nil {
		return cw.writer.Close()
	}
	return nil
}
