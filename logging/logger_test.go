package logging

import (
	"bytes"
	"log"
	"math"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

// ----- Test Helpers -----
func EqualPtr(t *testing.T, a, b interface{}) {
	assert.Equal(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func NotEqualPtr(t *testing.T, a, b interface{}) {
	assert.NotEqual(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

type GearResponse struct {
	*http.Response
}

var DefaultClient = &http.Client{}

func RequestBy(method, url string) (*GearResponse, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func NewRequst(method, url string) (*http.Request, error) {
	return http.NewRequest(method, url, nil)
}
func DefaultClientDo(req *http.Request) (*GearResponse, error) {
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}

func TestGearLog(t *testing.T) {
	t.Run("Log.Format", func(t *testing.T) {
		assert := assert.New(t)

		log := Log{"value": 1}
		str, err := log.Format()
		assert.Nil(err)
		assert.Equal(`{"value":1}`, str)

		log = Log{"value": math.NaN}
		str, err = log.Format()
		assert.NotNil(err)
		assert.Equal("", str)
	})

	t.Run("Log.String", func(t *testing.T) {
		assert := assert.New(t)

		log := Log{"value": 1}
		assert.Equal(`Log{value:1}`, log.String())

		log = Log{"key": "test", "value": 1}
		assert.True(strings.Contains(log.String(), `key:"test"`))
		assert.True(strings.Contains(log.String(), `value:1`))
		assert.True(strings.Contains(log.String(), `, `))
	})

	t.Run("Log.From", func(t *testing.T) {
		assert := assert.New(t)

		log1 := Log{"key1": 1}
		log2 := Log{"key2": true}

		EqualPtr(t, log1, log1.From(log2))
		assert.Equal(Log{"key1": 1, "key2": true}, log1)
	})

	t.Run("Log.Into", func(t *testing.T) {
		assert := assert.New(t)

		log1 := Log{"key1": 1}
		log2 := Log{"key2": true}

		EqualPtr(t, log2, log1.Into(log2))
		assert.Equal(Log{"key2": true, "key1": 1}, log2)
	})

	t.Run("Log.With", func(t *testing.T) {
		assert := assert.New(t)

		log1 := Log{"key1": 1}
		log2 := Log{"key2": true}
		log3 := log1.With(log2)

		NotEqualPtr(t, log1, log3)
		NotEqualPtr(t, log2, log3)
		assert.Equal(Log{"key1": 1, "key2": true}, log3)
		assert.Equal(log3, log1.With(map[string]interface{}{"key1": 1, "key2": true}))
	})

	t.Run("Log.KV", func(t *testing.T) {
		assert := assert.New(t)

		log1 := Log{"key1": 1}
		log2 := log1.KV("key2", "a").KV("key3", true)

		EqualPtr(t, log1, log2)
		assert.Equal(true, log1["key3"])
		assert.Equal(true, log2["key3"])
	})
}

func TestGearLogger(t *testing.T) {
	exit = func() {} // overwrite exit function

	t.Run("Default logger", func(t *testing.T) {
		assert := assert.New(t)

		logger := Default()
		assert.Equal(logger.l, DebugLevel)
		assert.Equal(logger.tf, "2006-01-02T15:04:05.000Z")
		assert.Equal(logger.lf, "[%s] %s %s")

		var buf bytes.Buffer

		logger.Out = &buf
		logger.Emerg("Hello")
		assert.True(strings.Index(buf.String(), "Z] emerg {") > 0)
		buf.Reset()

		Emerg("Hello1")
		assert.True(strings.Index(buf.String(), "Z] emerg {") > 0)
		buf.Reset()

		logger.Alert("Hello")
		assert.True(strings.Index(buf.String(), "Z] alert {") > 0)
		buf.Reset()

		Alert("Hello1")
		assert.True(strings.Index(buf.String(), "Z] alert {") > 0)
		buf.Reset()

		logger.Crit("Hello")
		assert.True(strings.Index(buf.String(), "Z] crit {") > 0)
		buf.Reset()

		Crit("Hello1")
		assert.True(strings.Index(buf.String(), "Z] crit {") > 0)
		buf.Reset()

		logger.Err("Hello")
		assert.True(strings.Index(buf.String(), "Z] err {") > 0)
		buf.Reset()

		Err("Hello1")
		assert.True(strings.Index(buf.String(), "Z] err {") > 0)
		buf.Reset()

		logger.Err(Log{"error": math.NaN()})
		assert.True(strings.Contains(buf.String(), "Log{error:NaN}"))
		buf.Reset()

		err := gear.Err.WithMsg("test")
		err.Data = math.NaN()
		logger.Err(err)
		assert.True(strings.Contains(buf.String(), "] err Error{"))
		assert.True(strings.Contains(buf.String(), "Data:NaN"))
		buf.Reset()

		logger.Warning("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] warning Hello\n"))
		buf.Reset()

		Warning("Hello1")
		assert.True(strings.HasSuffix(buf.String(), "Z] warning Hello1\n"))
		buf.Reset()

		logger.Warning(Log{"error": "some \n err\r\nor"})
		assert.True(strings.HasSuffix(buf.String(), "Z] warning {\"error\":\"some \\n err\\r\\nor\"}\n"))
		buf.Reset()

		logger.Notice("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] notice Hello\n"))
		buf.Reset()

		Notice("Hello\r1\n")
		assert.True(strings.HasSuffix(buf.String(), "Z] notice Hello\\r1\n"))
		buf.Reset()

		logger.Notice(Log{"msg": "some\r\nmsg\n"})
		assert.True(strings.HasSuffix(buf.String(), "Z] notice {\"msg\":\"some\\r\\nmsg\\n\"}\n"))
		buf.Reset()

		logger.Info("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] info Hello\n"))
		buf.Reset()

		logger.Info(Log{"name": "gear"})
		assert.True(strings.HasSuffix(buf.String(), "Z] info {\"name\":\"gear\"}\n"))
		buf.Reset()

		logger.Info(Log{"nan": math.NaN()})
		assert.True(strings.HasSuffix(buf.String(), "Z] info Log{nan:NaN}\n"))
		buf.Reset()

		Info("Hello\r\n1\r\n")
		assert.True(strings.HasSuffix(buf.String(), "Z] info Hello\\r\\n1\\r\n"))
		buf.Reset()

		logger.Debug("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] debug Hello\n"))
		buf.Reset()

		Debug("Hello1")
		assert.True(strings.HasSuffix(buf.String(), "Z] debug Hello1\n"))
		buf.Reset()

		logger.Debugf(":%s\n", "Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] debug :Hello\n"))
		buf.Reset()

		Debugf(":%s\n", "Hello1")
		assert.True(strings.HasSuffix(buf.String(), "Z] debug :Hello1\n"))
		buf.Reset()

		assert.Panics(func() {
			logger.Panic("Hello")
		})
		assert.True(strings.Index(buf.String(), "emerg {") > 0)
		buf.Reset()

		assert.Panics(func() {
			Panic("Hello1")
		})
		assert.True(strings.Index(buf.String(), "emerg {") > 0)
		buf.Reset()

		logger.Fatal("Hello")
		assert.True(strings.Index(buf.String(), "emerg {") > 0)
		buf.Reset()

		Fatal("Hello1")
		assert.True(strings.Index(buf.String(), "emerg {") > 0)
		buf.Reset()

		logger.Print("Hello")
		assert.Equal(buf.String(), "Hello")
		buf.Reset()

		Print("Hello1")
		assert.Equal(buf.String(), "Hello1")
		buf.Reset()

		logger.Printf(":%s", "Hello")
		assert.Equal(buf.String(), ":Hello")
		buf.Reset()

		Printf(":%s", "Hello1")
		assert.Equal(buf.String(), ":Hello1")
		buf.Reset()

		logger.Println("Hello")
		assert.Equal(buf.String(), "Hello\n")
		buf.Reset()

		Println("Hello1")
		assert.Equal(buf.String(), "Hello1\n")
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "Hello")
		assert.True(strings.HasSuffix(buf.String(), "info Hello\n"))
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "")
		assert.True(strings.HasSuffix(buf.String(), "info \n"))
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "\n")
		assert.True(strings.HasSuffix(buf.String(), "info \n"))
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "\r")
		assert.True(strings.HasSuffix(buf.String(), "info \\r\n"))
		buf.Reset()

	})

	t.Run("GetLevel", func(t *testing.T) {
		assert := assert.New(t)

		log := Logger{}
		log.SetLevel(ErrLevel)

		assert.Equal(ErrLevel, log.GetLevel())
	})

	t.Run("logger setting", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		logger := New(&buf)
		assert.Panics(func() {
			var level Level = 8
			logger.SetLevel(level)
		})
		logger.SetLevel(NoticeLevel)
		logger.Info("Hello")
		assert.Equal(buf.String(), "")
		buf.Reset()

		logger.SetLogFormat("%s") // with invalid format
		logger.SetLevel(DebugLevel)
		logger.Info("Hello")
		assert.Equal(strings.Contains(buf.String(), "info"), true)
		buf.Reset()
	})
}

