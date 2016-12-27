package gear

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type contextKey int

const paramsKey contextKey = 0

// Any interface is used by ctx.Any.
type Any interface {
	New(ctx *Context) (interface{}, error)
}

// Context represents the context of the current HTTP request. It holds request and
// response objects, path, path parameters, data, registered handler and content.Context.
type Context struct {
	app *App
	Req *http.Request
	Res *Response

	Host   string
	Method string
	Path   string

	ended      bool // indicate that app middlewares run out.
	query      url.Values
	afterHooks []func()
	endHooks   []func()
	ctx        context.Context
	cancelCtx  context.CancelFunc
	kv         map[interface{}]interface{}
	mu         sync.RWMutex
}

// NewContext creates an instance of Context. Export for testing middleware.
func NewContext(app *App, w http.ResponseWriter, req *http.Request) *Context {
	ctx := &Context{app: app, Req: req}
	ctx.Res = newResponse(ctx, w)

	ctx.Host = req.Host
	ctx.Method = req.Method
	ctx.Path = req.URL.Path
	ctx.kv = make(map[interface{}]interface{})
	if app.timeout == 0 {
		ctx.ctx, ctx.cancelCtx = context.WithCancel(req.Context())
	} else {
		ctx.ctx, ctx.cancelCtx = context.WithTimeout(req.Context(), app.timeout)
	}
	return ctx
}

// ----- implement context.Context interface ----- //

// Deadline returns the time when work done on behalf of this context
// should be canceled.
func (ctx *Context) Deadline() (time.Time, bool) {
	return ctx.ctx.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled.
func (ctx *Context) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

// Err returns a non-nil error value after Done is closed.
func (ctx *Context) Err() error {
	return ctx.ctx.Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (ctx *Context) Value(key interface{}) (val interface{}) {
	return ctx.ctx.Value(key)
}

// Cancel cancel the ctx and all it' children context.
// The ctx' process will ended too.
func (ctx *Context) Cancel() {
	ctx.cleanAfterHooks()
	ctx.setEnd(false) // end the middleware process
	ctx.cancelCtx()
}

// WithCancel returns a copy of the ctx with a new Done channel.
// The returned context's Done channel is closed when the returned cancel function is called or when the parent context's Done channel is closed, whichever happens first.
func (ctx *Context) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx.ctx)
}

// WithDeadline returns a copy of the ctx with the deadline adjusted to be no later than d.
func (ctx *Context) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx.ctx, deadline)
}

// WithTimeout returns WithDeadline(time.Now().Add(timeout)).
func (ctx *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx.ctx, timeout)
}

// WithValue returns a copy of the ctx in which the value associated with key is val.
func (ctx *Context) WithValue(key, val interface{}) context.Context {
	return context.WithValue(ctx.ctx, key, val)
}

// Timing runs fn with the given time limit. If a call runs for longer than its time limit,
// it will return context.DeadlineExceeded as error, otherwise return fn's result.
func (ctx *Context) Timing(dt time.Duration, fn func() interface{}) (interface{}, error) {
	ct, cancel := ctx.WithTimeout(dt)
	defer cancel()

	ch := make(chan interface{}, 1)
	go func() { ch <- fn() }()
	select {
	case <-ct.Done():
		return nil, ct.Err()
	case res := <-ch:
		return res, nil
	}
}

// Any returns the value on this ctx by key. If key is instance of Any and
// value not set, any.New will be called to eval the value, and then set to the ctx.
// if any.New returns error, the value will not be set.
//
//  // create some Any type for your project.
//  type someAnyType struct{}
//  type someAnyResult struct {
//  	r *http.Request
//  }
//
//  var someAnyKey = &someAnyType{}
//
//  func (t *someAnyType) New(ctx *gear.Context) (interface{}, error) {
//  	return &someAnyResult{r: ctx.Req}, nil
//  }
//
//  // use it in app
//  if val, err := ctx.Any(someAnyKey); err == nil {
//  	res := val.(*someAnyResult)
//  }
//
func (ctx *Context) Any(any interface{}) (val interface{}, err error) {
	var ok bool
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if val, ok = ctx.kv[any]; !ok {
		switch res := any.(type) {
		case Any:
			if val, err = res.New(ctx); err == nil {
				ctx.kv[any] = val
			}
		default:
			return nil, NewAppError("non-existent key")
		}
	}
	return
}

