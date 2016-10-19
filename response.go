package gweb

import "net/http"

// Response implement ResponseWriter
type Response struct {
	status     int // status code passed to WriteHeader
	err        error
	res        http.ResponseWriter
	body       []byte
	headerSent bool
	bodySent   bool
	ctx        *Context
	Header     http.Header
}

// NewResponse ...
func NewResponse(w http.ResponseWriter, ctx *Context) *Response {
	r := new(Response)
	r.res = w
	r.ctx = ctx
	r.status = 404
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

func (r *Response) Status(code int) {
	r.status = code
}

func (r *Response) Body(buf []byte) {
	r.body = buf
}

func (r *Response) Write(buf []byte) (int, error) {
	return r.res.Write(buf)
}

func (r *Response) WriteHeader() {
	if r.headerSent {
		return
	}
	r.headerSent = true
	r.res.WriteHeader(r.status)
}

func (r *Response) respond() {
	r.WriteHeader()
	r.Write(r.body)
}
