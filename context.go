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
	"strings"
	"time"

	"github.com/go-http-utils/cookie"
	"github.com/go-http-utils/negotiator"
)

type contextKey int

const (
	isInheritedContext contextKey = iota
	isGearContext
	paramsKey
	routerNodeKey
	routerRootKey
)

// Any interface is used by ctx.Any.
type Any interface {
	New(ctx *Context) (interface{}, error)
}

// BodyTemplate interface is used by ctx.ParseBody.
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

	Host    string
	Method  string
	Path    string
	StartAt time.Time

	query     url.Values
	ctx       context.Context
	cancelCtx context.CancelFunc
	done      <-chan struct{}
	kv        map[interface{}]interface{}
}

// NewContext creates an instance of Context. Export for testing middleware.
func NewContext(app *App, w http.ResponseWriter, r *http.Request) *Context {
	ctx := Context{
		app: app,
		Res: &Response{w: w, rw: w, handlerHeader: w.Header()},

		Host:    r.Host,
		Method:  r.Method,
		Path:    r.URL.Path,
		StartAt: time.Now().UTC(),

		Cookies: cookie.New(w, r, app.keys...),
		kv:      make(map[interface{}]interface{}),
	}

	if app.serverName != "" {
		ctx.SetHeader(HeaderServer, app.serverName)
	}

	if app.timeout <= 0 {
		ctx.ctx, ctx.cancelCtx = context.WithCancel(r.Context())
	} else {
		ctx.ctx, ctx.cancelCtx = context.WithTimeout(r.Context(), app.timeout)
	}

	ctx.ctx = context.WithValue(ctx.ctx, isInheritedContext, struct{}{})
	ctx.Req = r.WithContext(ctx.ctx)
	if app.withContext != nil {
		ctx.WithContext(app.withContext(ctx.Req))
	}

	ctx.done = ctx.ctx.Done()
	return &ctx
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
	if key == isGearContext {
		return struct{}{}
	}

	return ctx.ctx.Value(key)
}

