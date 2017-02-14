package secure

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

var DefaultClient = &http.Client{}

func TestGearMiddlewareSecure(t *testing.T) {
	t.Run("DNSPrefetchControl", func(t *testing.T) {
		t.Run(`Should set X-DNS-Prefetch-Control header to "on" when set allow to true`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(DNSPrefetchControl(true))
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("on", res.Header.Get(gear.HeaderXDNSPrefetchControl))
		})

		t.Run(`Should set X-DNS-Prefetch-Control header to "off" when set allow to false`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(DNSPrefetchControl(false))
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("off", res.Header.Get(gear.HeaderXDNSPrefetchControl))
		})
	})

	t.Run("FrameGuard", func(t *testing.T) {
		t.Run(`Should set X-Frame-Options header to "DENY" when action is FrameGuardActionDeny`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(FrameGuard(FrameGuardActionDeny))
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("DENY", res.Header.Get(gear.HeaderXFrameOptions))
		})

		t.Run(`Should set X-Frame-Options header to "SAMEORIGIN" when action is FrameGuardActionSameOrigin`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(FrameGuard(FrameGuardActionSameOrigin))
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("SAMEORIGIN", res.Header.Get(gear.HeaderXFrameOptions))
		})

		t.Run(`Should set X-Frame-Options header to "ALLOW-FROM Domain" when action is FrameGuardActionAllowFrom`, func(t *testing.T) {
			assert := assert.New(t)
			domain := "test.org"

			app := getAppWithMiddleware(FrameGuard(FrameGuardActionAllowFrom, domain))
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("ALLOW-FROM "+domain, res.Header.Get(gear.HeaderXFrameOptions))
		})

		t.Run(`Should panic when action is FrameGuardActionAllowFrom but no domain`, func(t *testing.T) {
			assert := assert.New(t)

			assert.Panics(func() {
				getAppWithMiddleware(FrameGuard(FrameGuardActionAllowFrom))
			})
		})
	})

	t.Run("HidePoweredBy", func(t *testing.T) {
		t.Run(`Should remove X-Prowered-By header`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(HidePoweredBy())
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Empty(res.Header.Get(gear.HeaderXPoweredBy))
		})
	})

	t.Run("PublicKeyPinning", func(t *testing.T) {
		t.Run("Should enable public key pinning", func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(PublicKeyPinning(PublicKeyPinningOptions{
				MaxAge: 100 * time.Second,
				Sha256s: []string{
					"cUPcTAZWKaASuYWhhneDttWpY3oBAkE3h2+soZS7sWs=", "M8HztCzM3elUxkcjR2S5P4hhyBNf6lHkmjAHKhpGPWE",
				},
				ReportURI:         "test.org",
				IncludeSubdomains: true,
			}))

			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal(`pin-sha256="cUPcTAZWKaASuYWhhneDttWpY3oBAkE3h2+soZS7sWs=";pin-sha256="M8HztCzM3elUxkcjR2S5P4hhyBNf6lHkmjAHKhpGPWE";max-age=100;includeSubDomains;report-uri="test.org"`, res.Header.Get(gear.HeaderPublicKeyPins))
		})

		t.Run("Should enable public key pinning and report only", func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(PublicKeyPinning(PublicKeyPinningOptions{
				MaxAge: 100 * time.Second,
				Sha256s: []string{
					"cUPcTAZWKaASuYWhhneDttWpY3oBAkE3h2+soZS7sWs=", "M8HztCzM3elUxkcjR2S5P4hhyBNf6lHkmjAHKhpGPWE",
				},
				ReportURI:         "test.org",
				IncludeSubdomains: true,
				ReportOnly:        true,
			}))

			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal(`pin-sha256="cUPcTAZWKaASuYWhhneDttWpY3oBAkE3h2+soZS7sWs=";pin-sha256="M8HztCzM3elUxkcjR2S5P4hhyBNf6lHkmjAHKhpGPWE";max-age=100;includeSubDomains;report-uri="test.org"`, res.Header.Get(gear.HeaderPublicKeyPinsReportOnly))
		})

		t.Run(`Should panic when sha256s is empty`, func(t *testing.T) {
			assert := assert.New(t)

			assert.Panics(func() {
				getAppWithMiddleware(PublicKeyPinning(PublicKeyPinningOptions{
					Sha256s: []string{},
				}))
			})
		})
	})

	t.Run("StrictTransportSecurity", func(t *testing.T) {
		t.Run("Should enable strict transport security", func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(StrictTransportSecurity(StrictTransportSecurityOptions{
				MaxAge:            100 * time.Second,
				IncludeSubDomains: true,
				Preload:           true,
			}))

			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("max-age=100;includeSubDomains;preload;", res.Header.Get(gear.HeaderStrictTransportSecurity))
		})
	})

	t.Run("IENoOpen", func(t *testing.T) {
		t.Run(`Should set X-Download-Options header to "noopen"`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(IENoOpen())

			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("noopen", res.Header.Get(gear.HeaderXDownloadOptions))
		})
	})

	t.Run("NoSniff", func(t *testing.T) {
		t.Run(`Should set X-Content-Type-Options header to "nosniff"`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(NoSniff())

			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("nosniff", res.Header.Get(gear.HeaderXContentTypeOptions))
		})
	})

	t.Run("NoCache", func(t *testing.T) {
		t.Run(`Should set Cache-Control header`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(NoCache())
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("private, no-cache, max-age=0, s-max-age=0, must-revalidate",
				res.Header.Get(gear.HeaderCacheControl))
			assert.Equal("no-cache", res.Header.Get(gear.HeaderPragma))
			assert.Equal("0", res.Header.Get(gear.HeaderExpires))
		})
	})

	t.Run("SetReferrerPolicy", func(t *testing.T) {
		t.Run("Should set Referrer-Policy header to given policy", func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(SetReferrerPolicy(ReferrerPolicyOrigin))
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal(string(ReferrerPolicyOrigin), res.Header.Get(gear.HeaderRefererPolicy))
		})
	})

	t.Run("XSSFilter", func(t *testing.T) {
		t.Run(`Should set X-XSS-Protection header to "0" on old IE (<9)`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(XSSFilter())
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			// IE 8
			req.Header.Set(gear.HeaderUserAgent, "Mozilla/5.0 (compatible; MSIE 8.0; Windows NT 6.1; Trident/4.0; GTB7.4; InfoPath.2; SV1; .NET CLR 3.3.69573; WOW64; en-US)")
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("0", res.Header.Get(gear.HeaderXXSSProtection))
		})

		t.Run(`Should set X-XSS-Protection header to "1; mode=block" on new IE (>=9)`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(XSSFilter())
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			// IE 9
			req.Header.Set(gear.HeaderUserAgent, "Mozilla/5.0 (Windows; U; MSIE 9.0; WIndows NT 9.0; en-US))")
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("1; mode=block", res.Header.Get(gear.HeaderXXSSProtection))
		})

		t.Run(`Should set X-XSS-Protection header to "1; mode=block" when UA is not IE`, func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(XSSFilter())
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			// Firefox
			req.Header.Set(gear.HeaderUserAgent, "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:40.0) Gecko/20100101 Firefox/40.1")
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("1; mode=block", res.Header.Get(gear.HeaderXXSSProtection))
		})
	})

	t.Run("ContentSecurityPolicy", func(t *testing.T) {
		t.Run("Should set all given directives", func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(ContentSecurityPolicy(CSPDirectives{
				DefaultSrc: []string{"'slef'", "www.google-analytics.com"},
				Sandbox:    []string{"allow-forms", "allow-scripts"},
				ReportURI:  "/some-report-uri",
			}))
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("default-src 'slef' www.google-analytics.com;sandbox allow-forms allow-scripts;report-uri /some-report-uri;", res.Header.Get(gear.HeaderContentSecurityPolicy))
		})

		t.Run("Should set all given directives but report only", func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(ContentSecurityPolicy(CSPDirectives{
				DefaultSrc: []string{"'slef'", "www.google-analytics.com"},
				Sandbox:    []string{"allow-forms", "allow-scripts"},
				ReportURI:  "/some-report-uri",
				ReportOnly: true,
			}))

			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("default-src 'slef' www.google-analytics.com;sandbox allow-forms allow-scripts;report-uri /some-report-uri;", res.Header.Get(gear.HeaderContentSecurityPolicyReportOnly))
		})
	})

	t.Run("Default", func(t *testing.T) {
		t.Run("Should run default middlewares", func(t *testing.T) {
			assert := assert.New(t)

			app := getAppWithMiddleware(Default)
			srv := app.Start()
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr().String(), nil)
			assert.Nil(err)
			res, err := DefaultClient.Do(req)
			assert.Nil(err)
			assert.Equal("off", res.Header.Get(gear.HeaderXDNSPrefetchControl))
			assert.Empty(res.Header.Get(gear.HeaderXPoweredBy))
			assert.Equal("noopen", res.Header.Get(gear.HeaderXDownloadOptions))
			assert.Equal("nosniff", res.Header.Get(gear.HeaderXContentTypeOptions))
			assert.Equal("1; mode=block", res.Header.Get(gear.HeaderXXSSProtection))
			assert.Equal("max-age=15552000;includeSubDomains;", res.Header.Get(gear.HeaderStrictTransportSecurity))
			assert.Equal("private, no-cache, max-age=0, s-max-age=0, must-revalidate",
				res.Header.Get(gear.HeaderCacheControl))
			assert.Equal("no-cache", res.Header.Get(gear.HeaderPragma))
			assert.Equal("0", res.Header.Get(gear.HeaderExpires))
		})
	})
}

func getAppWithMiddleware(middleware gear.Middleware) *gear.App {
	app := gear.New()
	app.Use(middleware)
	app.Use(func(ctx *gear.Context) error {
		ctx.Set(gear.HeaderXPoweredBy, "Gear")
		return ctx.HTML(200, "OK")
	})

	return app
}
