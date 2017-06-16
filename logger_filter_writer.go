package gear

import (
	"bytes"
	"io"
	"os"
)

// Https, avoid some handshake mismatch condition such as Aliyun SLB healthcheck (tcp or http)
//
// / 2017/06/09 07:18:04 http: TLS handshake error from 10.10.5.1:45001: tls: first record does not look like a TLS handshake
// 2017/06/14 02:39:29 http: TLS handshake error from 10.0.1.2:54975: read tcp 10.10.5.22:8081->10.0.1.2:54975: read: connection reset by peer
//
//
// Usage:
// func main() {
//	app := gear.New() // Create app
//	app.Use(func(ctx *gear.Context) error {
//		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//	})
//	// add http(s) error default mgr.
//	app.Set(gear.SetLogger, log.New(logging.DefaultFilterWriter(), "", log.LstdFlags))
//
//	app.Error(app.Listen(":3000"))
//}

var loggerFilterWriter = &LoggerFilterWriter{
	ignoreErrs:    [][]byte{[]byte("http: TLS handshake error"), []byte("EOF")},
	defaultWriter: os.Stderr,
}

func DefaultFilterWriter() *LoggerFilterWriter {
	return loggerFilterWriter
}

type LoggerFilterWriter struct {
	ignoreErrs    [][]byte
	defaultWriter io.Writer
}

// for test
func (s *LoggerFilterWriter) SetDefault(out io.Writer) {
	s.defaultWriter = out
}

func (s *LoggerFilterWriter) Add(err string) {
	s.ignoreErrs = append(s.ignoreErrs, []byte(err))
}

func (s *LoggerFilterWriter) Write(p []byte) (n int, err error) {
	skipFlag := false
	for _, ignore := range s.ignoreErrs {
		if bytes.Contains(p, ignore) {
			skipFlag = true
			break
		}
	}

	if skipFlag {
		return len(p), nil
	}

	return s.defaultWriter.Write(p)
}