// Cancel cancel the ctx and all it' children context.
// The ctx' process will ended too.
func (ctx *Context) Cancel() {
	ctx.Res.ended.setTrue() // end the middleware process
	ctx.Res.afterHooks = nil
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

// Context returns the underlying context of gear.Context
func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

// WithContext sets the context to underlying gear.Context.
// The context must be a children or a grandchild of gear.Context.
//
//  ctx.WithContext(ctx.WithValue("key", "value"))
//  // ctx.Value("key") == "value"
//
func (ctx *Context) WithContext(c context.Context) {
	if c.Value(isGearContext) != nil {
		panic(Err.WithMsg("should not use *gear.Context as parent context, please use ctx.Context()"))
	}
	if c.Value(isInheritedContext) == nil {
		panic(Err.WithMsg("the context is not created from ctx.Context()"))
	}

	ctx.Req = ctx.Req.WithContext(c)
	ctx.ctx = c
}

// LogErr writes error to underlayer logging system through app.Error.
func (ctx *Context) LogErr(err error) {
	ctx.app.Error(err)
}

// Timing runs fn with the given time limit. If a call runs for longer than its time limit or panic,
// it will return context.DeadlineExceeded error or panic error.
func (ctx *Context) Timing(dt time.Duration, fn func(context.Context)) (err error) {
	ct, cancel := ctx.WithTimeout(dt)
	defer cancel()

	ch := make(chan error, 1) // not block tryRunTiming
	go tryRunTiming(ct, fn, ch)

	select {
	case <-ct.Done():
		err = ct.Err()
	case err = <-ch:
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
		switch v := any.(type) {
		case Any:
			if val, err = v.New(ctx); err == nil {
				ctx.kv[any] = val
			}
		default:
			return nil, Err.WithMsg("non-existent key")
		}
	}
	return
}

// MustAny returns the value on this ctx by key. It is a sugar for ctx.Any,
// If some error occurred, it will panic.
func (ctx *Context) MustAny(any interface{}) interface{} {
	val, err := ctx.Any(any)
	if err != nil {
		panic(err)
	}
	return val
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
// The trustedProxy argument will be removed in v2.
func (ctx *Context) IP(trustedProxy ...bool) net.IP {
	trusted := ctx.Setting(SetTrustedProxy).(bool)
	if len(trustedProxy) > 0 {
		trusted = trustedProxy[0]
	}

	var ip string
	if trusted {
		ip = ctx.Req.Header.Get(HeaderXRealIP)

		if ip == "" {
			ip = ctx.Req.Header.Get(HeaderXForwardedFor)
			if i := strings.IndexByte(ip, ','); i > 0 {
				ip = ip[0:i]
			}
		}
	}
	if ip == "" {
		ra := ctx.Req.RemoteAddr
		ip, _, _ = net.SplitHostPort(ra)
	}

	return net.ParseIP(ip)
}

// Protocol -  Please use ctx.Scheme instead. This method will be changed in v2.
func (ctx *Context) Protocol(trustedProxy ...bool) string {
	return ctx.Scheme(trustedProxy...)
}

// Scheme returns the scheme ("http", "https", "ws", "wss") that a client used to connect to your proxy or load balancer.
// The trustedProxy argument will be removed in v2.
func (ctx *Context) Scheme(trustedProxy ...bool) string {
	trusted := ctx.Setting(SetTrustedProxy).(bool)
	if len(trustedProxy) > 0 {
		trusted = trustedProxy[0]
	}

	var s string
	if trusted {
		if s = ctx.GetHeader(HeaderXRealScheme); s == "" {
			if s = ctx.GetHeader(HeaderXForwardedProto); s == "" {
				s = ctx.GetHeader(HeaderXForwardedScheme)
			}
		}
		s = strings.ToLower(s)
	}

	if s == "" {
		if ctx.Req.TLS != nil {
			s = "https"
		} else {
			s = "http"
		}
	}
	return s
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

// ParseBody parses request content with BodyParser, stores the result in the value
// pointed to by BodyTemplate body, and validate it.
// DefaultBodyParser support JSON, Form and XML.
//
// Define a BodyTemplate type in some API:
//  type jsonBodyTemplate struct {
//  	ID   string `json:"id" form:"id"`
//  	Pass string `json:"pass" form:"pass"`
//  }
//
//  func (b *jsonBodyTemplate) Validate() error {
//  	if len(b.ID) < 3 || len(b.Pass) < 6 {
//  		return ErrBadRequest.WithMsg("invalid id or pass")
//  	}
//  	return nil
//  }
//
// Use it in middleware:
//  body := jsonBodyTemplate{}
//  if err := ctx.ParseBody(&body); err != nil {
//  	return err
//  }
//
func (ctx *Context) ParseBody(body BodyTemplate) error {
	if ctx.app.bodyParser == nil {
		return Err.WithMsg("bodyParser not registered")
	}
	if ctx.Req.Body == nil {
		return Err.WithMsg("missing request body")
	}

	var err error
	var buf []byte
	var mediaType string
	var encoding string
	var params map[string]string

	if mediaType = ctx.GetHeader(HeaderContentType); mediaType == "" {
		// RFC 2616, section 7.2.1 - empty type SHOULD be treated as application/octet-stream
		mediaType = MIMEOctetStream
	}

	ctx.SetAny("GEAR_REQUEST_CONTENT_TYPE", mediaType)
	if mediaType, params, err = mime.ParseMediaType(mediaType); err != nil {
		return ErrUnsupportedMediaType.From(err)
	}

	b := ctx.Req.Body
	if encoding = ctx.GetHeader(HeaderContentEncoding); encoding != "" {
		if b, err = Decompress(encoding, ctx.Req.Body); err != nil {
			return err
		}
	}

	reader := http.MaxBytesReader(ctx.Res, b, ctx.app.bodyParser.MaxBytes())
	defer reader.Close()

	if buf, err = ioutil.ReadAll(reader); err != nil {
		// err may not be 413 Request entity too large, just make it to 413
		return ErrRequestEntityTooLarge.From(err)
	}

	ctx.SetAny("GEAR_REQUEST_BODY", buf[:])
	if err = ctx.app.bodyParser.Parse(buf, body, mediaType, params["charset"]); err != nil {
		return ErrBadRequest.From(err)
	}
	return body.Validate()
}

// ParseURL parses router params (like ctx.Param) and queries (like ctx.Query) in request URL,
// stores the result in the struct object pointed to by BodyTemplate body, and validate it.
//
// Define a BodyTemplate type in some API:
//  type taskTemplate struct {
//  	ID      bson.ObjectId `json:"_taskID" param:"_taskID"` // router.Get("/tasks/:_taskID", APIhandler)
//  	StartAt time.Time     `json:"startAt" query:"startAt"` // GET /tasks/50c32afae8cf1439d35a87e6?startAt=2017-05-03T10:06:45.319Z
//  }
//
//  func (b *taskTemplate) Validate() error {
//  	if !b.ID.Valid() {
//  		return gear.ErrBadRequest.WithMsg("invalid task id")
//  	}
//  	if b.StartAt.IsZero() {
//  		return gear.ErrBadRequest.WithMsg("invalid task start time")
//  	}
//  	return nil
//  }
//
// Use it in APIhandler:
//  body := taskTemplate{}
//  if err := ctx.ParseURL(&body); err != nil {
//  	return err
//  }
//
func (ctx *Context) ParseURL(body BodyTemplate) error {
	if ctx.app.urlParser == nil {
		return Err.WithMsg("urlParser not registered")
	}

	if err := ctx.app.urlParser.Parse(ctx.Req.URL.Query(), body, "query"); err != nil {
		return ErrBadRequest.From(err)
	}

	if res, _ := ctx.Any(paramsKey); res != nil {
		if params, _ := res.(map[string]string); len(params) > 0 {
			paramValues := make(map[string][]string)
			for k, v := range params {
				paramValues[k] = []string{v}
			}

			if err := ctx.app.urlParser.Parse(paramValues, body, "param"); err != nil {
				return ErrBadRequest.From(err)
			}
		}
	}

	return body.Validate()
}

// Get - Please use ctx.GetHeader instead. This method will be changed in v2.
func (ctx *Context) Get(key string) string {
	return ctx.GetHeader(key)
}

// Set - Please use ctx.SetHeader instead. This method will be changed in v2.
func (ctx *Context) Set(key, value string) {
	ctx.SetHeader(key, value)
}

// GetHeader returns the first value associated with the given key from the request Header.
func (ctx *Context) GetHeader(key string) string {
	switch key {
	case "Referer", "referer", "Referrer", "referrer":
		if val := ctx.Req.Header.Get("Referer"); val != "" {
			return val
		}
		return ctx.Req.Header.Get("Referrer")
	default:
		return ctx.Req.Header.Get(key)
	}
}

// GetHeaders returns all values associated with the given key from the request Header.
func (ctx *Context) GetHeaders(key string) []string {
	switch key {
	case "Referer", "referer", "Referrer", "referrer":
		if vals := getHeaderValues(ctx.Req.Header, "Referer"); len(vals) > 0 {
			return vals
		}
		return getHeaderValues(ctx.Req.Header, "Referrer")
	default:
		return getHeaderValues(ctx.Req.Header, key)
	}
}

// SetHeader saves data to the response Header.
func (ctx *Context) SetHeader(key, value string) {
	ctx.Res.Set(key, value)
}

// Status set a status code to the response, ctx.Res.Status() returns the status code.
func (ctx *Context) Status(code int) {
	ctx.Res.status = code
}

// Type set a content type to the response, ctx.Res.Type() returns the content type.
func (ctx *Context) Type(str string) {
	ctx.Res.Set(HeaderContentType, str)
}

// HTML set an Html body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
func (ctx *Context) HTML(code int, str string) error {
	ctx.Type(MIMETextHTMLCharsetUTF8)
	return ctx.End(code, []byte(str))
}

// JSON set a JSON body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
func (ctx *Context) JSON(code int, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ctx.JSONBlob(code, buf)
}

// JSONBlob set a JSON blob body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
func (ctx *Context) JSONBlob(code int, buf []byte) error {
	ctx.Type(MIMEApplicationJSONCharsetUTF8)
	return ctx.End(code, buf)
}

// JSONP sends a JSONP response with status code. It uses `callback` to construct the JSONP payload.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
func (ctx *Context) JSONP(code int, callback string, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ctx.JSONPBlob(code, callback, buf)
}

// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
// to construct the JSONP payload.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
func (ctx *Context) JSONPBlob(code int, callback string, buf []byte) error {
	ctx.Type(MIMEApplicationJavaScriptCharsetUTF8)
	ctx.SetHeader(HeaderXContentTypeOptions, "nosniff")
	// the /**/ is a specific security mitigation for "Rosetta Flash JSONP abuse"
	// @see http://miki.it/blog/2014/7/8/abusing-jsonp-with-rosetta-flash/
	// the typeof check is just to reduce client error noise
	b := []byte(fmt.Sprintf(`/**/ typeof %s === "function" && %s(`, callback, callback))
	b = append(b, buf...)
	return ctx.End(code, append(b, ')', ';'))
}

// XML set an XML body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
func (ctx *Context) XML(code int, val interface{}) error {
	buf, err := xml.Marshal(val)
	if err != nil {
		return err
	}
	return ctx.XMLBlob(code, buf)
}

// XMLBlob set a XML blob body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
func (ctx *Context) XMLBlob(code int, buf []byte) error {
	ctx.Type(MIMEApplicationXMLCharsetUTF8)
	return ctx.End(code, buf)
}

// Send handle code and data with Sender interface.
// Sender can be registered using `app.Set(gear.SetSender, someSender)`.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// You can define a custom send function like this:
//
//  type mySenderT struct{}
//
//  func (s *mySenderT) Send(ctx *Context, code int, data interface{}) error {
// 	 switch v := data.(type) {
// 	 case []byte:
//  		ctx.Type(MIMETextPlainCharsetUTF8)
//  		return ctx.End(code, v)
//  	case string:
//  		return ctx.HTML(code, v)
//  	case error:
//  		return ctx.Error(v)
//  	default:
//  		return ctx.JSON(code, data)
//  	}
//  }
//
//  app.Set(gear.SetSender, &mySenderT{})
//  app.Use(func(ctx *Context) error {
//  	switch ctx.Path {
//  	case "/text":
//  		return ctx.Send(http.StatusOK, []byte("Hello, Gear!"))
//  	case "/html":
//  		return ctx.Send(http.StatusOK, "<h1>Hello, Gear!</h1>")
//  	case "/error":
//  		return ctx.Send(http.StatusOK, Err.WithMsg("some error"))
//  	default:
//  		return ctx.Send(http.StatusOK, map[string]string{"value": "Hello, Gear!"})
//  	}
//  })
func (ctx *Context) Send(code int, data interface{}) (err error) {
	if ctx.app.sender == nil {
		return Err.WithMsg("sender not registered")
	}
	return ctx.app.sender.Send(ctx, code, data)
}

// Render renders a template with data and sends a text/html response with status
// code. Templates can be registered using `app.Set(gear.SetRenderer, someRenderer)`.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
func (ctx *Context) Render(code int, name string, data interface{}) (err error) {
	if ctx.app.renderer == nil {
		return Err.WithMsg("renderer not registered")
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
func (ctx *Context) Stream(code int, contentType string, r io.Reader) (err error) {
	if ctx.Res.ended.swapTrue() {
		ctx.Status(code)
		ctx.Type(contentType)
		_, err = io.Copy(ctx.Res, r)
	} else {
		err = ErrInternalServerError.WithMsg("request ended before ctx.Stream")
	}
	return
}

// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
// client to save the file. If inline is true, the attachment will sends as inline,
// opening the file in the browser.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
func (ctx *Context) Attachment(name string, modtime time.Time, content io.ReadSeeker, inline ...bool) (err error) {
	if ctx.Res.ended.swapTrue() {
		dispositionType := "attachment"
		if len(inline) > 0 && inline[0] {
			dispositionType = "inline"
		}
		ctx.SetHeader(HeaderContentDisposition, ContentDisposition(name, dispositionType))
		http.ServeContent(ctx.Res, ctx.Req, name, modtime, content)
	} else {
		err = ErrInternalServerError.WithMsg("request ended before ctx.Attachment")
	}
	return
}

// Redirect redirects the request with status code 302.
// You can use other status code with ctx.Status method, It is a wrap of http.Redirect.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
func (ctx *Context) Redirect(url string) (err error) {
	if ctx.Res.ended.swapTrue() {
		if !isRedirectStatus(ctx.Res.status) {
			ctx.Res.status = http.StatusFound
		}
		http.Redirect(ctx.Res, ctx.Req, url, ctx.Res.status)
	} else {
		err = ErrInternalServerError.WithMsg("request ended before ctx.Redirect")
	}
	return
}

// OkHTML is a wrap of ctx.HTML with http.StatusOK
func (ctx *Context) OkHTML(str string) error {
	return ctx.HTML(http.StatusOK, str)
}

// OkJSON is a wrap of ctx.JSON with http.StatusOK
//
//  ctx.OkJSON(struct{}{})
func (ctx *Context) OkJSON(val interface{}) error {
	return ctx.JSON(http.StatusOK, val)
}

// OkXML is a wrap of ctx.XML with http.StatusOK
func (ctx *Context) OkXML(val interface{}) error {
	return ctx.XML(http.StatusOK, val)
}

// OkSend is a wrap of ctx.Send with http.StatusOK
func (ctx *Context) OkSend(val interface{}) error {
	return ctx.Send(http.StatusOK, val)
}

// OkRender is a wrap of ctx.Render with http.StatusOK
func (ctx *Context) OkRender(name string, val interface{}) error {
	return ctx.Render(http.StatusOK, name, val)
}

// OkStream is a wrap of ctx.Stream with http.StatusOK
func (ctx *Context) OkStream(contentType string, r io.Reader) error {
	return ctx.Stream(http.StatusOK, contentType, r)
}

// Error send a error with application/json type to response.
// It will not trigger gear.SetOnError hook.
// It will end the ctx. The middlewares after current middleware and "after hooks" will not run,
// but "end hooks" will run normally.
func (ctx *Context) Error(e error) error {
	ctx.Res.afterHooks = nil // clear afterHooks when any error
	ctx.Res.ResetHeader()
	err := ParseError(e, ctx.Res.status)
	if err == nil {
		err = ErrInternalServerError.WithMsg("nil error")
	}
	ctx.respondError(err)
	return nil
}

// ErrorStatus send a error by status code to response.
// It is sugar of ctx.Error
func (ctx *Context) ErrorStatus(status int) error {
	if status >= 400 && IsStatusCode(status) {
		return ctx.Error(ErrByStatus(status))
	}
	return ErrInternalServerError.WithMsg("invalid error status")
}

// End end the ctx with bytes and status code optionally.
// After it's called, the rest of middleware handles will not run.
// But "after hooks" and "end hooks" will run normally.
func (ctx *Context) End(code int, buf ...[]byte) (err error) {
	if ctx.Res.ended.swapTrue() {
		var body []byte
		if len(buf) > 0 {
			body = buf[0]
		}
		err = ctx.Res.respond(code, body)
	} else {
		err = ErrInternalServerError.WithMsg("request ended before ctx.End")
	}
	return
}

// After add a "after hook" to the ctx that will run after middleware process,
// but before Response.WriteHeader. So it will block response writing.
func (ctx *Context) After(hook func()) {
	if ctx.Res.wroteHeader.isTrue() {
		panic(Err.WithMsg(`can't add "after hook" after header wrote`))
	}
	ctx.Res.afterHooks = append(ctx.Res.afterHooks, hook)
}

// OnEnd add a "end hook" to the ctx that will run after Response.WriteHeader.
// They run in a goroutine and will not block response.
// Take care that http.ResponseWriter and http.Request maybe reset for reusing.
// Issue https://github.com/teambition/gear/issues/24
func (ctx *Context) OnEnd(hook func()) {
	if ctx.Res.wroteHeader.isTrue() {
		panic(Err.WithMsg(`can't add "end hook" after header wrote`))
	}
	ctx.Res.endHooks = append(ctx.Res.endHooks, hook)
}

func (ctx *Context) respondError(err HTTPError) {
	if !ctx.Res.wroteHeader.isTrue() {
		code := err.Status()
		// we don't need to logging 501, 4xx errors
		if code == 500 || code > 501 || code < 400 {
			ctx.app.Error(err)
		}
		// try to render error as json
		ctx.SetHeader(HeaderContentType, MIMEApplicationJSONCharsetUTF8)
		ctx.SetHeader(HeaderXContentTypeOptions, "nosniff")

		buf, _ := json.Marshal(err)
		ctx.Res.respond(code, buf)
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

func catchTiming(ch chan error) {
	defer close(ch)
	// recover the fn call
	if e := recover(); e != nil {
		ch <- ErrInternalServerError.WithMsgf("Timing panic: %#v", e)
	}
}

func tryRunTiming(ct context.Context, fn func(context.Context), ch chan error) {
	defer catchTiming(ch)
	fn(ct)
}
