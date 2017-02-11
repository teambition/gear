# [Gear](https://github.com/teambition/gear) 框架设计考量

Gear 是一个轻量级的、专注于可组合扩展和高性能的 Go 语言 Web 服务框架。

Gear 框架在设计与实现的过程中充分参考了 Go 语言下多款知名 Web 框架，也参考了 Node.js 下的知名 Web 框架，汲取各方优秀因素，结合我们的开发实践，历经几十次迭代打磨而成。

## 1. Server 底层基于原生 net/http 而不是 fasthttp

当我们计划使用并开始调研 Go 语言时，各种 Web 框架相关评测中 fasthttp 的优异表现让我们对 Go 有了很大的信心。但随着对 Go 的逐步深入学习和使用，当我们决定构建自己的 Web 框架时，还是选择了原生的 net/http。

一方面是当前新版 Go (1.7.x) 的 net/http 性能已经很好了，在我的 MBP 电脑上 Gear 框架与基于（最快的）基于 fasthttp 的 Iris 框架评测比分约为 `5:7`，已经不再是当初号称的 10倍、20倍差距。如果算上应用的业务逻辑的消耗，这个差距会变得更小，甚至可以忽略。并且可以预见，随着 Go 版本升级优化，net/http 的性能表现会越来越好。

另一方面从兼容性和生命力考量，随着 Go 语言的版本升级，性能之外，net/http 的功能也会越来越强大、越来越完善（比如 http2）。社区生态也会（也应该）往这个方向聚集。

## 2. 基于 context.Context 的 gear.Middleware 中间件机制

`context.Context` 是 Go 语言原生的用于解决异步流程控制的方案，它主要为异步控制流程提供了完全 cancel 的能力和域内传值 request-scoped value 的能力，net/http 底层就使用了它。

中间件模式则是被各语言生态下 Web 框架验证的可组合扩展的最佳模式，但仍然有 `级联` 和 `单向顺序` 两个截然不同的中间件流程模式，Gear 选择是单向顺序运行中间件的模式（后面讲解原因）。

一个 `http.HandlerFunc` 风格的 `gear.Middleware` 定义如下：

```go
func(ctx *gear.Context) error
```

或者是 `http.Handler` 风格的中间件：

```go
type Handler interface {
	Serve(ctx *gearContext) error
}
```

其中的 `gear.Context` 完全集成了 `context.Context`、`http.Request`、`http.ResponseWriter` 的能力，并且提供了很多核心、便捷的方法。开发者通过调用 `gear.Context` 即可快速实现各种 Web 业务逻辑。

`gear.Context` 充分利用了 `context.Context`，并实现了它的 interface，可以直接当成 `context.Context` 使用。也提供了 `ctx.WithCancel`, `ctx.WithDeadline`, `ctx.WithTimeout`, `ctx.WithValue` 等快速创建子级 `context.Context` 的便捷方法。还提供了 `ctx.Cancel` 主动完全退出中间件处理流程的方法。还有基于 WithTimeout Context 的中间件处理流程 timeout cancel 能力，甚至是 `ctx.Timing` 针对某个异步处理逻辑的 timeout cancel 能力等。

一个完整的 request - response 处理流程就是一系列 gear.Middleware 中间件运行的流程，中间件按照引入的顺序逐一、单向运行，每个中间件解决一个特定的需求，与其它任何中间件没有耦合。

当某一个中间件返回错误时（比如 400 参数验证错误），中间件处理流程就被中断，后续的中间件不再运行，Gear 应用会自动处理这个错误，并做出对应的 response 响应（也能由开发者自定错误响应结果）。开发者不再被错误处理压制，可以尽情的返回错误。

当某一个中间件调用了特定的方法（如 gear.Context 上的 `ctx.End`, `ctx.JSON`, `ctx.Error` 等，或者 Go 内置的 `http.Redirect`, `http.ServeContent` 等）直接导致了 response 响应时，中间件处理流程也被中断，后续的中间件不再运行。

中间件处理流程的中断称之为 `end` - 结束，对比正常终端，由 error 导致的中断除了会自动响应错误，还有一点不同是：中间件通过 `ctx.After` 注入的 hook 逻辑会被清理，不会运行（后面再详解）。

`gear.Context` 的能力会让你感到惊喜~

目前 Gear 内置了一些核心的中间件，包括 `gear.Router` 中间件，`gear/logging` 目录下的 `logging.Logger` 中间件，`gear/middleware` 目录下的 `cors`, `favicon`, `secure`, `static` 中间件。

另外 https://github.com/teambition 也有我们维护的一些 `gear-xxx` 的中间件，也非常欢迎开发者们参与 `gear-xxx` 中间件生态开发中来。

## 3. 错误和异常处理

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