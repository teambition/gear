package secure

// Inspire by https://github.com/helmetjs/helmet
// More info https://httpsecurityreport.com/best_practice.html

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/gear"
)

// FrameGuardAction represents a possible option of the "X-Frame-Options"
// header.
type FrameGuardAction string

// "X-Frame-Options" header options.
const (
	FrameGuardActionDeny       FrameGuardAction = "DENY"
	FrameGuardActionSameOrigin FrameGuardAction = "SAMEORIGIN"
	FrameGuardActionAllowFrom  FrameGuardAction = "ALLOW-FROM"
)

// ReferrerPolicy represents a possible policy of the "Referrer-Policy"
// header.
type ReferrerPolicy string

// Possible referrer policies.
const (
	ReferrerPolicyNoReferrer                  ReferrerPolicy = "no-referrer"
	ReferrerPolicyWhenDowngrade               ReferrerPolicy = "no-referrer-when-downgrade"
	ReferrerPolicyStrictOrigin                ReferrerPolicy = "strict-origin"
	ReferrerPolicyStrictOriginWhenCrossOrigin ReferrerPolicy = "strict-origin-when-cross-origin"
	ReferrerPolicySameOrigin                  ReferrerPolicy = "same-origin"
	ReferrerPolicyOrigin                      ReferrerPolicy = "origin"
	ReferrerPolicyOriginWhenCrossOrigin       ReferrerPolicy = "origin-when-cross-origin"
	ReferrerPolicyUnsafeURL                   ReferrerPolicy = "unsafe-url"
)

var oldIERegex = regexp.MustCompile(`(?i)msie\s*(\d+)`)

// Default provides protection for your Gear app by setting
// various HTTP headers.
//
//  app.Use(secure.Default)
//
// Equals:
//
//  app.Use(secure.DNSPrefetchControl(false))
//  app.Use(secure.HidePoweredBy())
//  app.Use(secure.IENoOpen())
//  app.Use(secure.NoSniff())
//  app.Use(secure.NoCache())
//  app.Use(secure.XSSFilter())
//  app.Use(secure.FrameGuard(FrameGuardActionSameOrigin))
//  app.Use(secure.StrictTransportSecurity(secure.StrictTransportSecurityOptions{
//  	MaxAge:            180 * 24 * time.Hour,
//  	IncludeSubDomains: true,
//  }))
//
var Default = gear.Compose(
	DNSPrefetchControl(false),
	HidePoweredBy(),
	IENoOpen(),
	NoSniff(),
	NoCache(),
	XSSFilter(),
	FrameGuard(FrameGuardActionSameOrigin),
	StrictTransportSecurity(StrictTransportSecurityOptions{
		MaxAge:            180 * 24 * time.Hour,
		IncludeSubDomains: true,
	}),
)

// DNSPrefetchControl controls browser DNS prefetching. And for potential
// privacy implications, it should be disabled.
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Controlling_DNS_prefetching .
func DNSPrefetchControl(allow bool) gear.Middleware {
	return func(ctx *gear.Context) error {
		if allow {
			ctx.SetHeader(gear.HeaderXDNSPrefetchControl, "on")
		} else {
			ctx.SetHeader(gear.HeaderXDNSPrefetchControl, "off")
		}
		return nil
	}
}

// FrameGuard mitigates clickjacking attacks by setting the X-Frame-Options
// header. Because ALLOW-FROM option only allow one domain, so when action is
// FrameGuardActionAllowFrom, you should only give one domain at the second
// parameter, and others will be ignored.
// See https://en.wikipedia.org/wiki/Clickjacking and https://developer.mozilla.org/en-US/docs/Web/HTTP/X-Frame-Options .
func FrameGuard(action FrameGuardAction, domains ...string) gear.Middleware {
	if action == FrameGuardActionAllowFrom && len(domains) != 1 {
		panic(gear.Err.WithMsg("'X-Frame-Options: ALLOW-FROM' only support one domain"))
	}

	return func(ctx *gear.Context) error {
		switch action {
		case FrameGuardActionDeny:
			ctx.SetHeader(gear.HeaderXFrameOptions, "DENY")
		case FrameGuardActionSameOrigin:
			ctx.SetHeader(gear.HeaderXFrameOptions, "SAMEORIGIN")
		case FrameGuardActionAllowFrom:
			ctx.SetHeader(gear.HeaderXFrameOptions, "ALLOW-FROM "+domains[0])
		}
		return nil
	}
}

