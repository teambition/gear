package gear

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-http-utils/cookie"
	"github.com/go-http-utils/negotiator"
)

type contextKey int

const paramsKey contextKey = 0

// Any interface is used by ctx.Any.
type Any interface {
	New(ctx *Context) (interface{}, error)
}

// BodyTemplate interface is used by ctx.Any.
type BodyTemplate interface {
	Validate() error
}

// Context represents the context of the current HTTP request. It holds request and
// response objects, path, path parameters, data, registered handler and content.Context.
type Context struct {
	app     *App
	Req     *http.Request
	Res     *Response
	Cookies *cookie.Cookies // https://github.com/go-http-utils/cookie

	Host   string
	Method string
	Path   string

	ended      atomicBool // indicate that app middlewares run out.
	query      url.Values
	afterHooks []func()
	endHooks   []func()
	ctx        context.Context
	_ctx       context.Context
	cancelCtx  context.CancelFunc
	kv         map[interface{}]interface{}
}

// NewContext creates an instance of Context. Export for testing middleware.
func NewContext(app *App, w http.ResponseWriter, r *http.Request) *Context {
	ctx := &Context{app: app, Req: r}
	ctx.Res = newResponse(ctx, w)
	ctx.Cookies = cookie.New(w, r, app.keys)

	ctx.Host = r.Host
	ctx.Method = r.Method
	ctx.Path = r.URL.Path
	ctx.kv = make(map[interface{}]interface{})

	if app.timeout <= 0 {
		ctx.ctx, ctx.cancelCtx = context.WithCancel(r.Context())
	} else {
		ctx.ctx, ctx.cancelCtx = context.WithTimeout(r.Context(), app.timeout)
	}

	if app.withContext != nil {
		ctx._ctx = app.withContext(r.WithContext(ctx.ctx))
	} else {
		ctx._ctx = ctx.ctx
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
	return ctx._ctx.Value(key)
}

// Cancel cancel the ctx and all it' children context.
// The ctx' process will ended too.
func (ctx *Context) Cancel() {
	ctx.ended.setTrue() // end the middleware process
	ctx.cleanAfterHooks()
	ctx.cancelCtx()
}

// WithCancel returns a copy of the ctx with a new Done channel.
// The returned context's Done channel is closed when the returned cancel function is called or when the parent context's Done channel is closed, whichever happens first.
func (ctx *Context) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx._ctx)
}

// WithDeadline returns a copy of the ctx with the deadline adjusted to be no later than d.
func (ctx *Context) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx._ctx, deadline)
}

// WithTimeout returns WithDeadline(time.Now().Add(timeout)).
func (ctx *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx._ctx, timeout)
}

// WithValue returns a copy of the ctx in which the value associated with key is val.
func (ctx *Context) WithValue(key, val interface{}) context.Context {
	return context.WithValue(ctx._ctx, key, val)
}

// WithContext sets the context to underlying gear.Context.
// The context must be a children or a grandchild of gear.Context.
//
//  ctx.WithContext(ctx.WithValue("key", "value"))
//  // ctx.Value("key") == "value"
//
// a Tracing middleware:
//
//  func Tracing(ctx *Context) error {
//  	sp := opentracing.StartSpan(ctx.Path)
//  	ctx.WithContext(opentracing.ContextWithSpan(ctx, sp))
//  	ctx.OnEnd(sp.Finish)
//  }
//
func (ctx *Context) WithContext(c context.Context) {
	ctx._ctx = c
}

// Timing runs fn with the given time limit. If a call runs for longer than its time limit,
// it will return context.DeadlineExceeded as error, otherwise return fn's result.
func (ctx *Context) Timing(dt time.Duration, fn func(context.Context) interface{}) (res interface{}, err error) {
	ct, cancel := ctx.WithTimeout(dt)
	defer cancel()

	ch := make(chan interface{})
	go func() {
		// recover the fn call
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("Timing panic: %#v", e)
			}
			close(ch)
		}()
		ch <- fn(ct)
	}()
	select {
	case <-ct.Done():
		err = ct.Err()
	case res = <-ch:
	}
	return
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
// Then we can use ctx.Any(key) to retrieve the value from ctx.
func (ctx *Context) SetAny(key, val interface{}) {
	ctx.kv[key] = val
}

