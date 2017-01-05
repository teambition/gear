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
	AllowOriginsValidator func(*gear.Context) string
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
func New(opts Options) gear.Middleware {
	if opts.AllowOrigins == nil {
		opts.AllowOrigins = defaultAllowOrigins
	}

	if opts.AllowMethods == nil {
		opts.AllowMethods = defaultAllowMethods
	}

	return func(ctx *gear.Context) (err error) {
		origin := ctx.Get(gear.HeaderOrigin)

		// not a CORS request.
		if origin == "" {
			return
		}

		allowOrigin := ""

		if opts.AllowOriginsValidator != nil {
			allowOrigin = opts.AllowOriginsValidator(ctx)
		} else {
			for _, o := range opts.AllowOrigins {
				if o == origin || o == "*" {
					allowOrigin = o
					break
				}
			}
		}

		// If the request Origin header is not allowed. Just terminate
		// the following steps.
		if allowOrigin == "" {
			return ctx.End(http.StatusForbidden, []byte(fmt.Sprintf("Origin: %v is not allowed", origin)))
		}

		// Handle preflighted requests (https://developer.mozilla.org/en-US/docs/Web/HTTP/Access_control_CORS#Preflighted_requests) .
		if ctx.Method == http.MethodOptions {
			requestMethod := ctx.Get(gear.HeaderAccessControlRequestMethod)

			// If there is no "Access-Control-Request-Method" request header. We just
			// treat this request as an invalid preflighted request, so terminate the
			// following steps.
			if requestMethod == "" {
				return ctx.End(http.StatusForbidden, []byte(fmt.Sprint("Invalid preflighted request, missing Access-Control-Request-Method header")))
			}

			ctx.Set(gear.HeaderAccessControlAllowOrigin, allowOrigin)

			configureCredentials(ctx, opts, allowOrigin)

			if len(opts.AllowMethods) > 0 {
				ctx.Set(gear.HeaderAccessControlAllowMethods, joinWithComma(opts.AllowMethods))
			}

			var allowHeaders string

			if len(opts.AllowHeaders) > 0 {
				allowHeaders = joinWithComma(opts.AllowHeaders)
			} else {
				allowHeaders = ctx.Get(gear.HeaderAccessControlRequestHeaders)
			}

			if allowHeaders != "" {
				ctx.Set(gear.HeaderAccessControlAllowHeaders, allowHeaders)
			}

			if opts.MaxAge > 0 {
				ctx.Set(gear.HeaderAccessControlMaxAge, strconv.Itoa(int(opts.MaxAge.Seconds())))
			}

			ctx.Status(http.StatusNoContent)
		} else {
			ctx.Set(gear.HeaderAccessControlAllowOrigin, allowOrigin)

			configureCredentials(ctx, opts, allowOrigin)

			if len(opts.ExposeHeaders) > 0 {
				ctx.Set(gear.HeaderAccessControlExposeHeaders, joinWithComma(opts.ExposeHeaders))
			}
		}

		return
	}
}

func joinWithComma(s []string) string {
	return strings.Join(s, ",")
}

func configureCredentials(ctx *gear.Context, opts Options, origin string) {
	if opts.Credentials {
		if origin == "*" {
			// when responding to a credentialed request, server must specify a
			// domain, and cannot use wild carding.
			// See *important note* in https://developer.mozilla.org/en-US/docs/Web/HTTP/Access_control_CORS#Requests_with_credentials .
			ctx.Res.Header().Del(gear.HeaderAccessControlAllowCredentials)
		} else {
			ctx.Set(gear.HeaderAccessControlAllowCredentials, "true")
		}
	}
}
