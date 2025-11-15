package certificatemanager

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	awsutil "github.com/external-secrets/external-secrets/pkg/provider/aws/util"
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

	// 1. Describe the certificate (optional: for validation/logging)
	_, err := cm.client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
		CertificateArn: aws.String(certARN),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe ACM certificate %s: %w", certARN, err)
	}

	// 2. Export the certificate (requires exportable cert)
	exportOut, err := cm.client.ExportCertificate(ctx, &acm.ExportCertificateInput{
		CertificateArn: aws.String(certARN),
		// Optional: add Passphrase if you later include it in provider config
	})
	if err != nil {
		return nil, awsutil.SanitizeErr(err)
	}

	// 3. Build the PEM payload
	// Kubernetes ExternalSecret expects a single []byte blob, so we concatenate.
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
	// Deletion of ACM certificates is not supported in this provider implementation.
	return fmt.Errorf("deletion of ACM certificates is not supported")
}

// Validate checks if the provider is configured correctly.
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
	// Existence check of ACM certificates is not supported in this provider implementation.
	return false, fmt.Errorf("SecretExists is not supported for CertificateManager provider")
}

func (cm *CertificateManager) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	return fmt.Errorf("pushing secrets to ACM is not supported")
}

func (cm *CertificateManager) Close(_ context.Context) error {
	return nil
}
