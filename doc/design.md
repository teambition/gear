# [Gear](https://github.com/teambition/gear) 框架设计考量

Gear 是由 [Teambition](https://www.teambition.com) 开发的一个轻量级的、专注于可组合扩展和高性能的 Go 语言 Web 服务框架。

Gear 框架在设计与实现的过程中充分参考了 Go 语言下多款知名 Web 框架，也参考了 Node.js 下的知名 Web 框架，汲取各方优秀因素，结合我们的开发实践，精心打磨而成。Gear 框架主要有如下特点：

1. 基于中间件模式的业务处理控制流程。中间件模式使功能模块开发标准化、解耦、易于组合和集成到应用
1. 框架级的错误和异常自动处理机制。开发者无需再担心业务逻辑中的每一个错误，只需在中间件返回错误，交给框架自动处理，也支持自定义处理逻辑
1. 集成了便捷的读写 HTTP Request/Response 对象的方法，使得 Web 应用开发更加高效
1. 高效而强大的路由处理器，能定义出各种路由规则满足业务逻辑需求
1. 丰富的中间件生态，如 CORS, CSRF, Secure, Logging, Favicon, Session, Rate limiter, Tracing 等
1. 完整的 HTTP/2.0 支持
1. 超轻量级，框架只实现核心的、共性的需求，可选需求均通过外部中间件或库来满足，确保应用实现的灵活自由，不被框架绑定束缚

## Summary

- [1. Server 底层基于原生 net/http 而不是 fasthttp](#1-server-底层基于原生-nethttp-而不是-fasthttp)
- [2. 通过 gear.Middleware 中间件模式扩展功能模块](#2-通过-gearmiddleware-中间件模式扩展功能模块)
- [3. 中间件的单向顺序流程控制和级联流程控制](#3-中间件的单向顺序流程控制和级联流程控制)
- [4. 功能强大，完美集成 context.Context 的 gear.Context](#4-功能强大完美集成-contextcontext-的-gearcontext)
- [5. 集中、智能、可自定义的错误和异常处理](#5-集中智能可自定义的错误和异常处理)
- [6. After Hook 和 End Hook 的后置处理](#6-after-hook-和-end-hook-的后置处理)
- [7. Any interface 无限的 gear.Context 状态扩展能力](#7-any-interface-无限的-gearcontext-状态扩展能力)
- [8. 请求数据的解析和验证](#8-请求数据的解析和验证)
- TODO


## 1. Server 底层基于原生 `net/http` 而不是 `fasthttp`

我们在计划使用并调研 Go 语言时，各种 Web 框架相关评测中 [fasthttp](https://github.com/valyala/fasthttp) 的优异表现让我们对 Go 有了很大的信心。但随着对 Go 的逐步深入学习和使用，当我们决定构建自己的 Web 框架时，还是选择了原生的 `net/http` 作为框架底层。

一方面是 1.7，1.8 版 Go 的 `net/http` 性能已经很好了，在我的 MBP 电脑上 Gear 框架与基于 `fasthttp` 的 [Iris](https://github.com/kataras/iris) 框架（据称最快）评测比分约为 `5:7`，已经不再是当初号称的10倍、20倍差距。如果算上应用的业务逻辑的消耗，这个差距会变得更小，甚至可以忽略。并且可以预见，随着 Go 版本升级优化，`net/http` 的性能表现会越来越好。

另一方面从兼容性和生命力考量，随着 Go 语言的版本升级，性能之外，`net/http` 的功能也会越来越强大、越来越完善（比如 `HTTP/2`）。社区生态也在往这个方向聚集，之前基于 `fasthttp` 的很多框架都提供了 `net/http` 的选择（如 Iris, Echo 等）。


## 2. 通过 `gear.Middleware` 中间件模式扩展功能模块

中间件模式则是被各语言生态下 Web 框架验证的可组合扩展的最佳模式，但仍然有 **级联** 和 **单向顺序** 两个截然不同的中间件运行流程模式，Gear 选择是单向顺序运行中间件的模式（后面讲解原因）。

### 中间件的定义

**一个 `http.HandlerFunc` 风格的 `gear.Middleware` 中间件定义如下：**

```go
type Middleware func(ctx *Context) error
```

我们用 `App.Use` 加载一个直接响应 Hello 的中间件到 app 应用：

```go
app.Use(func(ctx *gear.Context) error {
  return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
})
```

**一个 `http.Handler` 风格的 `gear.Handler` 中间件定义如下：**

```go
type Handler interface {
  Serve(ctx *Context) error
}
```

我们用 `App.UseHandler` 加载一个 `gear.Router` 实例中间件到 app 应用，因为它实现了 [Handler interface](https://github.com/teambition/gear/blob/master/router.go#L248)：

```go
// https://github.com/teambition/gear/blob/master/example/http2/app.go
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
```

另外我们也可以这样加载 `gear.Handler` 中间件：

```go
app.Use(router.Serve)
```

两种形式的中间件各有其用处，但本质上都是：

```go
func(ctx *gear.Context) error
```

类型的函数。另外我们可以看到上面 Router 示例代码中也使用了中间件：

```go
router.Get(path, func(ctx *gear.Context) error {
  // ...
})
```

router 本身是个 `gear.Handler` 形式的中间件，而它的内含逻辑却又由更多的 `gear.Middleware` 类型的中间件组成。Gear 内置了一些核心的中间件，包括 `gear.Router` 中间件，`gear/logging` 目录下的 `logging.Logger` 中间件，`gear/middleware` 目录下的 `cors`, `favicon`, `secure`, `static` 中间件等，都是相同的组合逻辑。

另外 https://github.com/teambition 也有我们维护的一些 `gear-xxx` 的中间件，也非常欢迎开发者们参与 `gear-xxx` 中间件生态开发中来。

因此，`func(ctx *gear.Context) error` 形态的中间件是 Gear 组合扩展的元语。它有两个核心元素 `gear.Context` 和 `error`，其中 `gear.Context` 集成了 Gear 框架的所有核心开发能力（后面讲解），而返回值 `error` 则是框架提供的一个非常强大的错误处理机制。

### 中间件处理流程

一个完整 Gear 框架的 Request - Response 处理流程就是一系列中间件及其组合体的运行的流程，中间件按照引入的顺序逐一、单向运行（而非 **级联**），每个中间件解决一个特定的需求，与其它任何中间件没有耦合。

单向顺序处理流程模式的中间件最大的特点就是 `cancelable`，随时可以中断，后续中间件不再运行。对于 Gear 框架来说有四种可能情况中断（cancel）或结束中间件处理流程：

#### 正常响应中断

当某一个中间件调用了特定的方法（如 gear.Context 上的 `ctx.End`, `ctx.JSON`, `ctx.Error` 等，或者 Go 内置的 `http.Redirect`, `http.ServeContent` 等）直接往 `http.ResponseWriter` 写入数据时，中间件处理流程中断，后续的中间件（如果有）不再运行，请求处理流程正常结束。

一般这样的正常结束都位于中间件流程的最末端，如 router 路由分支的最后一个中间件。但也有从中间甚至一开始就中断的情况，比如 `static` 中间件：

```go
func main() {
  app := gear.New()
  app.Use(static.New(static.Options{
    Root:        "./testdata",
    Prefix:      "/",
    StripPrefix: false,
  }))
  app.Use(func(ctx *gear.Context) error {
    return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
  })
  app.Error(app.Listen(":3000"))
}
```

当请求是静态文件资源请求时，第二个响应 "Hello, Gear!" 的中间件就不再运行。

#### `error` 中断

当某一个中间件返回 `error` 时（比如 400 参数错误，401 身份验证错误，数据库请求错误等），中间件处理流程就被中断，后续的中间件不再运行，Gear 应用会自动处理这个错误，并做出对应的 response 响应（也可以由开发者自定义错误响应结果，如响应一个包含错误信息的 JSON）。开发者不再疲于 `error` 的处理，可以尽情的 `return error`。

另外通过 `ctx.Error(err)` 和 `ctx.ErrorStatus(statusCode)` 主动响应错误也算 `error` 中断。

```go
// https://github.com/seccom/kpass/blob/master/pkg/api/user.go
func (a *User) Login(ctx *gear.Context) (err error) {
  body := new(tplUserLogin)
  if err = ctx.ParseBody(body); err != nil {
    return
  }

  var user *schema.User
  if user, err = a.user.CheckLogin(body.ID, body.Pass); err != nil {
    return
  }

  token, err := auth.NewToken(user.ID)
  if err != nil {
    return ctx.Error(err)
  }
  ctx.Set(gear.HeaderPragma, "no-cache")
  ctx.Set(gear.HeaderCacheControl, "no-store")
  return ctx.JSON(200, map[string]interface{}{
    "access_token": token,
    "token_type":   "Bearer",
    "expires_in":   auth.JWT().GetExpiresIn().Seconds(),
  })
}
```

上面这个示例代码包含了两种形式的 `error` 中断。无论哪种，其 err 都会被 Gear 框架层自动识别处理（后面详解），响应给客户端。

与正常响应中断不同，`error` 中断及后面的异常中断都会导致通过 `ctx.After` 注入的 **after hooks** 逻辑被清理，不会运行（后面再详解），已设置的 response headers 也会被清理，只保留必要的 [headers](https://github.com/teambition/gear/blob/master/response.go#L11)

#### `context.Context` cancel 中断

当中间件处理流还在运行，请求却因为某些原因被 `context.Context` 机制 cancel 时（如处理超时），中间件处理流程也会被中断，cancel 的 error 会被提取，然后按照类似 `error` 中断逻辑被框架自动处理。

#### `panic` 中断

最后就是某些中间件运行时可能出现的 panic error，它们能被框架捕获并按照类似 `error` 中断逻辑自动处理，错误信息中还会包含错误堆栈（Error.Stack），方便开发者在运行日志中定位错误。


## 3. 中间件的单向顺序流程控制和级联流程控制

Node.js 生态中知名框架 [koa](https://github.com/koajs/koa) 就是 **级联** 流程控制，其文档中的一个示例代码如下：

```js
const app = new Koa();

app.use(async (ctx, next) => {
  try {
    await next();
  } catch (err) {
    ctx.body = { message: err.message };
    ctx.status = err.status || 500;
  }
});

app.use(async ctx => {
  const user = await User.getById(ctx.session.userid);
  ctx.body = user;
});
```

Node.js 中最知名最经典的框架 [Express](https://github.com/expressjs/express) 和类 koa 的 [Toa](https://github.com/toajs/toa) 则选择了 **单向顺序** 流程控制模式。

Go 语言生态中，[Iris](https://github.com/kataras/iris)，[Gin](https://github.com/gin-gonic/gin) 等采用了 **级联** 流程控制模式。Gin 文档中的一个示例代码如下：

```go
func Logger() gin.HandlerFunc {
  return func(c *gin.Context) {
    t := time.Now()
    // Set example variable
    c.Set("example", "12345")
    // before request
    c.Next()
    // after request
    latency := time.Since(t)
    log.Print(latency)

    // access the status we are sending
    status := c.Writer.Status()
    log.Println(status)
  }
}
```

示例代码中的 `await next()` 和 `c.Next()` 以及它们的上下文就是级联逻辑，next 包含了当前中间件所有下游中间件的逻辑。

相对于 **单向顺序**，**级联** 唯一的优势就是在当前上下文中实现了 **after** 逻辑：在当前运行栈中，处理完所有后续中间件后再回来继续处理，正如上面 Logger。 Gear 框架使用 **after hooks** 来满足这个需求，另外也有 **end hooks** 来精确处理 **级联** 中无法实现的需求（比如上面 Logger 中间件中 `c.Next()` panic 了，这个日志就没了）。

那么 **级联** 流程控制有什么问题呢？这里提出两点：

1. next 中的逻辑是个有状态的黑盒，当前中间件可能会与这个黑盒发生状态耦合，或者说这个黑盒导致当前中间件充满不确定性的状态，比如黑盒中是否出了错误（如果出了错要另外处理的话）？是否写入了响应数据？是否会 panic？这都是无法预知的。
1. 无法被 `context.Context` 的 cancel 终止，正如上所述，这个巨大的级联黑盒无法知道运行到哪一层时 cancel 了，只能默默的往下运行。

## 4. 功能强大，完美集成 `context.Context` 的 `gear.Context`

`gear.Context` 是中间件 `func(ctx *gear.Context) error` 的一个核心。它完全集成了 `context.Context`、`http.Request`、`http.ResponseWriter` 的能力，并且提供了很多核心、便捷的方法。开发者通过调用 `gear.Context` 即可快速实现各种 Web 业务逻辑。

`context.Context` 是 Go 语言原生的用于解决异步流程控制的方案，它主要为异步控制流程提供了完全 cancel 的能力和域内传值（request-scoped value）的能力，`net/http` 底层就使用了它。

`gear.Context` 充分利用了 `context.Context`，并实现了它的 interface，可以直接当成 `context.Context` 使用。也提供了 `ctx.WithCancel`, `ctx.WithDeadline`, `ctx.WithTimeout`, `ctx.WithValue` 等快速创建子级 `context.Context` 的便捷方法。还提供了 `ctx.Cancel` 主动完全退出中间件处理流程的方法。还有 [App 级设置](https://github.com/teambition/gear/blob/master/app.go#L264) 的中间件处理流程 timeout cancel 能力，甚至是 `ctx.Timing` 针对某个异步处理逻辑的 timeout cancel 能力等。

更多的方法请参考 https://godoc.org/github.com/teambition/gear#Context，估计要翻墙访问。


## 5. 集中、智能、可自定义的错误和异常处理

`error` 是中间件 `func(ctx *gear.Context) error` 的另一个核心。这个由 Golang 语言层定义的、最简单的 `error` interface 在 Gear 框架下，其灵活度和强大的潜力超出你的想象。

对于 Web 服务而言，`error` 中必须要包含两个信息：error message 和 error code。比如一个 `400 Bad request` 的 error，框架能提取 status code 和 message 的话，就能自动响应给客户端了。对于实际业务需求，这个 400 错误还需要包含更具体的错误信息，甚至包含 i18n 信息。

### `gear.HTTPError`，`gear.Error`

所以 Gear 框架定义了一个核心的 `gear.HTTPError` interface：

```go
type HTTPError interface {
  Error() string
  Status() int
}
```

`gear.HTTPError` interface 实现了 `error` interface。另外又定义了一个基础的通用的 `gear.Error` 类型：

```go
type Error struct {
  Code  int         `json:"-"`
  Err   string      `json:"error"`
  Msg   string      `json:"message"`
  Data  interface{} `json:"data,omitempty"`
  Stack string      `json:"-"`
}
```

它实现了 `gear.HTTPError` interface，并额外提供了 `Data` 和 `Stack` 分别用于保存更具体的错误信息和错误堆栈，还提供了几个特别有用的方法：

1. 用于错误日志输出的 `String` 方法：

    ```go
    func (err *Error) String() string {
      if v, ok := err.Data.([]byte); ok && utf8.Valid(v) {
        err.Data = string(v)
      }
      return fmt.Sprintf(`Error{Code:%d, Err:"%s", Msg:"%s", Data:%#v, Stack:"%s"}`,
        err.Code, err.Err, err.Msg, err.Data, err.Stack)
    }
    ```

1. 用于从给定 error 模板和错误信息快速生成新 error 的 `WithMsg` 方法：

    ```go
    func (err Error) WithMsg(msgs ...string) *Error {
      if len(msgs) > 0 {
        err.Msg = strings.Join(msgs, ", ")
      }
      return &err
    }
    ```

    其使用方法如下：

    ```go
    err := gear.ErrBadRequest.WithMsg() // 未提供 message 则相当于纯粹的 clone
    err := gear.ErrBadRequest.WithMsg("invalid email")
    err := gear.ErrBadRequest.WithMsg("invalid email", "invalid phone number") // 支持多个 message
    ```

    你也可以定义自己的 `400` 错误模板：

    ```go
    ErrParamRequire := &gear.Error{Code: http.StatusBadRequest, Err: "ParamRequired"}
    err := ErrParamRequire.WithMsg("user name required")
    ```

1. 用于从给定 error 模板和错误码快速生成新 error 的 `WithCode` 方法：

    ```go
    func (err Error) WithCode(code int) *Error {
      err.Code = code
      if text := http.StatusText(code); text != "" {
        err.Err = text
      }
      return &err
    }
    ```

    比如 Gear 框架内建的 4xx 和 5xx error：

    ```go
    Err = &Error{Code: http.StatusInternalServerError, Err: "Error"}
    ErrBadRequest                    = Err.WithCode(http.StatusBadRequest)
    ErrUnauthorized                  = Err.WithCode(http.StatusUnauthorized)
    ErrPaymentRequired               = Err.WithCode(http.StatusPaymentRequired)
    ```

1. 用于从给定 error 模板和错误速生成新 error 的 `From` 方法：

    ```go
    func (err Error) From(e error) *Error {
      if IsNil(e) {
        return nil
      }

      switch v := e.(type) {
      case *Error:
        return v
      case HTTPError:
        err.Code = v.Status()
        err.Msg = v.Error()
      case *textproto.Error:
        err.Code = v.Code
        err.Msg = v.Msg
      default:
        err.Msg = e.Error()
      }

      if err.Err == "" {
        err.Err = http.StatusText(err.Code)
      }
      return &err
    }
    ```

    该方法尝试把任何 `error interface` 对象转换成 `*gear.Error`

`gear.Error` 类型既可以像传统错误一样直接响应给客户端：

```go
ctx.End(err.Status(), []byte(err.Error()))
```

也可以用 JSON 的形式响应：

```go
ctx.JSON(err.Status(), err.Error)
```

Gear 对捕获到的 error 会默认以 JSON 响应给请求方，如：

```go
// middle returns some error
app.Use(func(ctx *Context) error {
  return errors.New("some error")
})
```

请求方会收到 `500 Internal Server Error` 的 JSON 响应:

```json
{"error":"Internal Server Error","message":"some error"}
```

对于必要的（如 5xx 系列）错误会进入 `App.Error` 处理，这样也保留了错误堆栈。

```go
func (app *App) Error(err error) {
  if err := ErrorWithStack(err, 4); err != nil {
    app.logger.Println(err.String())
  }
}
```

其中 `gear.ErrorWithStack` 就是创建一个包含错误堆栈的 `gear.Error`：

```go
func ErrorWithStack(val interface{}, skip ...int) *Error {
  if IsNil(val) {
    return nil
  }

  var err *Error
  switch v := val.(type) {
  case *Error:
    err = v.WithMsg() // must clone, should not change the origin *Error instance
  case error:
    err = ErrInternalServerError.From(v)
  case string:
    err = ErrInternalServerError.WithMsg(v)
  default:
    err = ErrInternalServerError.WithMsgf("%#v", v)
  }

  if err.Stack == "" {
    buf := make([]byte, 2048)
    buf = buf[:runtime.Stack(buf, false)]
    s := 1
    if len(skip) != 0 {
      s = skip[0]
    }
    err.Stack = pruneStack(buf, s)
  }
  return err
}
```

从其逻辑我们可以看出，如果 val 已经是 `gear.Error`，则直接使用，如果 err 没有包含 `Stack`，则追加。

Gear 框架内建了一个 `gear.Error` 常量 `gear.Err`:

```go
var Err = &Error{Code: http.StatusInternalServerError, Err: "Error"}
```

并从 `gear.Err` 派生了常用的 4xx 和 5xx 错误模板：

```go
// https://golang.org/pkg/net/http/#pkg-constants
ErrBadRequest                    = Err.WithCode(http.StatusBadRequest)
ErrUnauthorized                  = Err.WithCode(http.StatusUnauthorized)
ErrPaymentRequired               = Err.WithCode(http.StatusPaymentRequired)
ErrForbidden                     = Err.WithCode(http.StatusForbidden)
ErrNotFound                      = Err.WithCode(http.StatusNotFound)
// more...
```

这些内建的错误即可满足常规需求，Gear 的其它中间件就使用了它，比如 `gear.Router` 中，当路由未定义时会：

```go
if r.otherwise == nil {
  return ErrNotImplemented.WithMsgf(`"%s" is not implemented`, ctx.Path)
}
```

又比如 `cors` 中间件中，当跨域域名不允许时：

```go
if allowOrigin == "" {
  return gear.ErrForbidden.WithMsgf("Origin: %v is not allowed", origin)
}
```

### `gear.ParseError`，`gear.SetOnError`

如果你觉得 `gear.Error` 还无法满足需求，你完全可以参考它实现一个更复杂的 `gear.HTTPError` interface 的 error 类型。Gear 框架下完全可以自定义更复杂的，充满想象力的错误处理机制。

框架内的任何 `error` interface 的错误，都会经过 `gear.ParseError` 处理成 `gear.HTTPError` interface，然后再交给 `gear.SetOnError` 做进一步自定义处理：

```go
func ParseError(e error, code ...int) HTTPError {
  if IsNil(e) {
    return nil
  }

  switch v := e.(type) {
  case HTTPError:
    return v
  case *textproto.Error:
    err := Err.WithCode(v.Code)
    err.Msg = v.Msg
    return err
  default:
    err := ErrInternalServerError.WithMsg(e.Error())
    if len(code) > 0 && code[0] > 0 {
      err = err.WithCode(code[0])
    }
    return err
  }
}
```

从上面的处理逻辑我们可以看出，`gear.HTTPError` 会被直接返回，所以保留了原始错误的所有信息，如自定义的 json tag。其它错误会被加工处理，无法取得 status code 的错误则默认取 `500`。

我们可以定义自己的 `MyError` 类型，然后通过设置 `gear.SetOnError` 对它进行特殊处理。下面我们通过 `switch type` 判断如果 `httpError` 是我们自定义的 `MyError` 类型（也就是我们预期的在业务逻辑中使用的）则用 `ctx.JSON` 主动处理，否则不处理，而是由框架自动处理：

```go
app.Set(gear.SetOnError, func(ctx *gear.Context, httpError gear.HTTPError) {
  switch err := httpError.(type) {
  case MyError, *MyError:
    ctx.JSON(err.Code, err)
  }
})
```

这里再次强调，框架内捕捉的所有错误，包括 `ctx.Error(error)` 和 `ctx.ErrorStatus(statusCode)` 主动发起的，包括中间件 `return error` 返回的，包括 panic 的，也包括 `context.Context` cancel 引发的错误等，都是经过上面叙述的错误处理流程处理，响应给客户端，有必要的则输出到日志。

## 6. After Hook 和 End Hook 的后置处理

前文提到，在 **级联** 流程控制模式下，很容易实现一种后置的处理逻辑。比如 logging，当请求进来时，初始化 log 数据，当处理流程完成时，再把 log 写入 IO。Gear 框架用 hook `func()` 机制来实现这类需求，并且这种后置处理需求细分为 `ctx.After(hook func())` 和 `ctx.OnEnd(hook func())` 两种。比如，Gear 的 logging 中间件的主要处理逻辑是：

```go
func (l *Logger) Serve(ctx *gear.Context) error {
  log := l.FromCtx(ctx)
  // Add a "end hook" to flush logs
  ctx.OnEnd(func() {
    // Ignore empty log
    if len(log) == 0 {
      return
    }
    log["Status"] = ctx.Res.Status()
    log["Type"] = ctx.Res.Type()
    log["Length"] = ctx.Res.Get(gear.HeaderContentLength)
    l.consume(log, ctx)
  })
  return nil
}
```

 这是一个 `gear.Handler` 类型的中间件，与 Gin 框架的 Logger 中间件稍有差异，在进入到中间件时 `l.FromCtx(ctx)` 会初始化 log，而 log 的消费处理逻辑则是在 End Hook 中进行的，这样不会因为中间件的错误异常而导致 log 被丢失。

 开发者可以在中间件处理流过程中动态的添加 After Hooks 和 End Hooks。中间件处理流完成后则不能再添加，否则会 panic 异常。

 After Hooks 将在中间件处理流结束后，`http.ResponseWriter` 的 `w.WriteHeader` 调用之前执行，End Hooks 则是在 `w.WriteHeader` 调用之后，一个独立的 **goroutine** 中执行（不阻塞当前处理进程），执行顺序与 Go 语言的 `defer` 一致，是 LIFO（后进先出）模式。所以，After Hooks 中仍然有修改 Response 内容的能力，比如修改 Headers, 或者 Cookie Session 的 Save 行为等。End Hooks 则不能再修改任何内容，只能做纯粹的后置处理逻辑，如写入日志，发起对外的 web hook 等。

 当中间件处理流出现错误或异常导致中断时，表明中间件处理流不再是预期的正常行为，After Hooks 队列将被清空，不会执行。但 End Hooks 仍会照样执行，这也是为什么 Gear logging 中间件的 `l.consume` 逻辑放在了 End Hook。

 再次说明，一般 **级联** 流程控制模式的框架都只能实现类 After Hook 的逻辑，而没有提供实现类 End Hook 逻辑的能力。这样主要有两个问题，一是中间件处理流异常时 After 处理逻辑会丢失；二是像 logging 这种需求，放在 End Hook 中处理在时间点上更准确。

## 7. Any interface 无限的 gear.Context 状态扩展能力

对于基于中间件模式的业务处理控制流程而言，在各个中间件之间传递业务逻辑的状态值很有必要。Go 语言原生的 `context.Context` 提供的 `Value` 能力并不能很好的满足这类需求。

比如 logging，我们需要传递一个 log 的结构体，在请求开始的时候初始化并追加一些初始状态数据，业务处理过程中再追加一些数据，业务处理完成后把这些数据处理后写入 IO。

又比如 cookie session，我们需要传递一个 session 的结构体，在请求开始的时候初始化、验证、提取 session 数据，业务处理过程中需要从 session 读取数据进行相应操作，业务处理最后可能要把 session 写回客户端。

Gear 框架创新性的提出了 `Any interface` 这一解决方案，它由三部分组成：

```go
// Any interface is used by ctx.Any.
type Any interface {
  New(ctx *Context) (interface{}, error)
}
```

```go
func (ctx *Context) Any(any interface{}) (val interface{}, err error) {
  var ok bool
  if val, ok = ctx.kv[any]; !ok {
    switch v := any.(type) {
    case Any:
      if val, err = v.New(ctx); err == nil {
        ctx.kv[any] = val
      }
    default:
      return nil, ErrAnyKeyNonExistent
    }
  }
  return
}
```

```go
// SetAny save a key, value pair on the ctx.
// Then we can use ctx.Any(key) to retrieve the value from ctx.
func (ctx *Context) SetAny(key, val interface{}) {
  ctx.kv[key] = val
}
```

其基本运行逻辑是，开发者可以通过 `ctx.SetAny(key, value)` 将任何键值对保存到 ctx 中，再通过 `ctx.Any(key)` 将 value 取出。如果 value 不存在，但 key 实现了 `Any interface`，那么其 `New` 方法将会运行，生成 value，并将 value 保存以备下次取值（但 `New` 返回错误时不会保存）。所以，对于一个中间件处理流，用实现了 `Any interface` 的 `key` 取值时，只会 `New` 一次，这个行为本身也相当于是惰性求值（真正需要时求值）。

以 [Gear-Auth](https://github.com/teambition/gear-auth) 中间件为例：

```go
func (a *Auth) New(ctx *gear.Context) (val interface{}, err error) {
  if token := a.ex(ctx); token != "" {
    val, err = a.j.Verify(token)
  }
  if val == nil {
    // create a empty jwt.Claims
    val = josejwt.Claims{}
    if err == nil {
      err = &gear.Error{Code: 401, Msg: "no token found"}
    }
  }
  ctx.SetAny(a, val)
  return
}

func (a *Auth) FromCtx(ctx *gear.Context) (josejwt.Claims, error) {
  val, err := ctx.Any(a)
  return val.(josejwt.Claims), err
}
//  claims, err := auther.FromCtx(ctx)
//  fmt.Println(claims, err)
```

它实现了 `Any interface`，当我们第一次调用 `ctx.Any(a)` 时，`New` 方法的逻辑就会运行，它从 `gear.Context` 中读取 token 并验证提取内容到 `Claims`，如果验证出错，还会生成一个空的 `Claims` 并通过 `ctx.SetAny(a, val)` 设置进去。也就是说 `ctx.Any` 总会返回值，只是当错误存在时这个值是空值。并且，这里保证了读取 token 并验证提取内容这一行为只会运行一次。

这里还提供了 `FromCtx`，其实它只是 `ctx.Any` 的语法糖，把 `ctx.Any` 返回的 `interface{}` 类型强制转成可用的 `josejwt.Claims` 类型了。在实际开发中 `FromCtx` 这个语法糖非常好用。

这是相对复杂的一个 `Any` 用例，实际上 logging，gear-session，gear-tracing 甚至框架内的 `ctx.Param` 都使用了它。总之，当涉及到要在中间件之间进行状态传值时，就可以用它了，够强大，够安全。

## 请求数据的解析和验证

### ctx.ParseBody

Go 语言原生提供了基于 Form 的请求数据解析，但这显然无法实际场景的需求。对于常规 RESTful API 而言，`application/json` 类型的请求数据会更加常见。Gear 框架提供了 `BodyParser interface` 来实现请求数据的解析，并提供了 `BodyTemplate interface` 配合进行数据验证。

```go
type BodyParser interface {
  // Maximum allowed size for a request body
  MaxBytes() int64
  Parse(buf []byte, body interface{}, mediaType, charset string) error
}
```

其中 `MaxBytes` 应该返回最大允许的数据长度，当请求数据长度超出限制时会响应 `413 Request entity too large` 错误。`Parse` 则为自定义的解析逻辑。Gear 框架实现了一个默认的 `BodyParser interface` 结构：

```go
type DefaultBodyParser int64

func (d DefaultBodyParser) MaxBytes() int64 {
  return int64(d)
}

func (d DefaultBodyParser) Parse(buf []byte, body interface{}, mediaType, charset string) error {
  if len(buf) == 0 {
    return &Error{Code: http.StatusBadRequest, Msg: "request entity empty"}
  }
  switch mediaType {
  case MIMEApplicationJSON:
    return json.Unmarshal(buf, body)
  case MIMEApplicationXML:
    return xml.Unmarshal(buf, body)
  case MIMEApplicationForm:
    val, err := url.ParseQuery(string(buf))
    if err == nil {
      err = ValuesToStruct(val, body, "form")
    }
    return err
  }
  return &Error{Code: http.StatusUnsupportedMediaType, Msg: "unsupported media type"}
}
```

`DefaultBodyParser` 支持 JSON、Form 和 XML 的解析，这对于大部分场景而言已经足够使用了，并且它被框架默认启用，默认支持最大 2MB 的请求数据：

```go
app.Set(gear.SetBodyParser, gear.DefaultBodyParser(2<<20)) // 2MB
```

用于请求数据验证的 interface 定义则更简单：

```go
type BodyTemplate interface {
  Validate() error
}
```

把数据解析和数据验证结合在一起的是 `ctx.ParseBody(body BodyTemplate)`。它是惰性的：只有当你使用时才开始读取请求数据、解析数据并验证。下面是一个简单的使用示例——用户注册

```go
// https://github.com/seccom/kpass/blob/master/src/api/user.go#L32
type tplUserJoin struct {
  ID   string `json:"id" form:"id"`
  Pass string `json:"pass" form:"pass"`
}

func (t *tplUserJoin) Validate() error {
  if len(t.ID) < 3 {
    return &gear.Error{Code: 400, Msg: "invalid id, length of id should >= 3"}
  }
  if !util.IsHashString(t.Pass) {
    return &gear.Error{Code: 400, Msg: "invalid pass, pass should be hashed by sha256"}
  }
  return nil
}

// @Router POST /api/join
func (a *User) Join(ctx *gear.Context) error {
  body := new(tplUserJoin)
  if err := ctx.ParseBody(body); err != nil {
    return ctx.Error(err)
  }
  if err := a.models.User.CheckID(body.ID); err != nil {
    return ctx.Error(err)
  }
  // ... More logic
}
```

### ctx.ParseURL

## ctx.Cookies 便捷的处理 cookie 或 signed cookie

## 处理 goroutine data race

## gear.Router 高效且强大的路由处理

## Settings 应用设置

## Compress 内置的响应内容压缩

## logging 日志处理