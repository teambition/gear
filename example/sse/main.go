// Golang HTML5 Server Side Events Example

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
)

// go run example/hello/main.go
func main() {
	app := gear.New()

	// Add logging middleware
	app.UseHandler(logging.Default(true))

	// Add router middleware
	router := gear.NewRouter()

	// try: http://127.0.0.1:3000/hello
	router.Get("/hello", func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
	})

	messageChan := make(chan string)
	go func() {
		i := 0
		for {
			time.Sleep(time.Second * 3)
			i++
			messageChan <- fmt.Sprintf("Hello %d", i)
		}
	}()

	// try: http://127.0.0.1:3000/events
	router.Get("/events", func(ctx *gear.Context) error {
		// https://github.com/kljensen/golang-html5-sse-example/blob/1e2f75f9ea91b900a42ac373c59dc7c8388dfb2a/server.go
		// Set the headers related to event streaming.
		ctx.SetHeader("Content-Type", "text/event-stream")
		ctx.SetHeader("Cache-Control", "no-cache")
		ctx.SetHeader("Connection", "keep-alive")

		notify := ctx.Res.CloseNotify()
		go func() {
			<-notify
			log.Println("HTTP connection just closed.")
		}()

		for {
			// Read from our messageChan.
			msg, open := <-messageChan
			if !open {
				// If our messageChan was closed, this means that the client has
				// disconnected.
				fmt.Fprint(ctx.Res, "message chan closed\n\n")
				ctx.Res.Flush()
				break
			}

			fmt.Fprintf(ctx.Res, "data: Message: %s\n\n", msg)
			// Flush the response.  This is only possible if the repsonse supports streaming.
			ctx.Res.Flush()
		}

		return nil
	})

	app.UseHandler(router)
	app.Error(app.Listen(":3000"))
}
