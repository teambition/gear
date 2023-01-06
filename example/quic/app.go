package main

// commented for test~
func main() {}

// import (
// 	"net/http"

// 	"github.com/lucas-clemente/quic-go/h2quic"
// 	"github.com/teambition/gear"
// 	"github.com/teambition/gear/logging"
// 	"github.com/teambition/gear/middleware/favicon"
// )

// func main() {

// 	const htmlBody = `
// <!DOCTYPE html>
// <html>
//   <head>
//     <link href="/hello.css" rel="stylesheet" type="text/css">
//   </head>
//   <body>
//     <h1>Hello, Gear!</h1>
//   </body>
// </html>`

// 	const pushBody = `
// h1 {
//   color: red;
// }
// `
// 	app := gear.New()

// 	app.UseHandler(logging.Default(true))
// 	app.Use(favicon.New("./testdata/favicon.ico"))

// 	router := gear.NewRouter()
// 	router.Get("/", func(ctx *gear.Context) error {
// 		ctx.Res.Push("/hello.css", &http.PushOptions{Method: "GET"})
// 		return ctx.HTML(200, htmlBody)
// 	})
// 	router.Get("/hello.css", func(ctx *gear.Context) error {
// 		ctx.Type("text/css")
// 		return ctx.End(200, []byte(pushBody))
// 	})
// 	router.Get("/json", func(ctx *gear.Context) error {
// 		return ctx.JSON(200, map[string]any{"name": "quic"})
// 	})
// 	app.UseHandler(router)
// 	app.Server.Addr = ":3000"
// 	quicServer := h2quic.Server{
// 		Server: app.Server,
// 	}
// 	app.Server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		quicServer.SetQuicHeaders(w.Header())
// 		app.ServeHTTP(w, r)
// 	})

// 	go app.Server.ListenAndServeTLS("./testdata/out/test.crt", "./testdata/out/test.key")

// 	quicServer.ListenAndServe()
// }

// Visit: https://127.0.0.1:3000/
