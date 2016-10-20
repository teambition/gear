package gweb

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// NewContext returns a Context
func NewContext(w http.ResponseWriter, req *http.Request) *Context {
	ctx := new(Context)
	ctx.ctx, ctx.cancelCtx = context.WithCancel(req.Context())
	ctx.Req = req
	ctx.Res = NewResponse(w, ctx)
	ctx.Host = req.Host
	ctx.Method = req.Method
	ctx.Path = req.URL.Path
	ctx.hooks = make([]Middleware, 0)
	return ctx
}

// Context docs
type Context struct {
	ctx       context.Context
	Req       *http.Request
	Res       *Response
	Host      string
	Method    string
	Path      string
	cancelCtx context.CancelFunc
	ended     bool
	vals      map[interface{}]interface{}
	hooks     []Middleware
	mu        sync.RWMutex
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
	return ctx.ctx.Deadline()
}

func (ctx *Context) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx *Context) Err() error {
	return ctx.ctx.Err()
}

func (ctx *Context) Value(key interface{}) (val interface{}) {
	var ok bool
	if val, ok = ctx.vals[key]; !ok {
		val = ctx.ctx.Value(key)
	}
	return
}

func (ctx *Context) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx.ctx)
}

func (ctx *Context) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx.ctx, deadline)
}

func (ctx *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx.ctx, timeout)
}

func (ctx *Context) WithValue(key, val interface{}) context.Context {
	return context.WithValue(ctx.ctx, key, val)
}

func (ctx *Context) String() string {
	return fmt.Sprintf("gweb.Context{Req: %v, Res: %v}", ctx.Req, ctx.Res)
}

func (ctx *Context) SetValue(key, val interface{}) {
	ctx.Lock()
	if ctx.vals == nil {
		ctx.vals = make(map[interface{}]interface{})
	}
	ctx.vals[key] = val
	ctx.Unlock()
}

func (ctx *Context) After(fn Middleware) {
	ctx.hooks = append(ctx.hooks, fn)
}

func (ctx *Context) Html(str string) {
	ctx.Set("content-type", "text/html; charset=utf-8")
	ctx.Res.stringBody(str)
}

func (ctx *Context) Text(str string) {
	ctx.Set("content-type", "text/plain; charset=utf-8")
	ctx.Res.stringBody(str)
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
	ctx.Res.Status = code
}

func (ctx *Context) End(code int) {
	ctx.ended = true
	if code > 0 {
		ctx.Res.Status = code
	}
}

func (ctx *Context) Cancel() {
	ctx.cancelCtx()
}
