package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// `go run gin/app.go -m100`
// `wrk 'http://localhost:3333/?foo[bar]=baz' -d 10 -c 100 -t 4`
func main() {
	router := gin.New()

	count := 0
	if len(os.Args) > 1 {
		m, err := strconv.ParseInt(os.Args[1][2:], 10, 64) // "-m10" -> 10
		if err == nil {
			count = int(m)
		}
	}

	for i := 0; i < count; i++ {
		router.Use(func(ctx *gin.Context) {
			ctx.Next()
		})
	}

	router.GET("/", func(ctx *gin.Context) {
		ctx.String(200, "<h1>Hello, Gin!</h1>")
	})

	fmt.Printf("Gin start with %d middleware\n", count)
	router.Run(":3333")
}
