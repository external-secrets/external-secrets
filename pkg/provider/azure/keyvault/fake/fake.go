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

package fake

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
)

type AzureMockClient struct {
	getKey             func(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string) (result keyvault.KeyBundle, err error)
	getSecret          func(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	getSecretsComplete func(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error)
	getCertificate     func(ctx context.Context, vaultBaseURL string, certificateName string, certificateVersion string) (result keyvault.CertificateBundle, err error)
	setSecret          func(ctx context.Context, vaultBaseURL string, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error)
	importCertificate  func(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error)
	importKey          func(ctx context.Context, vaultBaseURL string, keyName string, parameters keyvault.KeyImportParameters) (result keyvault.KeyBundle, err error)
	deleteCertificate  func(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.DeletedCertificateBundle, err error)
	deleteKey          func(ctx context.Context, vaultBaseURL string, keyName string) (result keyvault.DeletedKeyBundle, err error)
	deleteSecret       func(ctx context.Context, vaultBaseURL string, secretName string) (result keyvault.DeletedSecretBundle, err error)
}

func (mc *AzureMockClient) GetSecret(ctx context.Context, vaultBaseURL, secretName, secretVersion string) (result keyvault.SecretBundle, err error) {
	return mc.getSecret(ctx, vaultBaseURL, secretName, secretVersion)
}

func (mc *AzureMockClient) GetCertificate(ctx context.Context, vaultBaseURL, certificateName, certificateVersion string) (result keyvault.CertificateBundle, err error) {
	return mc.getCertificate(ctx, vaultBaseURL, certificateName, certificateVersion)
}

func (mc *AzureMockClient) GetKey(ctx context.Context, vaultBaseURL, keyName, keyVersion string) (result keyvault.KeyBundle, err error) {
	return mc.getKey(ctx, vaultBaseURL, keyName, keyVersion)
}

func (mc *AzureMockClient) GetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error) {
	return mc.getSecretsComplete(ctx, vaultBaseURL, maxresults)
}

func (mc *AzureMockClient) SetSecret(ctx context.Context, vaultBaseURL, secretName string, parameters keyvault.SecretSetParameters) (keyvault.SecretBundle, error) {
	return mc.setSecret(ctx, vaultBaseURL, secretName, parameters)
}

func (mc *AzureMockClient) ImportCertificate(ctx context.Context, vaultBaseURL, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error) {
	return mc.importCertificate(ctx, vaultBaseURL, certificateName, parameters)
}

func (mc *AzureMockClient) ImportKey(ctx context.Context, vaultBaseURL, keyName string, parameters keyvault.KeyImportParameters) (result keyvault.KeyBundle, err error) {
	return mc.importKey(ctx, vaultBaseURL, keyName, parameters)
}

func (mc *AzureMockClient) DeleteKey(ctx context.Context, vaultBaseURL, keyName string) (keyvault.DeletedKeyBundle, error) {
	return mc.deleteKey(ctx, vaultBaseURL, keyName)
}

func (mc *AzureMockClient) DeleteSecret(ctx context.Context, vaultBaseURL, secretName string) (keyvault.DeletedSecretBundle, error) {
	return mc.deleteSecret(ctx, vaultBaseURL, secretName)
}

func (mc *AzureMockClient) DeleteCertificate(ctx context.Context, vaultBaseURL, certificateName string) (keyvault.DeletedCertificateBundle, error) {
	return mc.deleteCertificate(ctx, vaultBaseURL, certificateName)
}

func (mc *AzureMockClient) WithValue(_, _, _ string, apiOutput keyvault.SecretBundle, err error) {
	if mc != nil {
		mc.getSecret = func(_ context.Context, _, _, _ string) (result keyvault.SecretBundle, retErr error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithKey(_, _, _ string, apiOutput keyvault.KeyBundle, err error) {
	if mc != nil {
		mc.getKey = func(_ context.Context, _, _, _ string) (result keyvault.KeyBundle, retErr error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithCertificate(_, _, _ string, apiOutput keyvault.CertificateBundle, err error) {
	if mc != nil {
		mc.getCertificate = func(_ context.Context, _, _, _ string) (result keyvault.CertificateBundle, retErr error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithImportCertificate(apiOutput keyvault.CertificateBundle, err error) {
	if mc != nil {
		mc.importCertificate = func(_ context.Context, _ string, _ string, _ keyvault.CertificateImportParameters) (keyvault.CertificateBundle, error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithImportKey(output keyvault.KeyBundle, err error) {
	if mc != nil {
		mc.importKey = func(_ context.Context, _ string, _ string, _ keyvault.KeyImportParameters) (keyvault.KeyBundle, error) {
			return output, err
		}
	}
}

func (mc *AzureMockClient) WithSetSecret(output keyvault.SecretBundle, err error) {
	if mc != nil {
		mc.setSecret = func(_ context.Context, _, _ string, _ keyvault.SecretSetParameters) (keyvault.SecretBundle, error) {
			return output, err
		}
	}
}

func (mc *AzureMockClient) WithDeleteSecret(output keyvault.DeletedSecretBundle, err error) {
	if mc != nil {
		mc.deleteSecret = func(_ context.Context, _, _ string) (keyvault.DeletedSecretBundle, error) {
			return output, err
		}
	}
}

func (mc *AzureMockClient) WithDeleteCertificate(output keyvault.DeletedCertificateBundle, err error) {
	if mc != nil {
		mc.deleteCertificate = func(_ context.Context, _, _ string) (keyvault.DeletedCertificateBundle, error) {
			return output, err
		}
	}
}

func (mc *AzureMockClient) WithDeleteKey(output keyvault.DeletedKeyBundle, err error) {
	if mc != nil {
		mc.deleteKey = func(_ context.Context, _, _ string) (keyvault.DeletedKeyBundle, error) {
			return output, err
		}
	}
}

func (mc *AzureMockClient) WithList(_ string, apiOutput keyvault.SecretListResultIterator, err error) {
	if mc != nil {
		mc.getSecretsComplete = func(_ context.Context, _ string, _ *int32) (keyvault.SecretListResultIterator, error) {
			return apiOutput, err
		}
	}
}
