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

	api "github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
)

// Real/gRPC implementation of CertificateManagerClient.
type grpcCertificateManagerClient struct {
	certificateContentServiceClient api.CertificateContentServiceClient
}

func NewGrpcCertificateManagerClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (CertificateManagerClient, error) {
	conn, err := common.NewGrpcConnection(
		ctx,
		apiEndpoint,
		"certificate-manager-data", // taken from https://api.cloud.yandex.net/endpoints
		authorizedKey,
		caCertificate,
	)
	if err != nil {
		return nil, err
	}
	return &grpcCertificateManagerClient{api.NewCertificateContentServiceClient(conn)}, nil
}

func (c *grpcCertificateManagerClient) GetCertificateContent(ctx context.Context, iamToken, certificateID, _ string) (*api.GetCertificateContentResponse, error) {
	response, err := c.certificateContentServiceClient.Get(
		ctx,
		&api.GetCertificateContentRequest{
			CertificateId: certificateID,
		},
		grpc.PerRPCCredentials(common.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}
