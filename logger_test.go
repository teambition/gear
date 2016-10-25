package gear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testLogger struct{}

func (l *testLogger) Init(ctx *Context) {
	ctx.Log["IP"] = ctx.IP()
	ctx.Log["Method"] = ctx.Method
	ctx.Log["URL"] = ctx.Req.URL.String()
	ctx.Log["Start"] = time.Now()
	ctx.Log["UserAgent"] = ctx.Get(HeaderUserAgent)
}

func (l *testLogger) Format(log Log) string {
	// Tiny format: ":Date INFO :JSONInfo"
	end := time.Now()
	info := map[string]interface{}{
		"IP":        log["IP"],
		"Method":    log["Method"],
		"URL":       log["URL"],
		"UserAgent": log["UserAgent"],
		"Status":    log["Status"],
		"Length":    log["Length"],
		"Data":      log["Data"],
		"Time":      end.Sub(log["Start"].(time.Time)) / 1e6,
	}
	res, err := json.Marshal(info)
	if err != nil {
		return fmt.Sprintf("%s ERROR %s", end.Format(time.RFC3339), err.Error())
	}
	return fmt.Sprintf("%s INFO %s", end.Format(time.RFC3339), bytes.NewBuffer(res).String())
}

func TestGearLogger(t *testing.T) {
	t.Run("Simple log", func(t *testing.T) {
		var buf bytes.Buffer
		app := New()
		app.Use(NewLogger(&buf, &testLogger{}))
		app.Use(func(ctx *Context) (err error) {
			ctx.Log["Data"] = map[string]interface{}{}
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		req := NewRequst()
		res, err := req.Get("http://" + srv.Addr().String())
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "text/html; charset=utf-8", res.Header.Get(HeaderContentType))
		log := buf.String()
		require.Contains(t, log, time.Now().Format(time.RFC3339)[0:19])
		require.Contains(t, log, " INFO ")
		require.Contains(t, log, `"Data":{}`)
		require.Contains(t, log, `"Method":"GET"`)
		require.Contains(t, log, `"Status":200`)
		require.Contains(t, log, `"UserAgent":`)
		res.Body.Close()
	})

	t.Run("Work with panic", func(t *testing.T) {
		var buf bytes.Buffer
		var errbuf bytes.Buffer

		app := New()
		app.ErrorLog = log.New(&errbuf, "TEST: ", 0)
		app.Use(NewLogger(&buf, &testLogger{}))
		app.Use(func(ctx *Context) (err error) {
			ctx.Log["Data"] = map[string]interface{}{}
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		req := NewRequst()
		res, err := req.Post("http://" + srv.Addr().String())
		require.Nil(t, err)
		require.Equal(t, 500, res.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(HeaderContentType))
		log := buf.String()
		require.Contains(t, log, time.Now().Format(time.RFC3339)[0:19])
		require.Contains(t, log, " INFO ")
		require.Contains(t, log, `"Data":{}`)
		require.Contains(t, log, `"Method":"POST"`)
		require.Contains(t, log, `"Status":500`)
		require.Contains(t, log, `"UserAgent":`)
		require.Contains(t, errbuf.String(), "Some error")
		res.Body.Close()
	})
}
