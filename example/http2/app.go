package main

import (
	"net/http"

	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
	"github.com/teambition/gear/middleware/favicon"
)

// go run example/http2/app.go
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

	app.UseHandler(logging.Default())
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

// Visit: https://127.0.0.1:3000/
// Logging:
// 127.0.0.1 GET /hello.css 200 22 - 0.145 ms
// 127.0.0.1 GET / 200 157 - 0.167 ms