// SetAny save a key, value pair on the ctx.
// package logging used ctx.SetAny and ctx.Any to implement "logger.FromCtx":
//
//  func (l *Logger) FromCtx(ctx *gear.Context) Log {
//  	if any, err := ctx.Any(l); err == nil {
//  		return any.(Log)
//  	}
//  	log := Log{}
//  	ctx.SetAny(l, log)
//  	l.init(log, ctx)
//  	return log
//  }
//
func (ctx *Context) SetAny(key, val interface{}) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.kv[key] = val
}

// Setting returns App's settings by key
//
//  fmt.Println(ctx.Setting("AppEnv").(string) == "development")
//  app.Set("AppEnv", "production")
//  fmt.Println(ctx.Setting("AppEnv").(string) == "production")
//
func (ctx *Context) Setting(key string) interface{} {
	if val, ok := ctx.app.settings[key]; ok {
		return val
	}
	return nil
}

// IP returns the client's network address based on `X-Forwarded-For`
// or `X-Real-IP` request header.
func (ctx *Context) IP() net.IP {
	ra := ctx.Req.RemoteAddr
	if ip := ctx.Req.Header.Get(HeaderXForwardedFor); ip != "" {
		ra = ip
	} else if ip := ctx.Req.Header.Get(HeaderXRealIP); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	return net.ParseIP(ra)
}

// Param returns path parameter by name.
func (ctx *Context) Param(key string) (val string) {
	if res, _ := ctx.Any(paramsKey); res != nil {
		val, _ = res.(map[string]string)[key]
	}
	return
}

// Query returns the query param for the provided name.
func (ctx *Context) Query(name string) string {
	if ctx.query == nil {
		ctx.query = ctx.Req.URL.Query()
	}
	return ctx.query.Get(name)
}

// QueryValues returns all query params for the provided name.
func (ctx *Context) QueryValues(name string) []string {
	if ctx.query == nil {
		ctx.query = ctx.Req.URL.Query()
	}
	return ctx.query[name]
}

// Cookie returns the named cookie provided in the request.
func (ctx *Context) Cookie(name string) (*http.Cookie, error) {
	return ctx.Req.Cookie(name)
}

// SetCookie adds a `Set-Cookie` header in HTTP response.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	http.SetCookie(ctx.Res, cookie)
}

// Cookies returns the HTTP cookies sent with the request.
func (ctx *Context) Cookies() []*http.Cookie {
	return ctx.Req.Cookies()
}

// Get retrieves data from the request Header.
func (ctx *Context) Get(key string) string {
	return ctx.Req.Header.Get(key)
}

// Set saves data to the response Header.
func (ctx *Context) Set(key, value string) {
	ctx.Res.Set(key, value)
}

// Type set a content type to response
func (ctx *Context) Type(str string) {
	if str == "" {
		ctx.Res.Del(HeaderContentType)
	} else {
		ctx.Res.Set(HeaderContentType, str)
	}
}

// String set an text body with status code to response.
func (ctx *Context) String(code int, str string) {
	ctx.Res.SetStatus(code)
	ctx.Type(MIMETextPlainCharsetUTF8)
	ctx.Res.setBody([]byte(str))
}

// HTML set an Html body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) HTML(code int, str string) error {
	ctx.Type(MIMETextHTMLCharsetUTF8)
	return ctx.End(code, []byte(str))
}

// JSON set a JSON body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSON(code int, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		return ctx.Error(err)
	}
	return ctx.JSONBlob(code, buf)
}

// JSONBlob set a JSON blob body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSONBlob(code int, buf []byte) error {
	ctx.Type(MIMEApplicationJSONCharsetUTF8)
	return ctx.End(code, buf)
}

// JSONP sends a JSONP response with status code. It uses `callback` to construct the JSONP payload.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSONP(code int, callback string, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		return ctx.Error(err)
	}
	return ctx.JSONPBlob(code, callback, buf)
}

// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
// to construct the JSONP payload.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSONPBlob(code int, callback string, buf []byte) error {
	ctx.Type(MIMEApplicationJavaScriptCharsetUTF8)
	ctx.Set(HeaderXContentTypeOptions, "nosniff")
	// the /**/ is a specific security mitigation for "Rosetta Flash JSONP abuse"
	// @see http://miki.it/blog/2014/7/8/abusing-jsonp-with-rosetta-flash/
	// the typeof check is just to reduce client error noise
	buf = bytes.Join([][]byte{[]byte(`/**/ typeof ` + callback + ` === "function" && ` + callback + "("),
		buf, []byte(");")}, []byte{})
	return ctx.End(code, buf)
}

// XML set an XML body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) XML(code int, val interface{}) error {
	buf, err := xml.Marshal(val)
	if err != nil {
		return ctx.Error(err)
	}
	return ctx.XMLBlob(code, buf)
}

