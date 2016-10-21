package gear

import (
	"bytes"
	"net/http"
)

// Response wraps an http.ResponseWriter and implements its interface to be used
// by an HTTP handler to construct an HTTP response.
type Response struct {
	ctx      Context
	res      http.ResponseWriter
	Status   int         // response Status
	Type     string      // response Content-Type
	Body     []byte      // response Content
	Header   http.Header // response Header
	finished bool
}

func (r *Response) reset(w http.ResponseWriter) {
	r.res = w
	r.Type = ""
	r.Body = nil
	r.finished = false
	r.Status = 404
	r.Header = w.Header()
}

// Add adds the key, value pair to the header. It appends to any existing values associated with key.
func (r *Response) Add(key, value string) {
	r.Header.Add(key, value)
}

// Del deletes the values associated with key.
func (r *Response) Del(key string) {
	r.Header.Del(key)
}

// Get gets the first value associated with the given key. If there are no values associated with the key, Get returns "". To access multiple values of a key, access the map directly with CanonicalHeaderKey.
func (r *Response) Get(key string) string {
	return r.Header.Get(key)
}

// Set sets the header entries associated with key to the single element value. It replaces any existing values associated with key.
func (r *Response) Set(key, value string) {
	r.Header.Set(key, value)
}

// Write writes the data to the connection as part of an HTTP reply.
func (r *Response) Write(buf []byte) (int, error) {
	r.finished = true
	return r.res.Write(buf)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (r *Response) WriteHeader(code int) {
	if code > 0 {
		r.Status = code
	}
	r.finished = true
	r.res.WriteHeader(r.Status)
}

func (r *Response) stringBody(str string) {
	r.Body = bytes.NewBufferString(str).Bytes()
}

func (r *Response) respond() {
	if r.finished {
		return
	}
	r.ctx.Lock()
	r.WriteHeader(0)
	if r.Body == nil && r.Status >= 300 {
		r.stringBody(http.StatusText(r.Status))
	}
	if r.Body != nil {
		r.Write(r.Body)
	}
	r.ctx.Unlock()
}
