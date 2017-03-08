package cors

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/gear"
)

// Options is cors middleware options.
type Options struct {
	// AllowOrigins defines the origins which will be allowed to access
	// the resource. Default value is []string{"*"} .
	AllowOrigins []string
	// AllowMethods defines the methods which will be allowed to access
	// the resource. It is used in handling the preflighted requests.
	// Default value is []string{"GET", "HEAD", "PUT", "POST", "DELETE", "PATCH"} .
	AllowMethods []string
	// AllowOriginsValidator validates the request Origin by validator
	// function.The validator function accpects an `*gear.Context` and returns the
	// Access-Control-Allow-Origin value. If the validator is set, then
	// AllowMethods will be ignored.
	AllowOriginsValidator func(origin string, ctx *gear.Context) string
	// AllowHeaders defines the headers which will be allowed in the actual
	// request, It is used in handling the preflighted requests.
	AllowHeaders []string
	// ExposeHeaders defines the allowed headers that client could send when
	// accessing the resource.
	ExposeHeaders []string
	// MaxAge defines the max age that the preflighted requests can be cached.
	MaxAge time.Duration
	// Credentials defines whether or not the response to the request
	// can be exposed.
	Credentials bool
}

var (
	defaultAllowOrigins = []string{"*"}
	defaultAllowMethods = []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPut,
		http.MethodPost,
		http.MethodDelete,
		http.MethodPatch,
	}
)

// New creates a middleware to provide CORS support for gear.
func New(options ...Options) gear.Middleware {
	opts := Options{}
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.AllowOrigins == nil {
		opts.AllowOrigins = defaultAllowOrigins
	}
	if opts.AllowMethods == nil {
		opts.AllowMethods = defaultAllowMethods
	}
	if opts.AllowOriginsValidator == nil {
		opts.AllowOriginsValidator = func(origin string, _ *gear.Context) (allowOrigin string) {
			for _, o := range opts.AllowOrigins {
				if o == origin || o == "*" {
					allowOrigin = origin
					break
				}
			}
			return
		}
	}

	return func(ctx *gear.Context) (err error) {
		// Always set Vary, see https://github.com/rs/cors/issues/10
		ctx.Res.Vary(gear.HeaderOrigin)

		origin := ctx.Get(gear.HeaderOrigin)
		// not a CORS request.
		if origin == "" {
			return
		}

		allowOrigin := opts.AllowOriginsValidator(origin, ctx)
		// If the request Origin header is not allowed. Just terminate the following steps.
		if allowOrigin == "" {
			return ctx.Error(&gear.Error{Code: http.StatusForbidden,
				Msg: fmt.Sprintf("Origin: %v is not allowed", origin)})
		}
		if opts.Credentials {
			// when responding to a credentialed request, server must specify a
			// domain, and cannot use wild carding.
			// See *important note* in https://developer.mozilla.org/en-US/docs/Web/HTTP/Access_control_CORS#Requests_with_credentials .
			ctx.Set(gear.HeaderAccessControlAllowCredentials, "true")
		}
		ctx.Set(gear.HeaderAccessControlAllowOrigin, allowOrigin)

		// Handle preflighted requests (https://developer.mozilla.org/en-US/docs/Web/HTTP/Access_control_CORS#Preflighted_requests) .
		if ctx.Method == http.MethodOptions {
			ctx.Res.Vary(gear.HeaderAccessControlRequestMethod)
			ctx.Res.Vary(gear.HeaderAccessControlRequestHeaders)

			requestMethod := ctx.Get(gear.HeaderAccessControlRequestMethod)
			// If there is no "Access-Control-Request-Method" request header. We just
			// treat this request as an invalid preflighted request, so terminate the
			// following steps.
			if requestMethod == "" {
				ctx.Res.Del(gear.HeaderAccessControlAllowOrigin)
				ctx.Res.Del(gear.HeaderAccessControlAllowCredentials)
				return ctx.Error(&gear.Error{Code: http.StatusForbidden,
					Msg: "invalid preflighted request, missing Access-Control-Request-Method header"})
			}
			if len(opts.AllowMethods) > 0 {
				ctx.Set(gear.HeaderAccessControlAllowMethods, strings.Join(opts.AllowMethods, ", "))
			}

			var allowHeaders string
			if len(opts.AllowHeaders) > 0 {
				allowHeaders = strings.Join(opts.AllowHeaders, ", ")
			} else {
				allowHeaders = ctx.Get(gear.HeaderAccessControlRequestHeaders)
			}
			if allowHeaders != "" {
				ctx.Set(gear.HeaderAccessControlAllowHeaders, allowHeaders)
			}

			if opts.MaxAge > 0 {
				ctx.Set(gear.HeaderAccessControlMaxAge, strconv.Itoa(int(opts.MaxAge.Seconds())))
			}
			return ctx.End(http.StatusNoContent)
		}

		if len(opts.ExposeHeaders) > 0 {
			ctx.Set(gear.HeaderAccessControlExposeHeaders, strings.Join(opts.ExposeHeaders, ", "))
		}
		return
	}
}
