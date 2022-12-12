package requestid

import (
	"encoding/hex"
	"math/rand"

	"github.com/teambition/gear"
)

type Options struct {
	// Generator defines a function to generate the requestID.
	// Optional. Default generate uuid v4 string.
	Generator func() string
}

// New creates a middleware to return X-Request-ID header
//
//	package main
//
//	import (
//		"github.com/teambition/gear"
//		"github.com/teambition/gear/middleware/requestid"
//	)
//
//	func main() {
//		app := gear.New()
//		app.Use(requestid.New())
//		app.Use(func(ctx *gear.Context) error {
//			return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//		})
//		app.Error(app.Listen(":3000"))
//	}
func New(options ...Options) gear.Middleware {
	opts := Options{
		Generator: generator,
	}

	if len(options) > 0 {
		opts = options[0]
	}

	return func(ctx *gear.Context) error {
		rid := ctx.GetHeader(gear.HeaderXRequestID)
		if rid == "" {
			rid = opts.Generator()
		}

		ctx.SetHeader(gear.HeaderXRequestID, rid)

		return nil
	}
}

// uuid version 4
type uuidv4 [16]byte

// String https://github.com/satori/go.uuid/blob/master/uuid.go
func (u uuidv4) String() string {
	buf := make([]byte, 36)

	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], u[10:])

	return string(buf)
}

func generator() string {
	id := uuidv4{}
	if _, err := rand.Read(id[:]); err != nil {
		return ""
	}

	// https://tools.ietf.org/html/rfc4122#section-4.1.3
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80

	return id.String()
}
