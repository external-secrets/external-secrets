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

	api "github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
)

// Real/gRPC implementation of CertificateManagerClient.
type grpcCertificateManagerClient struct {
	certificateContentServiceClient api.CertificateContentServiceClient
	certificateServiceClient        api.CertificateServiceClient
}

func NewGrpcCertificateManagerClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (CertificateManagerClient, error) {
	conn, err := common.NewGrpcConnection(
		// Requires: certificate-manager.certificates.downloader role
		ctx,
		apiEndpoint,
		"certificate-manager-data", // taken from https://api.cloud.yandex.net/endpoints
		authorizedKey,
		caCertificate,
	)
	if err != nil {
		return nil, err
	}
	conn2, err := common.NewGrpcConnection(
		// Requires: certificate-manager.viewer role
		ctx,
		apiEndpoint,
		"certificate-manager", // taken from https://api.cloud.yandex.net/endpoints
		authorizedKey,
		caCertificate,
	)
	if err != nil {
		return nil, err
	}
	return &grpcCertificateManagerClient{
		api.NewCertificateContentServiceClient(conn),
		api.NewCertificateServiceClient(conn2),
	}, nil
}

func (c *grpcCertificateManagerClient) GetCertificateIDByName(ctx context.Context, iamToken, folderID, certificateName string) (string, error) {
	list, err := c.certificateServiceClient.List(
		ctx,
		&api.ListCertificatesRequest{
			FolderId: folderID,
			PageSize: 500,
			View:     api.CertificateView_BASIC,
		},
		grpc.PerRPCCredentials(common.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return "", err
	}
	for _, cert := range list.Certificates {
		if strings.TrimSpace(cert.Name) == strings.TrimSpace(certificateName) {
			return cert.Id, nil
		}
	}
	return "", fmt.Errorf("certificate name %s not found in folder %s", certificateName, folderID)
}

func (c *grpcCertificateManagerClient) GetCertificateContent(ctx context.Context, iamToken, folderID, certificateID, _ string) (*api.GetCertificateContentResponse, error) {
	certificateID_ := certificateID

	// If the folderID is provided in the SecretStore, we can attempt to retrieve the secret by its name
	if folderID != "" {
		var err error
		certificateID_, err = c.GetCertificateIDByName(ctx, iamToken, folderID, certificateID)
		if err != nil {
			if len(certificateID) == 20 {
				certificateID_ = certificateID // Second chance to get the secret by ID
			} else {
				return nil, err
			}
		}
	}

	response, err := c.certificateContentServiceClient.Get(
		ctx,
		&api.GetCertificateContentRequest{
			CertificateId: certificateID_,
		},
		grpc.PerRPCCredentials(common.PerRPCCredentials{IamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}
