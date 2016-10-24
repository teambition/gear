![Gear](https://raw.githubusercontent.com/teambition/gear/master/gear.png)
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/teambition/gear)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/teambition/gear/master/LICENSE)
[![Build Status](http://img.shields.io/travis/teambition/gear.svg?style=flat-square)](https://travis-ci.org/teambition/gear)
[![Coverage Status](http://img.shields.io/coveralls/teambition/gear.svg?style=flat-square)](https://coveralls.io/r/teambition/gear)

-----
Gear implements a web framework with context.Context for Go. It focuses on performance and composition.

## Demo
```go
package main

import (
	"errors"
	"fmt"

	"github.com/teambition/gear"
	"github.com/teambition/gear/middleware"
)

func main() {
	// Create app
	app := gear.New()

	// Use a default logger middleware
	app.Use(gear.NewDefaultLogger())

	// Add a static middleware
	// http://localhost:3000/middleware/static.go
	app.Use(middleware.NewStatic(middleware.StaticOptions{
		Root:        "./dist",
		Prefix:      "/static",
		StripPrefix: true,
	}))

	// Add some middleware to app
	app.Use(func(ctx *gear.Context) (err error) {
		// fmt.Println(ctx.IP(), ctx.Method, ctx.Path
		// Do something...

		// Add after hook to the ctx
		ctx.After(func(ctx *gear.Context) {
			// Do something in after hook
			fmt.Println("After hook")
		})
		return
	})

	// Create views router
	ViewRouter := gear.NewRouter("", true)
	// "http://localhost:3000"
	ViewRouter.Get("/", func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
	})
	// "http://localhost:3000/view/abc"
	// "http://localhost:3000/view/123"
	ViewRouter.Get("/view/:view", func(ctx *gear.Context) error {
		view := ctx.Param("view")
		if view == "" {
			ctx.Status(400)
			return errors.New("Invalid view")
		}
		return ctx.HTML(200, "View: "+view)
	})
	// "http://localhost:3000/abc"
	// "http://localhost:3000/abc/efg"
	ViewRouter.Get("/:others*", func(ctx *gear.Context) error {
		others := ctx.Param("others")
		if others == "" {
			ctx.Status(400)
			return errors.New("Invalid path")
		}
		return ctx.HTML(200, "Request path: /"+others)
	})

	// Create API router
	APIRouter := gear.NewRouter("/api", true)
	// "http://localhost:3000/api/user/abc"
	// "http://localhost:3000/abc/user/123"
	APIRouter.Get("/user/:id", func(ctx *gear.Context) error {
		id := ctx.Param("id")
		if id == "" {
			ctx.Status(400)
			return errors.New("Invalid user id")
		}
		return ctx.JSON(200, map[string]string{
			"Method": ctx.Method,
			"Path":   ctx.Path,
			"UserID": id,
		})
	})

	// Must add APIRouter first.
	app.UseHandler(APIRouter)
	app.UseHandler(ViewRouter)
	// Start app at 3000
	app.Error(app.Listen(":3000"))
}
```

## Import

```go
// package gear
import "github.com/teambition/gear"
```

## Document

https://godoc.org/github.com/teambition/gear

## Middleware

https://godoc.org/github.com/teambition/gear/middleware

```go
// package middleware
"github.com/teambition/gear/middleware"
```

1. middleware.NewFavicon https://github.com/teambition/gear/blob/master/middleware/favicon.go
2. middleware.NewStatic https://github.com/teambition/gear/blob/master/middleware/static.go
3. middleware.NewTimeout https://github.com/teambition/gear/blob/master/middleware/timeout.go

## Bench
https://godoc.org/github.com/teambition/gear/blob/master/bench

### Gear with "net/http": 48307
```sh
> wrk 'http://localhost:3333/?foo[bar]=baz' -d 10 -c 100 -t 4

Running 10s test @ http://localhost:3333/?foo[bar]=baz
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.22ms    3.91ms 155.60ms   97.49%
    Req/Sec    12.58k     1.26k   18.76k    84.25%
  501031 requests in 10.01s, 65.46MB read
Requests/sec:  50030.72
Transfer/sec:      6.54MB
```

### Iris with "fasthttp": 70310
```sh
> wrk 'http://localhost:3333/?foo[bar]=baz' -d 10 -c 100 -t 4

Running 10s test @ http://localhost:3333/?foo[bar]=baz
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.37ms  648.31us  15.60ms   89.48%
    Req/Sec    17.75k     2.32k   39.65k    84.83%
  710317 requests in 10.10s, 102.29MB read
Requests/sec:  70310.19
Transfer/sec:     10.13MB
```

### Gin with "net/http": 48307
```sh
> wrk 'http://localhost:3333/?foo[bar]=baz' -d 10 -c 100 -t 4

Running 10s test @ http://localhost:3333/?foo[bar]=baz
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.07ms    1.50ms  30.44ms   90.04%
    Req/Sec    12.62k     1.12k   15.42k    77.50%
  502815 requests in 10.02s, 65.69MB read
Requests/sec:  50195.68
Transfer/sec:      6.56MB
```

## License
Gear is licensed under the [MIT](https://github.com/teambition/gear/blob/master/LICENSE) license.
Copyright &copy; 2016 [Teambition](https://www.teambition.com).