// XMLBlob set a XML blob body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) XMLBlob(code int, buf []byte) error {
	ctx.Type(MIMEApplicationXMLCharsetUTF8)
	return ctx.End(code, buf)
}

// Render renders a template with data and sends a text/html response with status
// code. Templates can be registered using `app.Renderer = Renderer`.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Render(code int, name string, data interface{}) (err error) {
	if ctx.app.renderer == nil {
		return NewAppError("renderer not registered")
	}
	buf := new(bytes.Buffer)
	if err = ctx.app.renderer.Render(ctx, buf, name, data); err == nil {
		ctx.Type(MIMETextHTMLCharsetUTF8)
		return ctx.End(code, buf.Bytes())
	}
	return
}

// Stream sends a streaming response with status code and content type.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Stream(code int, contentType string, r io.Reader) (err error) {
	if err = ctx.setEnd(true); err == nil {
		ctx.Res.SetStatus(code)
		ctx.Type(contentType)
		_, err = io.Copy(ctx.Res, r)
	}
	return
}

// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
// client to save the file. If inline is true, the attachment will sends as inline,
// opening the file in the browser.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Attachment(name string, content io.ReadSeeker, inline ...bool) (err error) {
	dispositionType := "attachment"
	if len(inline) > 0 && inline[0] {
		dispositionType = "inline"
	}
	if err = ctx.setEnd(true); err == nil {
		ctx.Set(HeaderContentDisposition, fmt.Sprintf("%s; filename=%s", dispositionType, name))
		http.ServeContent(ctx.Res, ctx.Req, name, time.Time{}, content)
	}
	return
}

// Redirect redirects the request with status code. It is a wrap of http.Redirect.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Redirect(code int, url string) (err error) {
	if err = ctx.setEnd(true); err == nil {
		http.Redirect(ctx.Res, ctx.Req, url, code)
	}
	return
}

// Error send a error message with status code to response.
// It will end the ctx. The middlewares after current middleware and "after hooks" will not run.
// "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Error(e error) (err error) {
	ctx.cleanAfterHooks() // clear afterHooks when any error
	if e := ParseError(e); e != nil {
		ctx.Type(MIMETextPlainCharsetUTF8)
		return ctx.End(e.Status(), []byte(e.Error()))
	}
	return &Error{Code: 500, Msg: NewAppError("nil-error").Error()}
}

// End end the ctx with bytes and status code optionally.
// After it's called, the rest of middleware handles will not run.
// But "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) End(code int, buf ...[]byte) (err error) {
	if err = ctx.setEnd(true); err == nil {
		if code != 0 {
			ctx.Res.SetStatus(code)
		}
		if len(buf) != 0 {
			ctx.Res.setBody(buf[0])
		}
		err = ctx.Res.respond()
	}
	return
}

// After add a "after hook" to the ctx that will run after middleware process,
// but before Response.WriteHeader.
func (ctx *Context) After(hook func()) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.ended { // should not add afterHooks if ctx.ended
		panic(NewAppError(`can't add "after hook" after context ended`))
	}
	ctx.afterHooks = append(ctx.afterHooks, hook)
}

func (ctx *Context) cleanAfterHooks() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.afterHooks = nil
}

// OnEnd add a "end hook" to the ctx that will run after Response.WriteHeader.
func (ctx *Context) OnEnd(hook func()) {
	if ctx.ended { // should not add endHooks if ctx.ended
		panic(NewAppError(`can't add "end hook" after context ended`))
	}
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.endHooks = append(ctx.endHooks, hook)
}

func (ctx *Context) cleanEndHooks() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.endHooks = nil
}

func (ctx *Context) setEnd(check bool) (err error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if check && ctx.ended {
		err = NewAppError("context is ended")
	} else {
		ctx.ended = true
	}
	return
}

func (ctx *Context) isEnded() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.ended
}

func (ctx *Context) salvage(err *Error) {
	ctx.app.Error(err)
	ctx.cleanAfterHooks()
	ctx.Set(HeaderXContentTypeOptions, "nosniff")
	ctx.String(err.Status(), err.Error())
	ctx.Res.respond()
}

func (ctx *Context) handleCompress() (cw *compressWriter) {
	if ctx.app.compress != nil && ctx.Method != http.MethodHead && ctx.Method != http.MethodOptions {
		if cw = newCompress(ctx.Res, ctx.app.compress, ctx.Get(HeaderAcceptEncoding)); cw != nil {
			ctx.Res.res = cw
		}
	}
	return
}
