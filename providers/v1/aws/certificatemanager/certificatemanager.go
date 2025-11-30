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

package certificatemanager

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	awsutil "github.com/external-secrets/external-secrets/providers/v1/aws/util"
)

type CertificateManager struct {
	cfg          *aws.Config
	client       ACMInterface
	referentAuth bool
	prefix       string
}

// ACMInterface defines the subset of ACM API methods used by the provider.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/acm/
type ACMInterface interface {
	DescribeCertificate(ctx context.Context, input *acm.DescribeCertificateInput, opts ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error)
	ExportCertificate(ctx context.Context, input *acm.ExportCertificateInput, opts ...func(*acm.Options)) (*acm.ExportCertificateOutput, error)
	ListCertificates(ctx context.Context, input *acm.ListCertificatesInput, opts ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
	GetCertificate(ctx context.Context, input *acm.GetCertificateInput, opts ...func(*acm.Options)) (*acm.GetCertificateOutput, error)
	AddTagsToCertificate(ctx context.Context, input *acm.AddTagsToCertificateInput, opts ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)
	RemoveTagsFromCertificate(ctx context.Context, input *acm.RemoveTagsFromCertificateInput, opts ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)
}

func New(_ context.Context, cfg *aws.Config, prefix string, referentAuth bool) (*CertificateManager, error) {
	return &CertificateManager{
		cfg:          cfg,
		referentAuth: referentAuth,
		client: acm.NewFromConfig(*cfg, func(o *acm.Options) {
			o.EndpointResolverV2 = customEndpointResolver{}
		}),
		prefix: prefix,
	}, nil
}

func (cm *CertificateManager) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	certARN := ref.Key
	if certARN == "" {
		return nil, fmt.Errorf("certificate ARN must be specified in remoteRef.key")
	}

	_, err := cm.client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
		CertificateArn: aws.String(certARN),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe ACM certificate %s: %w", certARN, err)
	}

	exportOut, err := cm.client.ExportCertificate(ctx, &acm.ExportCertificateInput{
		CertificateArn: aws.String(certARN),
	})
	if err != nil {
		return nil, awsutil.SanitizeErr(err)
	}

	var result string
	if exportOut.Certificate != nil {
		result += *exportOut.Certificate + "\n"
	}
	if exportOut.CertificateChain != nil {
		result += *exportOut.CertificateChain + "\n"
	}
	if exportOut.PrivateKey != nil {
		result += *exportOut.PrivateKey + "\n"
	}

	if result == "" {
		return nil, fmt.Errorf("no data returned from ExportCertificate for %s (ensure it is exportable)", certARN)
	}

	return []byte(result), nil
}

func (cm *CertificateManager) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	return fmt.Errorf("DeleteSecret is not supported for CertificateManager provider")
}

func (cm *CertificateManager) Validate() (esv1.ValidationResult, error) {
	if cm.referentAuth {
		return esv1.ValidationResultUnknown, nil
	}

	_, err := cm.cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

func (cm *CertificateManager) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetAllSecrets is not supported for CertificateManager provider")
}

func (cm *CertificateManager) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetSecretMap is not supported for CertificateManager provider")
}

func (cm *CertificateManager) SecretExists(ctx context.Context, pushSecretRef esv1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("SecretExists is not supported for CertificateManager provider")
}

func (cm *CertificateManager) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	return fmt.Errorf("PushSecret is not supported for CertificateManager provider")
}

func (cm *CertificateManager) Close(_ context.Context) error {
	return nil
}
