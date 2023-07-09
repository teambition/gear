package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
)

var help bool
var passHostHeader bool
var certFile string
var keyFile string
var target string

func init() {
	flag.BoolVar(&help, "help", false, "show help info")
	flag.BoolVar(&passHostHeader, "passhost", false, "pass host header")
	flag.StringVar(&certFile, "cert", "", "certificate file path")
	flag.StringVar(&keyFile, "key", "", "private key file path")
	flag.StringVar(&target, "target", "", "target host")
}

// go run example/hello/main.go
func main() {
	flag.Parse()
	if help {
		output := flag.CommandLine.Output()
		fmt.Fprintf(output, "Usage of gearproxy:\n")
		flag.PrintDefaults()
		fmt.Fprintf(output, "\nProxy localhost request to remote:\n")
		fmt.Fprintf(output, "\tgearproxy -target 'https://github.com'\n")
		fmt.Fprintf(output, "\n\tVisit: http://127.0.0.1/teambition/gear\n")
		fmt.Fprintf(output, "\nProxy https request to localhost:\n")
		fmt.Fprintf(output, "\tgearproxy -passhost -target 'http://localhost:3000' -key 'path_to/privkey.pem' -cert 'path_to/fullchain_cert.pem'\n")
		fmt.Fprintf(output, "\n\tUpdate /etc/hosts and visit: https://yourdomain.com/home\n")
		return
	}

	app := gear.New()

	// Add logging middleware
	app.UseHandler(logging.Default(true))

	targetUrl, err := url.Parse(target)
	if err != nil {
		logging.Fatal(err)
	}

	proxy := buildProxy(passHostHeader, targetUrl)
	app.Use(gear.WrapHandler(proxy))

	if certFile == "" {
		logging.Info("gearproxy start at 80")
		app.ListenWithContext(gear.ContextWithSignal(context.Background()), ":80")
	} else {
		logging.Info("tgearproxy start at 443")
		app.ListenWithContext(gear.ContextWithSignal(context.Background()), ":443", certFile, keyFile)
	}
}

func buildProxy(passHostHeader bool, target *url.URL) http.Handler {
	proxy := &httputil.ReverseProxy{
		Director: func(outReq *http.Request) {
			u := outReq.URL
			if outReq.RequestURI != "" {
				parsedURL, err := url.ParseRequestURI(outReq.RequestURI)
				if err == nil {
					u = parsedURL
				}
			}

			outReq.Host = ""
			outReq.URL.Scheme = target.Scheme
			outReq.URL.Host = target.Host
			outReq.URL.Path = u.Path
			outReq.URL.RawPath = u.RawPath
			outReq.URL.RawQuery = strings.ReplaceAll(u.RawQuery, ";", "&")
			outReq.RequestURI = "" // Outgoing request should not have RequestURI

			outReq.Proto = "HTTP/1.1"
			outReq.ProtoMajor = 1
			outReq.ProtoMinor = 1

			// Do not pass client Host header unless optsetter PassHostHeader is set.
			if passHostHeader {
				outReq.Host = u.Host
			}
		},
		FlushInterval: time.Duration(time.Millisecond * 100),
		ErrorHandler: func(w http.ResponseWriter, request *http.Request, err error) {
			statusCode := http.StatusInternalServerError

			switch {
			case errors.Is(err, io.EOF):
				statusCode = http.StatusBadGateway
			case errors.Is(err, context.Canceled):
				statusCode = 499
			default:
				var netErr net.Error
				if errors.As(err, &netErr) {
					if netErr.Timeout() {
						statusCode = http.StatusGatewayTimeout
					} else {
						statusCode = http.StatusBadGateway
					}
				}
			}

			err = gear.ErrByStatus(statusCode).WithMsg(err.Error())
			logging.Debugf("'%d' caused by: %v", statusCode, err)
			w.WriteHeader(statusCode)
			_, werr := w.Write([]byte(err.Error()))
			if werr != nil {
				logging.Debugf("Error while writing status code: %v", werr)
			}
		},
	}

	return proxy
}