func TestGearLoggerMiddleware(t *testing.T) {
	t.Run("Default log", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()
		logger := Default()
		logger.Out = &buf
		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) error {
			log := logger.FromCtx(ctx)
			if ctx.Path == "/reset" {
				log.Reset()
			} else if ctx.Path == "/nan" {
				log["data"] = math.NaN()
			} else {
				log["data"] = []int{1, 2, 3}
			}
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:16])
		assert.Contains(log, "] info ")
		assert.Contains(log, `"data":[1,2,3]`)
		assert.Contains(log, `"method":"GET"`)
		assert.Contains(log, `"length":2`)
		assert.Contains(log, `"status":200`)
		res.Body.Close()

		buf.Reset()
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/nan")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log = buf.String()
		logger.mu.Unlock()
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:16])
		assert.Contains(log, "] info ")
		assert.Contains(log, `data:NaN`)
		assert.Contains(log, `method:"GET"`)
		assert.Contains(log, `length:2`)
		assert.Contains(log, `status:200`)
		res.Body.Close()

		buf.Reset()
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/reset")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		assert.Equal(buf.String(), "")
		logger.mu.Unlock()
		res.Body.Close()
	})

	t.Run("Default log with development mode", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("use native color func for windows platform")
		}
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()
		logger := Default(true)
		logger.Out = &buf
		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) error {
			log := FromCtx(ctx)
			EqualPtr(t, log, logger.FromCtx(ctx))
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()

		assert.Contains(log, "\x1b[32;1m127.0.0.1\x1b[39;22m - -")
		assert.Contains(log, `"GET / `)
		assert.Contains(log, "\x1b[32;1m200\x1b[39;22m")
		res.Body.Close()
	})

	t.Run("custom log", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()

		logger := New(&buf)
		logger.
			SetLogInit(func(log Log, ctx *gear.Context) {
				log["ip"] = ctx.IP()
				log["method"] = ctx.Method
				log["url"] = ctx.Req.URL.String()
				log["start"] = time.Now()
				log["userAgent"] = ctx.GetHeader(gear.HeaderUserAgent)
			}).
			SetLogConsume(func(log Log, _ *gear.Context) {
				end := time.Now()
				log["time"] = end.Sub(log["start"].(time.Time)) / 1e6
				delete(log, "start")
				if res, err := log.Format(); err == nil {
					logger.Output(end, InfoLevel, res)
				} else {
					logger.Output(end, WarningLevel, log.String())
				}
			})

		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) error {
			logger.SetTo(ctx, "data", []int{1, 2, 3})
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:18])
		assert.Contains(log, "] info ")
		assert.Contains(log, `"data":[1,2,3],`)
		assert.Contains(log, `"method":"GET",`)
		assert.Contains(log, `"length":2,`)
		assert.Contains(log, `"status":200,`)
		assert.Contains(log, `"userAgent":`)
		assert.Equal(rune(log[len(log)-1]), '\n')
		res.Body.Close()
	})

	t.Run("json log", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()

		logger := New(&buf)
		logger.SetJSONLog()
		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) error {
			logger.SetTo(ctx, "data", []int{1, 2, 3})
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.True(strings.HasPrefix(log, "{"))
		assert.True(strings.HasSuffix(log, "}\n"))
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:18])
		assert.Contains(log, `"data":[1,2,3]`)
		assert.Contains(log, `"method":"GET"`)
		res.Body.Close()
	})

	t.Run("Work with panic", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		var errbuf bytes.Buffer

		app := gear.New()
		app.Set(gear.SetLogger, log.New(&errbuf, "TEST: ", 0))

		logger := New(&buf)
		logger.
			SetLogInit(func(log Log, ctx *gear.Context) {
				log["ip"] = ctx.IP()
				log["method"] = ctx.Method
				log["uri"] = ctx.Req.URL.String()
				log["start"] = time.Now()
				log["userAgent"] = ctx.GetHeader(gear.HeaderUserAgent)
			}).
			SetLogConsume(func(log Log, _ *gear.Context) {
				end := time.Now()
				log["duration"] = end.Sub(log["start"].(time.Time)) / 1e6
				delete(log, "start")
				if res, err := log.Format(); err == nil {
					logger.Output(end, InfoLevel, res)
				} else {
					logger.Output(end, WarningLevel, log.String())
				}
			})

		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) (err error) {
			log := logger.FromCtx(ctx)
			log["data"] = map[string]interface{}{"a": 0}
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("POST", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("application/json; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:18])
		assert.Contains(log, "] info ")
		assert.Contains(log, `"data":{"a":0}`)
		assert.Contains(log, `"method":"POST"`)
		assert.Contains(log, `"status":500`)
		assert.Contains(log, `"userAgent":`)
		assert.Contains(errbuf.String(), "Some error")
		res.Body.Close()
	})

	t.Run("Color", func(t *testing.T) {
		assert := assert.New(t)

		assert.Equal(ColorGreen, colorStatus(200))
		assert.Equal(ColorGreen, colorStatus(204))
		assert.Equal(ColorCyan, colorStatus(304))
		assert.Equal(ColorYellow, colorStatus(404))
		assert.Equal(ColorRed, colorStatus(504))
	})

	t.Run("keep request body and response body when 500", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()
		logger := New(&buf)
		logger.SetJSONLog()
		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) error {
			var body bodyTpl
			if err := ctx.ParseBody(&body); err != nil {
				return err
			}
			panic("some error")
		})
		srv := app.Start()
		defer srv.Close()

		req, err := http.NewRequest("POST", "http://"+srv.Addr().String(), bytes.NewReader([]byte(`{"msg":"OK"}`)))
		assert.Nil(err)
		req.Header.Set("Content-Type", "application/json")
		res, err := DefaultClient.Do(req)
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)

		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.Contains(log, `"requestBody":"{\"msg\":\"OK\"}"`)
		assert.Contains(log, `"requestContentType":"application/json"`)
		assert.Contains(log, `"responseBody":"{\"error\":\"InternalServerError\",\"message\":\"some error\"}"`)
		assert.Contains(log, `"responseContentType":"application/json; charset=utf-8"`)
		res.Body.Close()
	})
}

