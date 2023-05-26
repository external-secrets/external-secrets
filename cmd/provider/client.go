package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "unix:///tmp/plugin.sock", "the address to connect to")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewSecretsClientClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res, err := c.GetSecret(ctx, &pb.GetSecretRequest{
		RemoteRef: &pb.RemoteRef{
			Key: "foo",
		},
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("secret=%s, err=%s", string(res.Secret), res.Error)
}
