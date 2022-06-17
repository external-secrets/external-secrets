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

func (mc *AzureMockClient) SetSecret(ctx context.Context, vaultBaseURL string, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error) {
	return mc.setSecret(ctx, vaultBaseURL, secretName, parameters)
}

func (mc *AzureMockClient) ImportCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error) {
	return mc.importCertificate(ctx, vaultBaseURL, certificateName, parameters)
}

func (mc *AzureMockClient) ImportKey(ctx context.Context, vaultBaseURL string, keyName string, parameters keyvault.KeyImportParameters) (result keyvault.KeyBundle, err error) {
	return mc.importKey(ctx, vaultBaseURL, keyName, parameters)
}

func (mc *AzureMockClient) WithValue(serviceURL, secretName, secretVersion string, apiOutput keyvault.SecretBundle, err error) {
	if mc != nil {
		mc.getSecret = func(ctx context.Context, serviceURL, secretName, secretVersion string) (keyvault.SecretBundle, error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithKey(serviceURL, secretName, secretVersion string, apiOutput keyvault.KeyBundle, err error) {
	if mc != nil {
		mc.getKey = func(ctx context.Context, vaultBaseURL, keyName, keyVersion string) (keyvault.KeyBundle, error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithCertificate(serviceURL, secretName, secretVersion string, apiOutput keyvault.CertificateBundle, err error) {
	if mc != nil {
		mc.getCertificate = func(ctx context.Context, vaultBaseURL, keyName, keyVersion string) (keyvault.CertificateBundle, error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithImportCertificate(apiOutput keyvault.CertificateBundle, err error) {
	if mc != nil {
		mc.importCertificate = func(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (keyvault.CertificateBundle, error) {
			return apiOutput, err
		}
	}
}

func (mc *AzureMockClient) WithImportKey(output keyvault.KeyBundle, err error) {
	if mc != nil {
		mc.importKey = func(ctx context.Context, vaultBaseURL string, keyName string, parameters keyvault.KeyImportParameters) (keyvault.KeyBundle, error) {
			return output, err
		}
	}
}

func (mc *AzureMockClient) WithSetSecret(output keyvault.SecretBundle, err error) {
	if mc != nil {
		mc.setSecret = func(ctx context.Context, vaultBaseURL string, secretName string, parameters keyvault.SecretSetParameters) (keyvault.SecretBundle, error) {
			return output, err
		}
	}
}

func (mc *AzureMockClient) WithList(serviceURL string, apiOutput keyvault.SecretListResultIterator, err error) {
	if mc != nil {
		mc.getSecretsComplete = func(ctx context.Context, vaultBaseURL string, maxresults *int32) (keyvault.SecretListResultIterator, error) {
			return apiOutput, err
		}
	}
}