func TestParseLevel(t *testing.T) {
	t.Run("ParseLevel", func(t *testing.T) {
		assert := assert.New(t)

		expected := map[string]Level{
			"emerg":     EmergLevel,
			"emergency": EmergLevel,
			"alert":     AlertLevel,
			"crit":      CritiLevel,
			"criti":     CritiLevel,
			"critical":  CritiLevel,
			"err":       ErrLevel,
			"error":     ErrLevel,
			"warn":      WarningLevel,
			"warning":   WarningLevel,
			"notice":    NoticeLevel,
			"info":      InfoLevel,
			"debug":     DebugLevel,
		}

		for key, expectedLevel := range expected {
			level, err := ParseLevel(key)
			assert.Equal(nil, err)
			assert.Equal(expectedLevel, level)

			level, err = ParseLevel(strings.ToUpper(key))
			assert.Equal(nil, err)
			assert.Equal(expectedLevel, level)
		}

		_, err := ParseLevel("unknown")
		assert.NotEqual(nil, err)
	})

	t.Run("SetLoggerLevel", func(t *testing.T) {
		assert := assert.New(t)

		logger := &Logger{}
		SetLoggerLevel(logger, "crit")
		assert.Nil(SetLoggerLevel(logger, "crit"))
		assert.Equal(CritLevel, logger.GetLevel())
	})
}

type bodyTpl map[string]string

func (b *bodyTpl) Validate() error {
	return nil
}
