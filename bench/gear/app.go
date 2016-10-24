package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/teambition/gear"
)

// `go run gear/app.go -m100`
// `wrk 'http://localhost:3333/?foo[bar]=baz' -d 10 -c 100 -t 4`
func main() {
	app := gear.New()
	count := 0
	if len(os.Args) > 1 {
		m, err := strconv.ParseInt(os.Args[1][2:], 10, 64) // "-m10" -> 10
		if err == nil {
			count = int(m)
		}
	}

	for i := 0; i < count; i++ {
		app.Use(func(ctx *gear.Context) error {
			return nil
		})
	}

	router := gear.NewRouter("", true)
	router.Get("/", func(ctx *gear.Context) error {
		time.Sleep(time.Millisecond * 100)
		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
	})

	app.UseHandler(router)
	fmt.Printf("Gear start with %d middleware\n", count)
	app.Error(app.Listen(":3333"))
}
