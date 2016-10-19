package gweb

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"
)

// NewContext returns a Context
func NewContext(w http.ResponseWriter, req *http.Request) *Context {
	ctx := new(Context)
	ctx.Context = req.Context()
	ctx.Req = req
	ctx.Res = NewResponse(w, ctx)
	ctx.Method = req.Method
	ctx.Path = req.URL.Path
	return ctx
}

// Context docs
type Context struct {
	Context context.Context
	Req     *http.Request
	Res     *Response
	Method  string
	Path    string
	vals    map[interface{}]interface{}
	mu      sync.RWMutex
}

// implement Locker interface

func (ctx *Context) Lock() {
	ctx.mu.Lock()
}

func (ctx *Context) Unlock() {
	ctx.mu.Unlock()
}

func (ctx *Context) RLock() {
	ctx.mu.RLock()
}

func (ctx *Context) RUnlock() {
	ctx.mu.RUnlock()
}

// implement context.Context interface

func (ctx *Context) Deadline() (time.Time, bool) {
	return ctx.Context.Deadline()
}

func (ctx *Context) Done() <-chan struct{} {
	return ctx.Context.Done()
}

func (ctx *Context) Err() error {
	return ctx.Context.Err()
}

func (ctx *Context) Value(key interface{}) (val interface{}) {
	val = ctx.vals[key]
	return
}

// func (ctx *Context) String() string {
// 	return "gweb.Context"
// }

func (ctx *Context) SetValue(key, val interface{}) {
	ctx.Lock()
	if ctx.vals == nil {
		ctx.vals = make(map[interface{}]interface{})
	}
	ctx.vals[key] = val
	ctx.Unlock()
}

func (ctx *Context) Html(str string) {
	ctx.Set("content-type", "text/html; charset=utf-8")
	ctx.Res.Body(bytes.NewBufferString(str).Bytes())
}

func (ctx *Context) Text(str string) {
	ctx.Set("content-type", "text/plain; charset=utf-8")
	ctx.Res.Body(bytes.NewBufferString(str).Bytes())
}

func (ctx *Context) Attachment(filename string) {

}

func (ctx *Context) Redirect(url string) {

}

// Get gets the first value associated with the given key. If there are no values associated with the key, Get returns "". To access multiple values of a key, access the map directly with CanonicalHeaderKey.
func (ctx *Context) Get(key string) string {
	return ctx.Req.Header.Get(key)
}

// Set sets the header entries associated with key to the single element value. It replaces any existing values associated with key.
func (ctx *Context) Set(key, value string) {
	ctx.Res.Set(key, value)
}

func (ctx *Context) Status(code int) {
	ctx.Res.Status(code)
}
