//Copyright External Secrets Inc. All Rights Reserved

package client

import (
	"context"

	api "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
)

// Real/gRPC implementation of LockboxClient.
type grpcLockboxClient struct {
	lockboxPayloadClient api.PayloadServiceClient
}

func NewGrpcLockboxClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (LockboxClient, error) {
	conn, err := common.NewGrpcConnection(
		ctx,
		apiEndpoint,
		"lockbox-payload", // taken from https://api.cloud.yandex.net/endpoints
		authorizedKey,
		caCertificate,
	)
	if err != nil {
		return nil, err
	}
	return &grpcLockboxClient{api.NewPayloadServiceClient(conn)}, nil
}

func (c *grpcLockboxClient) GetPayloadEntries(ctx context.Context, iamToken, secretID, versionID string) ([]*api.Payload_Entry, error) {
	payload, err := c.lockboxPayloadClient.Get(
		ctx,
		&api.GetPayloadRequest{
			SecretId:  secretID,
			VersionId: versionID,
		},
		grpc.PerRPCCredentials(common.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return payload.Entries, nil
}
