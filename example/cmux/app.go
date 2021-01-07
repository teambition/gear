package main

import (
	"log"
	"net"

	"github.com/soheilhy/cmux"
	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// HelloSvc is used to implement helloworld.GreeterServer.
type HelloSvc struct{}

// SayHello implements helloworld.GreeterServer
func (s *HelloSvc) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
	// return nil, errors.New("some error")
}

func main() {
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatal(err)
	}

	// Create a cmux.
	m := cmux.New(l)

	// Match connections in order:
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any()) // Any means anything that is not yet matched.

	grpcS := grpc.NewServer()
	pb.RegisterGreeterServer(grpcS, &HelloSvc{})
	go grpcS.Serve(grpcL)

	app := gear.New()
	app.UseHandler(logging.Default())
	app.Use(func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>HTTP Server after cmux</h1>")
	})
	go app.ServeWithContext(context.Background(), httpL)

	// Start serving!
	m.Serve()
}