// Setting returns App's settings by key
//
//  fmt.Println(ctx.Setting(gear.SetEnv).(string) == "development")
//  app.Set(gear.SetEnv, "production")
//  fmt.Println(ctx.Setting(gear.SetEnv).(string) == "production")
//
func (ctx *Context) Setting(key interface{}) interface{} {
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

// AcceptType returns the most preferred content type from the HTTP Accept header.
// If nothing accepted, then empty string is returned.
func (ctx *Context) AcceptType(preferred ...string) string {
	return negotiator.New(ctx.Req.Header).Type(preferred...)
}

// AcceptLanguage returns the most preferred language from the HTTP Accept-Language header.
// If nothing accepted, then empty string is returned.
func (ctx *Context) AcceptLanguage(preferred ...string) string {
	return negotiator.New(ctx.Req.Header).Language(preferred...)
}

// AcceptEncoding returns the most preferred encoding from the HTTP Accept-Encoding header.
// If nothing accepted, then empty string is returned.
func (ctx *Context) AcceptEncoding(preferred ...string) string {
	return negotiator.New(ctx.Req.Header).Encoding(preferred...)
}

// AcceptCharset returns the most preferred charset from the HTTP Accept-Charset header.
// If nothing accepted, then empty string is returned.
func (ctx *Context) AcceptCharset(preferred ...string) string {
	return negotiator.New(ctx.Req.Header).Charset(preferred...)
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

// QueryAll returns all query params for the provided name.
func (ctx *Context) QueryAll(name string) []string {
	if ctx.query == nil {
		ctx.query = ctx.Req.URL.Query()
	}
	return ctx.query[name]
}

// ParseBody parses request content with BodyParser, DefaultBodyParser support JSON and XML.
// stores the result in the value pointed to by BodyTemplate body, and validate it.
//
// Defaine a BodyTemplate type in some API:
//  type jsonBodyTemplate struct {
//  	ID   string `json:"id"`
//  	Pass string `json:"pass"`
//  }
//
//  func (b *jsonBodyTemplate) Validate() error {
//  	if len(b.ID) < 3 || len(b.Pass) < 6 {
//  		return &Error{Code: 400, Msg: "invalid id or pass"}
//  	}
//  	return nil
//  }
//
// Use it in middleware:
//  body := &jsonBodyTemplate{}
//  if err := ctx.ParseBody(body) {
//  	return err
//  }
//
func (ctx *Context) ParseBody(body BodyTemplate) error {
	if ctx.app.bodyParser == nil {
		return NewAppError("bodyParser not registered")
	}
	if ctx.Req.Body == nil {
		return NewAppError("missing request body")
	}

	var err error
	var buf []byte
	var mediaType string
	var params map[string]string
	if mediaType = ctx.Get(HeaderContentType); mediaType == "" {
		// RFC 2616, section 7.2.1 - empty type SHOULD be treated as application/octet-stream
		mediaType = MIMEOctetStream
	}
	if mediaType, params, err = mime.ParseMediaType(mediaType); err != nil {
		return &Error{Code: http.StatusUnsupportedMediaType, Msg: err.Error()}
	}

	reader := http.MaxBytesReader(ctx.Res, ctx.Req.Body, ctx.app.bodyParser.MaxBytes())
	if buf, err = ioutil.ReadAll(reader); err != nil {
		// err may not be 413 Request entity too large, just make it to 413
		return &Error{Code: http.StatusRequestEntityTooLarge, Msg: err.Error()}
	}
	if err = ctx.app.bodyParser.Parse(buf, body, mediaType, params["charset"]); err != nil {
		return err
	}
	return body.Validate()
}

// Get retrieves data from the request Header.
func (ctx *Context) Get(key string) string {
	return ctx.Req.Header.Get(key)
}

// Set saves data to the response Header.
func (ctx *Context) Set(key, value string) {
	ctx.Res.Set(key, value)
}

// Status set a status code (optional) to the response, returns the new status code.
func (ctx *Context) Status(code ...int) int {
	if len(code) > 0 && IsStatusCode(code[0]) {
		ctx.Res.status = code[0]
	}
	return ctx.Res.status
}

// Type set a content type (optional) to the response, returns the new content type.
func (ctx *Context) Type(str ...string) string {
	if len(str) > 0 {
		ctx.Res.Set(HeaderContentType, str[0])
	}
	return ctx.Res.Get(HeaderContentType)
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
	if ctx.ended.swapTrue() {
		ctx.Status(code)
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
func (ctx *Context) Attachment(name string, modtime time.Time, content io.ReadSeeker, inline ...bool) (err error) {
	if ctx.ended.swapTrue() {
		dispositionType := "attachment"
		if len(inline) > 0 && inline[0] {
			dispositionType = "inline"
		}
		ctx.Set(HeaderContentDisposition, fmt.Sprintf("%s; filename=%s", dispositionType, name))
		http.ServeContent(ctx.Res, ctx.Req, name, modtime, content)
	}
	return
}

// Redirect redirects the request with status code 302.
// You can use other status code with ctx.Status method, It is a wrap of http.Redirect.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Redirect(url string) (err error) {
	if ctx.ended.swapTrue() {
		if !isRedirectStatus(ctx.Res.status) {
			ctx.Res.status = http.StatusFound
		}
		http.Redirect(ctx.Res, ctx.Req, url, ctx.Res.status)
	}
	return
}

// Error send a error to response.
// It will not reset response headers and not use app.OnError hook
// It will end the ctx. The middlewares after current middleware and "after hooks" will not run.
// "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Error(e error) error {
	ctx.cleanAfterHooks() // clear afterHooks when any error
	ctx.Res.ResetHeader()
	err := ParseError(e, ctx.Res.status)
	if err == nil {
		err = &Error{Code: http.StatusInternalServerError, Msg: NewAppError("nil error").Error()}
	}
	if ctx.app.onerror != nil {
		ctx.app.onerror(ctx, err)
	}
	//  try to respond error if `OnError` does't do it.
	ctx.respondError(err)
	return nil
}

// ErrorStatus send a error by status code to response. The status should be 4xx or 5xx code.
// It will not reset response headers and not use app.OnError hook
// It will end the ctx. The middlewares after current middleware and "after hooks" will not run.
// "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) ErrorStatus(status int) (err error) {
	if status >= 400 && status < 600 {
		if msg := http.StatusText(status); msg != "" {
			return ctx.Error(&Error{Code: status, Msg: msg})
		}
	}
	return &Error{Code: http.StatusInternalServerError, Msg: NewAppError("invalid status").Error()}
}

// End end the ctx with bytes and status code optionally.
// After it's called, the rest of middleware handles will not run.
// But "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) End(code int, buf ...[]byte) (err error) {
	if ctx.ended.swapTrue() {
		var body []byte
		if len(buf) > 0 {
			body = buf[0]
		}
		err = ctx.Res.respond(code, body)
	}
	return
}

