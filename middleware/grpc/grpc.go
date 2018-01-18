package grpc

import (
	"net/http"
	"strings"

	"github.com/teambition/gear"
)

// New creates a middleware with gRPC server to Handle gRPC requests.
func New(srv http.Handler) gear.Middleware {
	return func(ctx *gear.Context) error {
		// "application/grpc", "application/grpc+proto"
		if strings.HasPrefix(ctx.GetHeader(gear.HeaderContentType), "application/grpc") {
			srv.ServeHTTP(ctx.Res, ctx.Req)
			ctx.End(204) // Must end with 204 to handle rpc error
		}
		return nil
	}
}
