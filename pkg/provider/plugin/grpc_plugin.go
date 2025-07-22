package plugin

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/external-secrets/external-secrets/pkg/provider/plugin/proto"
)

// SecretsGRPCPlugin implements the plugin.GRPCPlugin interface
type SecretsGRPCPlugin struct {
	plugin.Plugin
	Impl proto.SecretsPluginServiceClient
}

// GRPCServer should return a gRPC server implementation of the plugin interface
func (p *SecretsGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// This is called when we're acting as a server, which we're not in this case
	return nil
}

// GRPCClient should return a gRPC client implementation of the plugin interface
func (p *SecretsGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return proto.NewSecretsPluginServiceClient(c), nil
}
