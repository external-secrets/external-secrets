/*
Copyright © The ESO Authors

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

package secretmanager

// Metrics constants identify the GCP Secret Manager provider and API calls.
const (
	ProviderGCPSM                = "GCP/SecretManager"
	CallGCPSMGetSecret           = "GetSecret"
	CallGCPSMDeleteSecret        = "DeleteSecret"
	CallGCPSMCreateSecret        = "CreateSecret"
	CallGCPSMUpdateSecret        = "UpdateSecret"
	CallGCPSMAccessSecretVersion = "AccessSecretVersion"
	CallGCPSMAddSecretVersion    = "AddSecretVersion"
	CallGCPSMListSecrets         = "ListSecrets"
	CallGCPSMGenerateSAToken     = "GenerateServiceAccountToken"
	CallGCPSMGenerateIDBindToken = "GenerateIDBindToken"
	CallGCPSMGenerateAccessToken = "GenerateAccessToken"
)