// After add a "after hook" to the ctx that will run after middleware process,
// but before Response.WriteHeader.
func (ctx *Context) After(hook func()) {
	if ctx.ended.isTrue() { // should not add afterHooks if ctx.ended
		panic(NewAppError(`can't add "after hook" after context ended`))
	}
	ctx.afterHooks = append(ctx.afterHooks, hook)
}

func (ctx *Context) cleanAfterHooks() {
	ctx.afterHooks = nil
}

// OnEnd add a "end hook" to the ctx that will run after Response.WriteHeader.
func (ctx *Context) OnEnd(hook func()) {
	if ctx.ended.isTrue() { // should not add endHooks if ctx.ended
		panic(NewAppError(`can't add "end hook" after context ended`))
	}
	ctx.endHooks = append(ctx.endHooks, hook)
}

func (ctx *Context) respondError(err *Error) {
	if !ctx.Res.wroteHeader.isTrue() {
		if err.Code < 400 {
			err.Code = 500
		}
		ctx.app.Error(err)
		ctx.Set(HeaderContentType, MIMETextPlainCharsetUTF8)
		ctx.Set(HeaderXContentTypeOptions, "nosniff")
		ctx.Res.respond(err.Code, []byte(err.Msg))
	}
}

func (ctx *Context) handleCompress() (cw *compressWriter) {
	if ctx.app.compress != nil && ctx.Method != http.MethodHead && ctx.Method != http.MethodOptions {
		if cw = newCompress(ctx.Res, ctx.app.compress, ctx.AcceptEncoding("gzip", "deflate")); cw != nil {
			ctx.Res.rw = cw // override with http.ResponseWriter wrapper.
		}
	}
	return
}
