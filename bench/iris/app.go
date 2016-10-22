package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kataras/iris"
)

// `go run iris/app.go -m100`
// `wrk 'http://localhost:3333/?foo[bar]=baz' -d 10 -c 100 -t 4`
func main() {
	count := 0
	if len(os.Args) > 1 {
		m, err := strconv.ParseInt(os.Args[1][2:], 10, 64) // "-m10" -> 10
		if err == nil {
			count = int(m)
		}
	}

	for i := 0; i < count; i++ {
		iris.UseFunc(func(ctx *iris.Context) {
			ctx.Next()
		})
	}

	iris.Get("/", func(ctx *iris.Context) {
		ctx.HTML(200, "<h1>Hello, Iris!</h1>")
	})

	fmt.Printf("Iris start with %d middleware\n", count)
	iris.Listen(":3333")
}
