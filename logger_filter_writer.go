package gear

import (
	"bytes"
	"io"
	"os"
)

// LoggerFilterWriter is a writer for Logger to filter bytes.
// In a https server, avoid some handshake mismatch condition such as loadbalance healthcheck:
//
//  2017/06/09 07:18:04 http: TLS handshake error from 10.10.5.1:45001: tls: first record does not look like a TLS handshake
//  2017/06/14 02:39:29 http: TLS handshake error from 10.0.1.2:54975: read tcp 10.10.5.22:8081->10.0.1.2:54975: read: connection reset by peer
//
// Usage:
//
//  func main() {
//  	app := gear.New() // Create app
//  	app.Use(func(ctx *gear.Context) error {
//  		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//  	})
//
//  	app.Set(gear.SetLogger, log.New(gear.DefaultFilterWriter(), "", log.LstdFlags))
//  	app.Listen(":3000")
//  }
//
type LoggerFilterWriter struct {
	phrases [][]byte
	out     io.Writer
}

var loggerFilterWriter = &LoggerFilterWriter{
	phrases: [][]byte{[]byte("http: TLS handshake error"), []byte("EOF")},
	out:     os.Stderr,
}

// DefaultFilterWriter returns the default LoggerFilterWriter instance.
func DefaultFilterWriter() *LoggerFilterWriter {
	return loggerFilterWriter
}

// SetOutput sets the output destination for the loggerFilterWriter.
func (s *LoggerFilterWriter) SetOutput(out io.Writer) {
	s.out = out
}

// Add add a phrase string to filter
func (s *LoggerFilterWriter) Add(err string) {
	s.phrases = append(s.phrases, []byte(err))
}

func (s *LoggerFilterWriter) Write(p []byte) (n int, err error) {
	for _, phrase := range s.phrases {
		if bytes.Contains(p, phrase) {
			return len(p), nil
		}
	}

	return s.out.Write(p)
}
