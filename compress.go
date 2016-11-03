package gear

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type compressWriter struct {
	http.ResponseWriter
	*http.Request

	writer io.WriteCloser
}

func (w *compressWriter) WriteHeader(code int) {
	defer w.ResponseWriter.WriteHeader(code)

	h := w.ResponseWriter.Header()

	if h.Get(HeaderContentEncoding) != "" {
		return
	}

	if code == http.StatusNoContent ||
		code == http.StatusResetContent ||
		code == http.StatusNotModified {
		return
	}

	for _, encoding := range strings.Split(w.Request.Header.Get(HeaderAcceptEncoding), ",") {
		switch encoding {
		case "gzip":
			h.Set(HeaderContentEncoding, "gzip")
			h.Set(HeaderVary, HeaderAcceptEncoding)
			h.Del(HeaderContentLength)

			gw := gzip.NewWriter(w.ResponseWriter)

			w.writer = gw
			return
		case "deflate":
			h.Set(HeaderContentEncoding, "deflate")
			h.Set(HeaderVary, HeaderAcceptEncoding)
			h.Del(HeaderContentLength)

			fw, _ := flate.NewWriter(w.ResponseWriter, flate.DefaultCompression)

			w.writer = fw
			return
		}
	}
}

func (w *compressWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *compressWriter) Write(b []byte) (int, error) {
	if w.writer == nil {
		return w.ResponseWriter.Write(b)
	}

	return w.writer.Write(b)
}

func (w *compressWriter) Close() error {
	if w.writer == nil {
		return nil
	}

	return w.writer.Close()
}
