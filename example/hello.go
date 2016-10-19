package main

import (
	"fmt"

	"github.com/teambition/gweb"
)

func main() {
	app := gweb.New()
	app.Use(func(c *gweb.Context) (err error) {
		fmt.Println(c.Method)
		fmt.Println(c.Path)
		c.Status(200)
		c.Html("<h1>Hello! </h1>")
		return
	})
	app.Listen(":3000")
}
