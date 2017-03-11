# Frequently Asked Questions for [Gear](https://github.com/teambition/gear)

## 1. 如何从源码自动生成 Swagger v2 的文档？

有一些 Go Web 框架的特性之一就是支持自动生成 Swagger 文档。Gear 框架并没有集成这个功能，因为它不是一个共性的、核心的需求。不过，我们开发了一个独立的工具来支持这个功能 —— [Swaggo](https://github.com/teambition/swaggo)。它是一个与框架无关的通过解析源码和注释来自动生成 Swagger v2 文档的工具，任何框架、任何 Go 项目都可以使用。

中文文档：https://github.com/teambition/swaggo/wiki/Declarative-Comments-Format

项目使用示例：https://github.com/seccom/kpass

## 2. Go 语言完整的应用项目结构最佳实践是怎样的？

由于特殊的包管理机制，用 Go 语言开发项目时项目结构会成为一个头疼的问题，不信可以 Google 一下 `golang application structure`。我们目前的最佳实践已经体现在 [KPass](https://github.com/seccom/kpass)。

首先，项目下的目录和文件命名一律小写，有必要的用下划线 `_`，目录名一律单数形式，目录下的包名尽量与 目录名一致。

`cmd` 目录存放用于编译可运行程序的 `main` 源码，它又分成了子级目录，主要是考虑一个项目可能有多种可运行程序。

`src` 目录放主要源码，集中在这个目录主要是为了方便查找和替换。`src` 目录下除了 `app.go`，`router.go` 这种顶层入口，又细分如下：

1. `util`，工具函数，不会依赖本项目的任何其它逻辑，只会被其它源码依赖；
1. `service`，对外部服务的封装，如对 mongodb、redis、zipkin 等 client 的封装，也不会依赖本项目 `util` 之外的任何其它逻辑，只会被其它源码依赖；
1. `schema`，数据模型，与数据库无关，也不会依赖本项目 `util` 之外的任何其它逻辑，只会被其它源码依赖；
1. `model`，通常依赖 `util`，`service` 和 `schema`，实现对数据库操作的主要逻辑，各个 model 内部无相互依赖；
1. `bll`，Business logic layer，通常依赖 `util`，`schema` 和 `model`，通过组合调用 `model` 实现更复杂的业务逻辑；
1. `api`，API 接口，通常依赖 `util`，`schema` 和 `bll`，挂载于 Router 上，直接受理客户端请求、提取和验证数据，调用 `bll` 层处理数据，然后响应给客户端；
1. `ctl`，Controller，类似 `api` 层，通常依赖 `util`，`schema` 和 `bll`，挂载于 Router 上，为客户端响应 View 页面；
1. 其它如 `auth`、`logger` 等则是一些带状态的被其它组件依赖的全局性组件。

与 `cmd`、`src` 平级的目录可能还会有：`web` 前端源码目录；`config` 配置文件目录；`vendor` go 依赖包目录；`dist` 编译后的可执行文件目录；`doc` 文档目录；`k8s` k8s 配置文件目录等。