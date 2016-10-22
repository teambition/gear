package gear

import (
	"bytes"
	"net/http"
)

// Response wraps an http.ResponseWriter and implements its interface to be used
// by an HTTP handler to construct an HTTP response.
type Response struct {
	ctx      *gearCtx
	res      http.ResponseWriter
	Status   int         // response Status
	Type     string      // response Content-Type
	Body     []byte      // response Content
	header   http.Header // response Header
	finished bool
}

func (r *Response) reset(w http.ResponseWriter) {
	r.res = w
	r.Type = ""
	r.Body = nil
	r.Status = 500
	r.finished = false
	if w != nil {
		r.header = w.Header()
	} else {
		r.header = nil
	}
}

// Add adds the key, value pair to the header. It appends to any existing values associated with key.
func (r *Response) Add(key, value string) {
	r.header.Add(key, value)
}

// Del deletes the values associated with key.
func (r *Response) Del(key string) {
	r.header.Del(key)
}

// Get gets the first value associated with the given key. If there are no values associated with the key, Get returns "". To access multiple values of a key, access the map directly with CanonicalHeaderKey.
func (r *Response) Get(key string) string {
	return r.header.Get(key)
}

// Set sets the header entries associated with key to the single element value. It replaces any existing values associated with key.
func (r *Response) Set(key, value string) {
	r.header.Set(key, value)
}

// Header returns the header map that will be sent by WriteHeader.
func (r *Response) Header() http.Header {
	return r.header
}

// Write writes the data to the connection as part of an HTTP reply.
func (r *Response) Write(buf []byte) (int, error) {
	if !r.finished {
		r.WriteHeader(r.Status)
	}
	return r.res.Write(buf)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (r *Response) WriteHeader(code int) {
	r.Status = code

	if r.ctx.afterHooks != nil {
		r.ctx.runAfterHooks()
	}
	if r.ctx.endHooks != nil {
		r.ctx.runEndHooks()
	}
	r.finished = true
	r.res.WriteHeader(r.Status) // r.Status maybe changed in hooks
}

func (r *Response) respond() (err error) {
	if r.finished {
		return
	}
	r.WriteHeader(r.Status)
	if r.Body == nil && r.Status >= 300 {
		r.Body = stringToBytes(http.StatusText(r.Status))
	}
	if r.Body != nil {
		_, err = r.Write(r.Body)
	}
	return
}

func stringToBytes(str string) []byte {
	return bytes.NewBufferString(str).Bytes()
}