// HidePoweredBy removes the X-Powered-By header to make it slightly harder for
// attackers to see what potentially-vulnerable technology powers your site.
func HidePoweredBy() gear.Middleware {
	return func(ctx *gear.Context) error {
		ctx.After(func() {
			ctx.Res.Header().Del(gear.HeaderXPoweredBy)
		})
		return nil
	}
}

// PublicKeyPinningOptions is public key pinning middleware options.
type PublicKeyPinningOptions struct {
	MaxAge            time.Duration
	Sha256s           []string
	ReportURI         string
	IncludeSubdomains bool
	ReportOnly        bool
}

// PublicKeyPinning helps you set the Public-Key-Pins header to prevent
// person-in-the-middle attacks(HPKP).
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Public_Key_Pinning .
func PublicKeyPinning(options PublicKeyPinningOptions) gear.Middleware {
	if len(options.Sha256s) == 0 {
		panic(gear.Err.WithMsg("empty Public-Key-Pins sha256s"))
	}

	return func(ctx *gear.Context) error {
		publicKeyPins := ""
		for _, sha256 := range options.Sha256s {
			publicKeyPins += fmt.Sprintf(`pin-sha256="%v";`, sha256)
		}
		if options.MaxAge != 0 {
			publicKeyPins += fmt.Sprintf("max-age=%.f;", options.MaxAge.Seconds())
		}
		if options.IncludeSubdomains {
			publicKeyPins += "includeSubDomains;"
		}
		if options.ReportURI != "" {
			publicKeyPins += fmt.Sprintf(`report-uri="%v"`, options.ReportURI)
		}

		if options.ReportOnly {
			ctx.SetHeader(gear.HeaderPublicKeyPinsReportOnly, publicKeyPins)
		} else {
			ctx.SetHeader(gear.HeaderPublicKeyPins, publicKeyPins)
		}
		return nil
	}
}

// StrictTransportSecurityOptions is the StrictTransportSecurity middleware
// options.
type StrictTransportSecurityOptions struct {
	MaxAge            time.Duration
	IncludeSubDomains bool
	Preload           bool
}

// StrictTransportSecurity sets the Strict-Transport-Security header to keep
// your users on HTTPS(HSTS).
// See https://developer.mozilla.org/en-US/docs/Web/Security/HTTP_strict_transport_security .
func StrictTransportSecurity(options StrictTransportSecurityOptions) gear.Middleware {
	return func(ctx *gear.Context) error {
		val := fmt.Sprintf("max-age=%.f;", options.MaxAge.Seconds())
		if options.IncludeSubDomains {
			val += "includeSubDomains;"
		}
		if options.Preload {
			val += "preload;"
		}

		ctx.SetHeader(gear.HeaderStrictTransportSecurity, val)
		return nil
	}
}

// IENoOpen sets the X-Download-Options to prevent Internet Explorer from
// executing downloads in your site’s context.
// See https://blogs.msdn.microsoft.com/ie/2008/07/02/ie8-security-part-v-comprehensive-protection/ .
func IENoOpen() gear.Middleware {
	return func(ctx *gear.Context) error {
		ctx.SetHeader(gear.HeaderXDownloadOptions, "noopen")
		return nil
	}
}

// NoSniff helps prevent browsers from trying to guess (“sniff”) the MIME type,
// which can have security implications. It does this by setting the
// X-Content-Type-Options header to nosniff.
// See https://blog.fox-it.com/2012/05/08/mime-sniffing-feature-or-vulnerability/ .
func NoSniff() gear.Middleware {
	return func(ctx *gear.Context) error {
		ctx.SetHeader(gear.HeaderXContentTypeOptions, "nosniff")
		return nil
	}
}

