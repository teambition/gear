package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	if in.Name == "error" {
		return nil, errors.New("some error")
	}
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func TestGearMiddlewareGRPC(t *testing.T) {
	app := gear.New()
	rpc := grpc.NewServer()
	pb.RegisterGreeterServer(rpc, &server{})

	app.Use(New(rpc))
	app.Use(func(ctx *gear.Context) error {
		return ctx.HTML(200, "OK")
	})

	go func() {
		app.ListenTLS(":13334", "../../testdata/out/test.crt", "../../testdata/out/test.key")
	}()
	defer app.Close()

	cr, err := credentials.NewClientTLSFromFile("../../testdata/out/test.crt", "127.0.0.1")
	assert.Nil(t, err)
	conn, err := grpc.Dial("127.0.0.1:13334", grpc.WithTransportCredentials(cr))
	assert.Nil(t, err)

	defer conn.Close()
	client := pb.NewGreeterClient(conn)
	time.Sleep(time.Millisecond * 100)

	t.Run("should work with gRPC", func(t *testing.T) {
		assert := assert.New(t)

		res, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "gear"})
		assert.Nil(err)
		assert.Equal("Hello gear", res.Message)
	})

	t.Run("should work with gRPC when error", func(t *testing.T) {
		assert := assert.New(t)

		res, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "error"})
		assert.Nil(res)
		assert.Equal("rpc error: code = Unknown desc = some error", err.Error())
	})

	t.Run("should work with HTTPS", func(t *testing.T) {
		assert := assert.New(t)

		transport := &http.Transport{}
		tlsCfg := &tls.Config{InsecureSkipVerify: true}
		certificate, err := tls.LoadX509KeyPair("../../testdata/out/test.crt", "../../testdata/out/test.key")
		assert.Nil(err)
		tlsCfg.Certificates = []tls.Certificate{certificate}
		transport.TLSClientConfig = tlsCfg

		cli := &http.Client{Transport: transport}
		res, err := cli.Get("https://127.0.0.1:13334")
		assert.Nil(err)
		assert.Equal("HTTP/1.1", res.Proto)
		b, err := ioutil.ReadAll(res.Body)
		assert.Nil(err)
		assert.Equal("OK", string(b))
		res.Body.Close()
	})

	t.Run("should work with HTTP/2", func(t *testing.T) {
		assert := assert.New(t)

		transport := &http2.Transport{}
		tlsCfg := &tls.Config{InsecureSkipVerify: true}
		certificate, err := tls.LoadX509KeyPair("../../testdata/out/test.crt", "../../testdata/out/test.key")
		assert.Nil(err)
		tlsCfg.Certificates = []tls.Certificate{certificate}
		transport.TLSClientConfig = tlsCfg

		cli := &http.Client{Transport: transport}
		res, err := cli.Get("https://127.0.0.1:13334")
		assert.Nil(err)
		assert.Equal("HTTP/2.0", res.Proto)
		b, err := ioutil.ReadAll(res.Body)
		assert.Nil(err)
		assert.Equal("OK", string(b))
		res.Body.Close()
	})
}
