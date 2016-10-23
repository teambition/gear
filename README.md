Gear
=====
Gear implements a web framework with context.Context for Go. It focuses on performance and composition.

[![Build Status][travis-image]][travis-url]
[![GoDoc][GoDoc-image]][GoDoc-url]

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

## Bench
https://godoc.org/github.com/teambition/gear/blob/master/bench

### Gear with "net/http": 48307
```sh
> wrk 'http://localhost:3333/?foo[bar]=baz' -d 10 -c 100 -t 4

Running 10s test @ http://localhost:3333/?foo[bar]=baz
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.30ms    2.53ms  59.54ms   94.28%
    Req/Sec    12.15k     1.56k   20.98k    81.75%
  484231 requests in 10.02s, 63.27MB read
Requests/sec:  48307.40
Transfer/sec:      6.31MB
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

## License
Gear is licensed under the [MIT](https://github.com/teambition/gear/blob/master/LICENSE) license.
Copyright &copy; 2016 [Teambition](https://www.teambition.com).

[travis-url]: https://travis-ci.org/teambition/gear
[travis-image]: http://img.shields.io/travis/teambition/gear.svg

[GoDoc-url]: https://travis-ci.org/teambition/gear
[GoDoc-image]: https://godoc.org/github.com/teambition/gear?status.svg
