package logging

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIgnoreError(t *testing.T) {
	assert := assert.New(t)
	_ = assert

	testMsgs := []struct {
		Msg    string
		Expect string
	}{
		{"http: TLS handshake error from 10.10.5.1:45001: tls: first record does not look like a TLS handshake", ""},
		{"http: TLS handshake error from 10.0.1.2:54975: read tcp 10.10.5.22:8081->10.0.1.2:54975: read: connection reset by peer", ""},
		{"error from 10.0.1.2:54975: read EOF", ""},
		{"Hello World", "Hello World"},
	}

	for _, msg := range testMsgs {
		r, w, _ := os.Pipe()
		DefaultSrvErr().SetDefault(w)
		log := log.New(DefaultSrvErr(), "", log.LstdFlags)
		log.Print(msg.Msg)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if msg.Expect == "" {
			assert.Equal(buf.Bytes(), []byte(msg.Expect))
		} else {
			assert.Contains(string(buf.Bytes()), msg.Expect)
		}

	}
}
