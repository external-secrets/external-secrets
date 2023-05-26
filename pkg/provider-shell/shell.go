/*
Copyright © 2022 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package shell

import (
	"context"
	"fmt"
	"log"
	"net"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedSecretsClientServer
	secretsClient esapi.SecretsClient
}

func RunServer(secretsClient esapi.SecretsClient) error {
	pluginServer := &Server{
		secretsClient: secretsClient,
	}
	lis, err := net.Listen("unix", "/tmp/plugin.sock")
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()
	s := grpc.NewServer()
	pb.RegisterSecretsClientServer(s, pluginServer)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return nil
}

func (s *Server) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretReply, error) {
	ref := esapi.ExternalSecretDataRemoteRef{
		Key:                req.RemoteRef.Key,
		MetadataPolicy:     esapi.ExternalSecretMetadataPolicy(req.RemoteRef.MetadataPolicy),
		Property:           req.RemoteRef.Property,
		Version:            req.RemoteRef.Version,
		ConversionStrategy: esapi.ExternalSecretConversionStrategy(req.RemoteRef.ConversionStrategy),
		DecodingStrategy:   esapi.ExternalSecretDecodingStrategy(req.RemoteRef.DecodingStrategy),
	}
	secret, err := s.secretsClient.GetSecret(ctx, ref)
	if err != nil {
		// TODO: handle NoSecret error on the client side
		return &pb.GetSecretReply{
			Error: err.Error(),
		}, nil
	}
	return &pb.GetSecretReply{
		Secret: secret,
	}, nil
}
