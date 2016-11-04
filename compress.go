package gear

import (
	"io"
	"net/http"
)

type compressWriter struct {
	http.ResponseWriter

	threshold int
	body      *[]byte
	writer    io.WriteCloser
}

func (w *compressWriter) WriteHeader(code int) {
	defer w.ResponseWriter.WriteHeader(code)

	h := w.ResponseWriter.Header()

	if code == http.StatusNoContent ||
		code == http.StatusResetContent ||
		code == http.StatusNotModified {
		return
	}

	if len(*w.body) < w.threshold {
		return
	}

	h.Set(HeaderVary, HeaderAcceptEncoding)
	h.Del(HeaderContentLength)
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
