//Copyright External Secrets Inc. All Rights Reserved

package fake

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/secrets"
	"github.com/oracle/oci-go-sdk/v65/vault"
)

type OracleMockVaultClient struct {
	SecretSummaries []vault.SecretSummary
	CreatedCount    int
	UpdatedCount    int
	DeletedCount    int
}

func (o *OracleMockVaultClient) ListSecrets(_ context.Context, _ vault.ListSecretsRequest) (response vault.ListSecretsResponse, err error) {
	return vault.ListSecretsResponse{
		Items: o.SecretSummaries,
	}, nil
}

func (o *OracleMockVaultClient) CreateSecret(_ context.Context, _ vault.CreateSecretRequest) (response vault.CreateSecretResponse, err error) {
	o.CreatedCount++
	return vault.CreateSecretResponse{}, nil
}

func (o *OracleMockVaultClient) UpdateSecret(_ context.Context, _ vault.UpdateSecretRequest) (response vault.UpdateSecretResponse, err error) {
	o.UpdatedCount++
	return vault.UpdateSecretResponse{}, nil
}

func (o *OracleMockVaultClient) ScheduleSecretDeletion(_ context.Context, _ vault.ScheduleSecretDeletionRequest) (response vault.ScheduleSecretDeletionResponse, err error) {
	o.DeletedCount++
	return vault.ScheduleSecretDeletionResponse{}, nil
}

type OracleMockClient struct {
	getSecret     func(ctx context.Context, request secrets.GetSecretBundleByNameRequest) (response secrets.GetSecretBundleByNameResponse, err error)
	SecretBundles map[string]secrets.SecretBundle
}

func (mc *OracleMockClient) GetSecretBundleByName(ctx context.Context, request secrets.GetSecretBundleByNameRequest) (response secrets.GetSecretBundleByNameResponse, err error) {
	if mc.SecretBundles != nil {
		if bundle, ok := mc.SecretBundles[*request.SecretName]; ok {
			return secrets.GetSecretBundleByNameResponse{
				SecretBundle: bundle,
			}, nil
		}
		return secrets.GetSecretBundleByNameResponse{}, &ServiceError{Code: 404}
	}
	return mc.getSecret(ctx, request)
}

func (mc *OracleMockClient) WithValue(_ secrets.GetSecretBundleByNameRequest, output secrets.GetSecretBundleByNameResponse, err error) {
	if mc != nil {
		mc.getSecret = func(ctx context.Context, paramReq secrets.GetSecretBundleByNameRequest) (secrets.GetSecretBundleByNameResponse, error) {
			return output, err
		}
	}
}

type ServiceError struct {
	Code int
}

func (s *ServiceError) Error() string {
	return ""
}
func (s *ServiceError) GetHTTPStatusCode() int {
	return s.Code
}

func (s *ServiceError) GetMessage() string {
	return ""
}

func (s *ServiceError) GetCode() string {
	return ""
}

func (s *ServiceError) GetOpcRequestID() string {
	return ""
}
