/*
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

package client

import (
	"context"
	"fmt"
	"github.com/keeper-security/secrets-manager-go/core/logger"
	api "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
)

// Real/gRPC implementation of LockboxClient.
type grpcLockboxClient struct {
	lockboxPayloadClient api.PayloadServiceClient
	lockboxSecretClient  api.SecretServiceClient
}

func NewGrpcLockboxClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (LockboxClient, error) {
	conn, err := common.NewGrpcConnection(
		// Required lockbox.payloadViewer role
		ctx,
		apiEndpoint,
		"lockbox-payload", // taken from https://api.cloud.yandex.net/endpoints
		authorizedKey,
		caCertificate,
	)
	if err != nil {
		return nil, err
	}
	conn2, err := common.NewGrpcConnection(
		// Required lockbox.viewer role
		ctx,
		apiEndpoint,
		"lockbox", // taken from https://api.cloud.yandex.net/endpoints
		authorizedKey,
		caCertificate,
	)
	if err != nil {
		return nil, err
	}
	return &grpcLockboxClient{
		api.NewPayloadServiceClient(conn),
		api.NewSecretServiceClient(conn2),
	}, nil
}

func (c *grpcLockboxClient) GetPayloadEntries(ctx context.Context, iamToken, folderID, secretIDOrName, versionID string) ([]*api.Payload_Entry, error) {
	// If the folderID is provided in the SecretStore, we can attempt to retrieve the secret by its name
	if folderID != "" {
		payloadEntry, err := c.GetSecretByName(ctx, iamToken, folderID, secretIDOrName, versionID)
		if err != nil {
			logger.Error(fmt.Sprintf("Method done with error - %s. Properties are :method is %s, folderId: %s, versionId: %s, secretIdOrName: %s", err.Error(), "GetSecretByName", folderID, versionID, secretIDOrName))
			return nil, err
		}
		logger.Debug(fmt.Sprintf("Method done with success. Properties are: method: %s, folderId: %s, versionId: %s, secretIdOrName: %s", "GetSecretByName", folderID, versionID, secretIDOrName))

		return payloadEntry, nil
	}

	// If the folderID is not provided in the SecretStore, we can attempt to retrieve the secret by its ID
	payloadEntry, err := c.GetSecretById(ctx, iamToken, secretIDOrName, versionID)
	if err != nil {
		logger.Error(fmt.Sprintf("Method done with error - %s. Properties are :method is %s, folderId: %s, versionId: %s, secretIdOrName: %s", err.Error(), "GetSecretById", folderID, versionID, secretIDOrName))
		return nil, err
	}
	logger.Debug(fmt.Sprintf("Method done with success. Properties are: method: %s, folderId: %s, versionId: %s, secretIdOrName: %s", "GetSecretById", folderID, versionID, secretIDOrName))

	return payloadEntry, nil
}

func (c *grpcLockboxClient) GetSecretByName(ctx context.Context, iamToken, folderID, secretIDOrName, versionID string) ([]*api.Payload_Entry, error) {
	response, err := c.lockboxPayloadClient.GetEx(
		ctx,
		&api.GetExRequest{
			Identifier: &api.GetExRequest_FolderAndName{
				FolderAndName: &api.FolderAndName{
					FolderId:   folderID,
					SecretName: secretIDOrName,
				},
			},
			VersionId: versionID,
		},
		grpc.PerRPCCredentials(common.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	// Convert the response of GetEx method to api.Payload
	payload := &api.Payload{VersionId: response.VersionId, Entries: make([]*api.Payload_Entry, len(response.Entries))}
	for key, value := range response.Entries {
		payload.Entries = append(payload.Entries, &api.Payload_Entry{
			Key:   key,
			Value: &api.Payload_Entry_TextValue{TextValue: string(value)},
		})
	}
	return payload.Entries, nil
}

func (c *grpcLockboxClient) GetSecretById(ctx context.Context, iamToken, secretID, versionID string) ([]*api.Payload_Entry, error) {
	// Try to get the secret by ID
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
