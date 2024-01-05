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
package ibm

import (
	"fmt"

	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
)

func extractSecretMetadata(response sm.SecretMetadataIntf, givenName *string, secretType string) (*string, *string, error) {
	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		metadata, ok := response.(*sm.ArbitrarySecretMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_Arbitrary, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil
	case sm.Secret_SecretType_UsernamePassword:
		metadata, ok := response.(*sm.UsernamePasswordSecretMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_UsernamePassword, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil

	case sm.Secret_SecretType_IamCredentials:
		metadata, ok := response.(*sm.IAMCredentialsSecretMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_IamCredentials, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil

	case sm.Secret_SecretType_ServiceCredentials:
		metadata, ok := response.(*sm.ServiceCredentialsSecretMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_ServiceCredentials, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil

	case sm.Secret_SecretType_ImportedCert:
		metadata, ok := response.(*sm.ImportedCertificateMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_ImportedCert, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil

	case sm.Secret_SecretType_PublicCert:
		metadata, ok := response.(*sm.PublicCertificateMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_PublicCert, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil

	case sm.Secret_SecretType_PrivateCert:
		metadata, ok := response.(*sm.PrivateCertificateMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_PrivateCert, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil

	case sm.Secret_SecretType_Kv:
		metadata, ok := response.(*sm.KVSecretMetadata)
		if !ok {
			return nil, nil, fmt.Errorf(errExtractingSecret, *givenName, sm.Secret_SecretType_Kv, "extractSecretMetadata")
		}
		return metadata.ID, metadata.Name, nil

	default:
		return nil, nil, fmt.Errorf("unknown secret type %s", secretType)
	}
}
