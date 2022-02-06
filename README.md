![Gear](https://raw.githubusercontent.com/teambition/gear/master/gear.png)
[![Build Status](http://img.shields.io/travis/teambition/gear.svg?style=flat-square)](https://travis-ci.org/teambition/gear)
[![Coverage Status](http://img.shields.io/coveralls/teambition/gear.svg?style=flat-square)](https://coveralls.io/r/teambition/gear)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/teambition/gear/master/LICENSE)
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/teambition/gear)

A lightweight, composable and high performance web service framework for Go.

## Features

- Effective and flexible middlewares flow control, create anything by middleware
- Powerful and smart HTTP error handling
- Trie base gear.Router, as faster as [HttpRouter](https://github.com/julienschmidt/httprouter), support regexp parameters and group routes
- Integrated timeout context.Context
- Integrated response content compress
- Integrated structured logging middleware
- Integrated request body parser
- Integrated signed cookies
- Integrated JSON, JSONP, XML and HTML renderer
- Integrated CORS, Secure, Favicon and Static middlewares
- More useful methods on gear.Context to manipulate HTTP Request/Response
- Run HTTP and gRPC on the same port
- Completely HTTP/2.0 supported

## Documentation

[Go-Documentation](https://godoc.org/github.com/teambition/gear)

## Import

```go
// package gear
import "github.com/teambition/gear"
```

## Design

1. [Server 底层基于原生 net/http 而不是 fasthttp](https://github.com/teambition/gear/blob/master/doc/design.md#1-server-底层基于原生-nethttp-而不是-fasthttp)
1. [通过 gear.Middleware 中间件模式扩展功能模块](https://github.com/teambition/gear/blob/master/doc/design.md#2-通过-gearmiddleware-中间件模式扩展功能模块)
1. [中间件的单向顺序流程控制和级联流程控制](https://github.com/teambition/gear/blob/master/doc/design.md#3-中间件的单向顺序流程控制和级联流程控制)
1. [功能强大，完美集成 context.Context 的 gear.Context](https://github.com/teambition/gear/blob/master/doc/design.md#4-功能强大完美集成-contextcontext-的-gearcontext)
1. [集中、智能、可自定义的错误和异常处理](https://github.com/teambition/gear/blob/master/doc/design.md#5-集中智能可自定义的错误和异常处理)
1. [After Hook 和 End Hook 的后置处理](https://github.com/teambition/gear/blob/master/doc/design.md#6-after-hook-和-end-hook-的后置处理)
1. [Any interface 无限的 gear.Context 状态扩展能力](https://github.com/teambition/gear/blob/master/doc/design.md#7-any-interface-无限的-gearcontext-状态扩展能力)
1. [请求数据的解析和验证](https://github.com/teambition/gear/blob/master/doc/design.md#8-请求数据的解析和验证)

## FAQ

1. [如何从源码自动生成 Swagger v2 的文档？](https://github.com/teambition/gear/blob/master/doc/faq.md#1-如何从源码自动生成-swagger-v2-的文档)
1. [Go 语言完整的应用项目结构最佳实践是怎样的？](https://github.com/teambition/gear/blob/master/doc/faq.md#2-go-语言完整的应用项目结构最佳实践是怎样的)

## Demo

### Hello

https://github.com/teambition/gear/tree/master/example/hello

```go
  app := gear.New()

  // Add logging middleware
  app.UseHandler(logging.Default(true))

  // Add router middleware
  router := gear.NewRouter()

  // try: http://127.0.0.1:3000/hello
  router.Get("/hello", func(ctx *gear.Context) error {
    return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
  })

  // try: http://127.0.0.1:3000/test?query=hello
  router.Otherwise(func(ctx *gear.Context) error {
    return ctx.JSON(200, map[string]interface{}{
      "Host":    ctx.Host,
      "Method":  ctx.Method,
      "Path":    ctx.Path,
      "URI":     ctx.Req.RequestURI,
      "Headers": ctx.Req.Header,
    })
  })
  app.UseHandler(router)
  app.Error(app.Listen(":3000"))
```

### HTTP2 with Push

https://github.com/teambition/gear/tree/master/example/http2

```go
package main

import (
  "net/http"

  "github.com/teambition/gear"
  "github.com/teambition/gear/logging"
  "github.com/teambition/gear/middleware/favicon"
)

// go run example/http2/app.go
// Visit: https://127.0.0.1:3000/
func main() {
  const htmlBody = `
<!DOCTYPE html>
<html>
  <head>
    <link href="/hello.css" rel="stylesheet" type="text/css">
  </head>
  <body>
    <h1>Hello, Gear!</h1>
  </body>
</html>`

  const pushBody = `
h1 {
  color: red;
}
`

  app := gear.New()

  app.UseHandler(logging.Default(true))
  app.Use(favicon.New("./testdata/favicon.ico"))

  router := gear.NewRouter()
  router.Get("/", func(ctx *gear.Context) error {
    ctx.Res.Push("/hello.css", &http.PushOptions{Method: "GET"})
    return ctx.HTML(200, htmlBody)
  })
  router.Get("/hello.css", func(ctx *gear.Context) error {
    ctx.Type("text/css")
    return ctx.End(200, []byte(pushBody))
  })
  app.UseHandler(router)
  app.Error(app.ListenTLS(":3000", "./testdata/out/test.crt", "./testdata/out/test.key"))
}
```

### A CMD tool: static server

https://github.com/teambition/gear/tree/master/example/staticgo

Install it with go:

```sh
go install github.com/teambition/gear/example/staticgo
```

It is a useful CMD tool that serve your local files as web server (support TLS).
You can build `osx`, `linux`, `windows` version with `make build`.

```go
package main

import (
  "flag"

  "github.com/teambition/gear"
  "github.com/teambition/gear/logging"
  "github.com/teambition/gear/middleware/cors"
  "github.com/teambition/gear/middleware/static"
)

var (
  address  = flag.String("addr", "127.0.0.1:3000", `address to listen on.`)
  path     = flag.String("path", "./", `static files path to serve.`)
  certFile = flag.String("certFile", "", `certFile path, used to create TLS static server.`)
  keyFile  = flag.String("keyFile", "", `keyFile path, used to create TLS static server.`)
)

func main() {
  flag.Parse()
  app := gear.New()

  app.UseHandler(logging.Default(true))
  app.Use(cors.New())
  app.Use(static.New(static.Options{Root: *path}))

  logging.Println("staticgo v1.1.0, created by https://github.com/teambition/gear")
  logging.Printf("listen: %s, serve: %s\n", *address, *path)

  if *certFile != "" && *keyFile != "" {
    app.Error(app.ListenTLS(*address, *certFile, *keyFile))
  } else {
    app.Error(app.Listen(*address))
  }
}
```

### HTTP2 & gRPC

https://github.com/teambition/gear/tree/master/example/grpc_server

https://github.com/teambition/gear/tree/master/example/grpc_client

## About Router

[gear.Router](https://godoc.org/github.com/teambition/gear#Router) is a trie base HTTP request handler.
Features:

1. Support named parameter
1. Support regexp
1. Support suffix matching
1. Support multi-router
1. Support router layer middlewares
1. Support fixed path automatic redirection
1. Support trailing slash automatic redirection
1. Automatic handle `405 Method Not Allowed`
1. Automatic handle `OPTIONS` method
1. Best Performance

The registered path, against which the router matches incoming requests, can contain six types of parameters:

| Syntax | Description |
|--------|------|
| `:name` | named parameter |
| `:name(regexp)` | named with regexp parameter |
| `:name+suffix` | named parameter with suffix matching |
| `:name(regexp)+suffix` | named with regexp parameter and suffix matching |
| `:name*` | named with catch-all parameter |
| `::name` | not named parameter, it is literal `:name` |

Named parameters are dynamic path segments. They match anything until the next '/' or the path end:

Defined: `/api/:type/:ID`

```md
/api/user/123             matched: type="user", ID="123"
/api/user                 no match
/api/user/123/comments    no match
```

Named with regexp parameters match anything using regexp until the next '/' or the path end:

Defined: `/api/:type/:ID(^\d+$)`

```md
/api/user/123             matched: type="user", ID="123"
/api/user                 no match
/api/user/abc             no match
/api/user/123/comments    no match
```

Named parameters with suffix, such as [Google API Design](https://cloud.google.com/apis/design/custom_methods):

Defined: `/api/:resource/:ID+:undelete`

```md
/api/file/123                     no match
/api/file/123:undelete            matched: resource="file", ID="123"
/api/file/123:undelete/comments   no match
```

Named with regexp parameters and suffix:

Defined: `/api/:resource/:ID(^\d+$)+:cancel`

```md
/api/task/123                   no match
/api/task/123:cancel            matched: resource="task", ID="123"
/api/task/abc:cancel            no match
```

Named with catch-all parameters match anything until the path end, including the directory index (the '/' before the catch-all). Since they match anything until the end, catch-all parameters must always be the final path element.

Defined: `/files/:filepath*`

```
/files                           no match
/files/LICENSE                   matched: filepath="LICENSE"
/files/templates/article.html    matched: filepath="templates/article.html"
```

The value of parameters is saved on the `Matched.Params`. Retrieve the value of a parameter by name:

```go
type := matched.Params("type")
id   := matched.Params("ID")
```

## More Middlewares

- Structured logging: [github.com/teambition/gear/logging](https://github.com/teambition/gear/tree/master/logging)
- CORS handler: [github.com/teambition/gear/middleware/cors](https://github.com/teambition/gear/tree/master/middleware/cors)
- Secure handler: [github.com/teambition/gear/middleware/secure](https://github.com/teambition/gear/tree/master/middleware/secure)
- Static serving: [github.com/teambition/gear/middleware/static](https://github.com/teambition/gear/tree/master/middleware/static)
- Favicon serving: [github.com/teambition/gear/middleware/favicon](https://github.com/teambition/gear/tree/master/middleware/favicon)
- gRPC serving: [github.com/teambition/gear/middleware/grpc](https://github.com/teambition/gear/tree/master/middleware/grpc)
- JWT and Crypto auth: [Gear-Auth](https://github.com/teambition/gear-auth)
- Cookie session: [Gear-Session](https://github.com/teambition/gear-session)
- Session middleware: [https://github.com/go-session/gear-session](https://github.com/go-session/gear-session)
- Smart rate limiter: [Gear-Ratelimiter](https://github.com/teambition/gear-ratelimiter)
- CSRF: [Gear-CSRF](https://github.com/teambition/gear-csrf)
- Opentracing with Zipkin: [Gear-Tracing](https://github.com/teambition/gear-tracing)

## License

Gear is licensed under the [MIT](https://github.com/teambition/gear/blob/master/LICENSE) license.
Copyright &copy; 2016-2022 [Teambition](https://www.teambition.com).
