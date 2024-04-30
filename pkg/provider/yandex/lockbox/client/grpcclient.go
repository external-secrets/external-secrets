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
	"strings"

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

func (c *grpcLockboxClient) GetSecretIDByName(ctx context.Context, iamToken, folderID, secretName string) (string, error) {
	list, err := c.lockboxSecretClient.List(
		ctx,
		&api.ListSecretsRequest{
			FolderId: folderID,
			PageSize: 500,
		},
		grpc.PerRPCCredentials(common.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return "", err
	}
	for _, secret := range list.Secrets {
		if strings.TrimSpace(secret.Name) == strings.TrimSpace(secretName) {
			return secret.Id, nil
		}
	}
	return "", fmt.Errorf("secret name %s not found in folder %s", secretName, folderID)
}

func (c *grpcLockboxClient) GetPayloadEntries(ctx context.Context, iamToken, folderID, secretID, versionID string) ([]*api.Payload_Entry, error) {

	secretID_ := secretID

	// If folderID is provided in SecretStore, we can try to get secret by its name
	if folderID != "" {
		var err error
		secretID_, err = c.GetSecretIDByName(ctx, iamToken, folderID, secretID)
		if err != nil {
			if len(secretID) == 20 && strings.HasPrefix(secretID, "e6q") {
				secretID_ = secretID // Another chance to get the secret by ID
			} else {
				return nil, err
			}
		}
	}

	payload, err := c.lockboxPayloadClient.Get(
		ctx,
		&api.GetPayloadRequest{
			SecretId:  secretID_,
			VersionId: versionID,
		},
		grpc.PerRPCCredentials(common.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return payload.Entries, nil
}
