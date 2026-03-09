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

	api "github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"

	ydxcommon "github.com/external-secrets/external-secrets/providers/v1/yandex/common"
)

// Real/gRPC implementation of CertificateManagerClient.
type grpcCertificateManagerClient struct {
	certificateContentServiceClient api.CertificateContentServiceClient
}

// NewGrpcCertificateManagerClient creates a new gRPC client for Yandex Certificate Manager.
func NewGrpcCertificateManagerClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (CertificateManagerClient, error) {
	conn, err := ydxcommon.NewGrpcConnection(
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

func (c *grpcCertificateManagerClient) GetCertificateContent(ctx context.Context, iamToken, certificateID, versionID string) (*api.GetCertificateContentResponse, error) {
	response, err := c.certificateContentServiceClient.Get(
		ctx,
		&api.GetCertificateContentRequest{
			CertificateId: certificateID,
			VersionId:     versionID,
		},
		grpc.PerRPCCredentials(ydxcommon.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *grpcCertificateManagerClient) GetExCertificateContent(ctx context.Context, iamToken, folderID, name, versionID string) (*api.GetExCertificateContentResponse, error) {
	response, err := c.certificateContentServiceClient.GetEx(
		ctx,
		&api.GetExCertificateContentRequest{
			Identifier: &api.GetExCertificateContentRequest_FolderAndName{
				FolderAndName: &api.FolderAndName{
					FolderId:        folderID,
					CertificateName: name,
				},
			},
			VersionId: versionID,
		},
		grpc.PerRPCCredentials(ydxcommon.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}
