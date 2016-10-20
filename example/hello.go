package main

import "github.com/teambition/gweb"

func main() {
	app := gweb.New()
	router := gweb.NewRouter()
	app.Use(func(c *gweb.Context) (err error) {
		// fmt.Println(c.Method)
		// fmt.Println(c.Path)
		c.Status(200)
		c.Html("<h1>Hello! </h1>")
		// or
		// fmt.Fprintf(c.Res, "<h1>Hello! </h1>")
		return
	})
	app.UseHandler(router)
	app.Listen(":3000")
}
