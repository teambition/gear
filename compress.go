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
	// Compressible checks the response Content-Type and Content-Length to
	// determine whether to compress.
	// Recommend use mime database https://github.com/GitbookIO/mimedb to find
	// which Content-Type is compressible.
	// `length == 0` means response body maybe stream, or will be writed later.
	Compressible(contentType string, contentLength int) bool
}

// DefaultCompress is defalut Compress implemented. Use it to enable compress:
//
//  app.Set("AppCompress", &gear.DefaultCompress{})
//
type DefaultCompress struct{}

// Compressible implemented Compress interface.
func (d *DefaultCompress) Compressible(contentType string, contentLength int) bool {
	if contentLength > 0 && contentLength <= 1024 {
		return false
	}
	return contentType != ""
}

type compressWriter struct {
	body     *[]byte
	compress Compress
	encoding string
	writer   io.WriteCloser
	res      http.ResponseWriter
}

func newCompress(res *Response, c Compress, acceptEncoding string) *compressWriter {
	encodings := strings.Split(acceptEncoding, ",")
	encoding := strings.TrimSpace(encodings[0])
	switch encoding {
	case "gzip", "deflate":
		return &compressWriter{
			body:     &res.Body,
			compress: c,
			encoding: encoding,
			res:      res.res,
		}
	default:
		return nil
	}
}

func (cw *compressWriter) WriteHeader(code int) {
	defer cw.res.WriteHeader(code)

	switch code {
	case http.StatusNoContent, http.StatusResetContent, http.StatusNotModified:
		return
	}

	header := cw.res.Header()
	if cw.compress.Compressible(header.Get(HeaderContentType), len(*cw.body)) {
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
