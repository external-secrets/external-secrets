//Copyright External Secrets Inc. All Rights Reserved

package client

import (
	"context"

	api "github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
)

// Requests the content of the given certificate from Certificate Manager.
type CertificateManagerClient interface {
	GetCertificateContent(ctx context.Context, iamToken, certificateID, versionID string) (*api.GetCertificateContentResponse, error)
}
