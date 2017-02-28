# [Gear](https://github.com/teambition/gear) 框架设计考量

Gear 是由 [Teambition](https://www.teambition.com) 开发的一个轻量级的、专注于可组合扩展和高性能的 Go 语言 Web 服务框架。

Gear 框架在设计与实现的过程中充分参考了 Go 语言下多款知名 Web 框架，也参考了 Node.js 下的知名 Web 框架，汲取各方优秀因素，结合我们的开发实践，精心打磨而成。


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


## 3. 中间件的 **单向顺序** 流程控制和 **级联** 流程控制

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


## 5. 错误和异常处理

前面我们提到 Gear 的中间件定义允许返回一个 `error` 类型的错误，这个错误会被自动处理。其实 Gear 在错误和异常处理方面做了很多工作，完全可以覆盖 Web 业务的实际需求。

（待续。。。）

## After Hook 和 End hook 的应用

## ctx.Any 无限的 gear.Context 状态扩展能力

## ctx.ParseBody 请求 body 的解析和验证

## ctx.Cookies 便捷的处理 cookie 或 signed cookie

## 处理 goroutine data race

## gear.Router 高效且强大的路由处理

## Settings 应用设置

## Compress 内置的响应内容压缩

## logging 日志处理