package gweb

import (
	"bytes"
	"net/http"
)

// Response implement ResponseWriter
type Response struct {
	ctx      *Context
	res      http.ResponseWriter
	Status   int         // response Status
	Type     string      // response Content-Type
	Body     []byte      // response Content
	Header   http.Header // response Header
	finished bool
}

// NewResponse ...
func NewResponse(w http.ResponseWriter, ctx *Context) *Response {
	r := new(Response)
	r.res = w
	r.ctx = ctx
	r.Status = 404
	r.Header = w.Header()
	return r
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

func (r *Response) Write(buf []byte) (int, error) {
	r.finished = true
	return r.res.Write(buf)
}

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

func (r *Response) end(code int) {
	if r.finished {
		return
	}
	r.ctx.Lock()
	if code > 0 {
		r.Status = code
	}
	statusText := http.StatusText(r.Status)
	if statusText == "" {
		r.Status = 500
		statusText = http.StatusText(r.Status)
	}
	if r.Body == nil && r.Status >= 300 {
		r.stringBody(statusText)
	}
	r.WriteHeader(0)
	if r.Body != nil {
		r.Write(r.Body)
	}
	r.ctx.Unlock()
}
