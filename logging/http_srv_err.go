package logging

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
// 	srvLogger := &logger.HttpSrvErrMgr{}
//  srvLogger.AddIgnoreErr("http: TLS handshake error")
//  srvLogger.AddIgnoreErr("EOF")
//  app.Set(gear.SetLogger, log.New(srvLogger, "", log.LstdFlags))

var httpErr = &HttpSrvErrMgr{
	ignoreErrs:    [][]byte{[]byte("http: TLS handshake error"), []byte("EOF")},
	defaultWriter: os.Stderr,
}

func DefaultSrvErr() *HttpSrvErrMgr {
	return httpErr
}

type HttpSrvErrMgr struct {
	ignoreErrs    [][]byte
	defaultWriter io.Writer
}

// for test
func (s *HttpSrvErrMgr) SetDefault(out io.Writer) {
	s.defaultWriter = out
}

func (s *HttpSrvErrMgr) AddIgnoreErr(err string) {
	s.ignoreErrs = append(s.ignoreErrs, []byte(err))
}

func (s *HttpSrvErrMgr) Write(p []byte) (n int, err error) {
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
