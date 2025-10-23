/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"

	api "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"

	"github.com/external-secrets/external-secrets/providers/v1/yandex/common"
)

// Real/gRPC implementation of LockboxClient.
type grpcLockboxClient struct {
	lockboxPayloadClient api.PayloadServiceClient
}

// NewGrpcLockboxClient creates a new LockboxClient.
func NewGrpcLockboxClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (LockboxClient, error) {
	conn, err := ydxcommon.NewGrpcConnection(
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
		grpc.PerRPCCredentials(ydxcommon.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return payload.Entries, nil
}

func (c *grpcLockboxClient) GetExPayload(ctx context.Context, iamToken, folderID, name, versionID string) (map[string][]byte, error) {
	request := &api.GetExRequest{
		Identifier: &api.GetExRequest_FolderAndName{
			FolderAndName: &api.FolderAndName{
				FolderId:   folderID,
				SecretName: name,
			},
		},
		VersionId: versionID,
	}

	response, err := c.lockboxPayloadClient.GetEx(
		ctx,
		request,
		grpc.PerRPCCredentials(ydxcommon.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}

	return response.Entries, nil
}