// NoCache will (try to) abolish all client-side caching.
// It's possible that you've got bugs in an old HTML or JavaScript file,
// and with a cache, some users will be stuck with those old versions.
// See https://github.com/helmetjs/nocache
// See http://docs.spring.io/spring-security/site/docs/current/reference/html/headers.html#headers-cache-control .
// `s-max-age=0` equal to `Surrogate-Control: no-store`
func NoCache() gear.Middleware {
	return func(ctx *gear.Context) error {
		ctx.SetHeader(gear.HeaderCacheControl, "private, no-cache, max-age=0, s-max-age=0, must-revalidate")
		ctx.SetHeader(gear.HeaderPragma, "no-cache")
		ctx.SetHeader(gear.HeaderExpires, "0")
		return nil
	}
}

// SetReferrerPolicy controls the behavior of the Referer header by setting the
// Referrer-Policy header.
// See https://www.w3.org/TR/referrer-policy/#referrer-policy-header .
func SetReferrerPolicy(policy ReferrerPolicy) gear.Middleware {
	return func(ctx *gear.Context) error {
		ctx.SetHeader(gear.HeaderRefererPolicy, string(policy))
		return nil
	}
}

// XSSFilter sets the X-XSS-Protection header to "1; mode=block" to prevent
// reflected XSS attacks. Because on old versions of IE (<9), this will cause
// some even worse security vulnerabilities, so it will set the header to "0"
// for old IE.
// See https://blogs.msdn.microsoft.com/ieinternals/2011/01/31/controlling-the-xss-filter/ .
func XSSFilter() gear.Middleware {
	return func(ctx *gear.Context) error {
		ieVersion, err := getIEVersionFromUA(ctx.GetHeader(gear.HeaderUserAgent))
		if err == nil && ieVersion < 9 {
			ctx.SetHeader(gear.HeaderXXSSProtection, "0")
		} else {
			ctx.SetHeader(gear.HeaderXXSSProtection, "1; mode=block")
		}
		return nil
	}
}

func getIEVersionFromUA(ua string) (float64, error) {
	matches := oldIERegex.FindStringSubmatch(ua)
	if len(matches) <= 1 {
		return 0, errors.New("secure: Not IE UserAgent")
	}
	return strconv.ParseFloat(matches[1], 64)
}

// CSPDirectives represents all valid directives that the
// "Content-Security-Policy" header is made up of.
type CSPDirectives struct {
	DefaultSrc     []string `csp:"default-src"`
	ScriptSrc      []string `csp:"script-src"`
	StyleSrc       []string `csp:"style-src"`
	ImgSrc         []string `csp:"img-src"`
	ConnectSrc     []string `csp:"connect-src"`
	FontSrc        []string `csp:"font-src"`
	ObjectSrc      []string `csp:"object-src"`
	MediaSrc       []string `csp:"media-src"`
	FrameSrc       []string `csp:"frame-src"`
	ChildSrc       []string `csp:"child-src"`
	Sandbox        []string `csp:"sandbox"`
	FormAction     []string `csp:"form-action"`
	FrameAncestors []string `csp:"frame-ancestors"`
	PluginTypes    []string `csp:"plugin-types"`
	ReportURI      string   `csp:"report-uri"`
	ReportOnly     bool
}

// ContentSecurityPolicy (CSP) sets the Content-Security-Policy header which
// can help protect against malicious injection of JavaScript, CSS, plugins,
// and more.
// See https://content-security-policy.com .
func ContentSecurityPolicy(directives CSPDirectives) gear.Middleware {
	return func(ctx *gear.Context) error {
		csp := ""
		elems := reflect.ValueOf(&directives).Elem()

		for i := 0; i < elems.NumField(); i++ {
			val := elems.Field(i)
			typ := elems.Type().Field(i)
			if val.Kind() != reflect.Slice || val.Len() == 0 {
				continue
			}
			csp += (typ.Tag.Get("csp") + " " + strings.Join(val.Interface().([]string), " ") + ";")
		}

		if directives.ReportURI != "" {
			csp += fmt.Sprintf("report-uri %v;", directives.ReportURI)
		}

		if directives.ReportOnly {
			ctx.SetHeader(gear.HeaderContentSecurityPolicyReportOnly, csp)
		} else {
			ctx.SetHeader(gear.HeaderContentSecurityPolicy, csp)
		}
		return nil
	}
}
