package main

import (
	"log"
	"strings"

	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
	// return nil, errors.New("some error")
}

// go run example/grpc_server/app.go
// Visit: https://127.0.0.1:3000/ or go run example/grpc_client/app.go
func main() {

	rpc := grpc.NewServer()
	pb.RegisterGreeterServer(rpc, &server{})

	app := gear.New()
	app.UseHandler(logging.Default())

	app.Use(func(ctx *gear.Context) error {
		// "application/grpc", "application/grpc+proto"
		if strings.HasPrefix(ctx.GetHeader(gear.HeaderContentType), "application/grpc") {
			rpc.ServeHTTP(ctx.Res, ctx.Req)
			ctx.End(204) // Must end with 204 to handle rpc error
		}
		return nil
	})
	app.Use(func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>gRPC</h1>")
	})

	log.Fatalln(app.ListenTLS(":3000", "./testdata/out/test.crt", "./testdata/out/test.key"))
}
