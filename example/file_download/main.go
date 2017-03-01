package main

import (
	"bytes"
	"time"

	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
	"github.com/teambition/gear/middleware/cors"
)

func main() {
	app := gear.New()

	app.UseHandler(logging.Default())
	app.Use(cors.New())
	app.Use(func(ctx *gear.Context) error {
		file := bytes.NewReader([]byte("Hello !"))
		return ctx.Attachment("统计数据.txt", time.Now(), file)
	})

	app.Error(app.Listen(":3000"))
}
