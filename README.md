gweb
=====
A pithy and powerful web framework for Go, work with context.Context and middleware, like koajs/toajs.

## Demo
```go
package main

import (
	"fmt"

	"github.com/zensh/gweb"
)

func main() {
	app := gweb.New()
	app.Use(func(c *gweb.Context) (err error) {
		fmt.Println(c.Method)
		fmt.Println(c.Path)
		c.Status(200)
		c.Html("<h1>Hello! </h1>")
		// or
		// fmt.Fprintf(c.Res, "<h1>Hello! </h1>")
		return
	})
	app.Listen(":3000")
}
```
