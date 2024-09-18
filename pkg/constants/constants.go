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

package constants

const (
	ProviderAWSSM                = "AWS/SecretsManager"
	CallAWSSMGetSecretValue      = "GetSecretValue"
	CallAWSPSGetParametersByPath = "GetParametersByPath"
	CallAWSSMDescribeSecret      = "DescribeSecret"
	CallAWSSMDeleteSecret        = "DeleteSecret"
	CallAWSSMCreateSecret        = "CreateSecret"
	CallAWSSMPutSecretValue      = "PutSecretValue"
	CallAWSSMListSecrets         = "ListSecrets"

	ProviderAWSPS                = "AWS/ParameterStore"
	CallAWSPSGetParameter        = "GetParameter"
	CallAWSPSPutParameter        = "PutParameter"
	CallAWSPSDeleteParameter     = "DeleteParameter"
	CallAWSPSDescribeParameter   = "DescribeParameter"
	CallAWSPSListTagsForResource = "ListTagsForResource"

	ProviderAzureKV              = "Azure/KeyVault"
	CallAzureKVGetKey            = "GetKey"
	CallAzureKVDeleteKey         = "DeleteKey"
	CallAzureKVImportKey         = "ImportKey"
	CallAzureKVGetSecret         = "GetSecret"
	CallAzureKVGetSecrets        = "GetSecrets"
	CallAzureKVDeleteSecret      = "DeleteSecret"
	CallAzureKVGetCertificate    = "GetCertificate"
	CallAzureKVDeleteCertificate = "DeleteCertificate"
	CallAzureKVImportCertificate = "ImportCertificate"

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

	ProviderHCVault            = "HashiCorp/Vault"
	CallHCVaultLogin           = "Login"
	CallHCVaultRevokeSelf      = "RevokeSelf"
	CallHCVaultLookupSelf      = "LookupSelf"
	CallHCVaultReadSecretData  = "ReadSecretData"
	CallHCVaultWriteSecretData = "WriteSecretData"
	CallHCVaultDeleteSecret    = "DeleteSecret"
	CallHCVaultListSecrets     = "ListSecrets"

	ProviderKubernetes                         = "Kubernetes"
	CallKubernetesGetSecret                    = "GetSecret"
	CallKubernetesListSecrets                  = "ListSecrets"
	CallKubernetesCreateSecret                 = "CreateSecret"
	CallKubernetesDeleteSecret                 = "DeleteSecret"
	CallKubernetesUpdateSecret                 = "UpdateSecret"
	CallKubernetesCreateSelfSubjectRulesReview = "CreateSelfSubjectRulesReview"

	ProviderIBMSM                = "IBM/SecretsManager"
	CallIBMSMGetSecret           = "GetSecret"
	CallIBMSMListSecrets         = "ListSecrets"
	CallIBMSMGetSecretByNameType = "GetSecretByNameType"

	ProviderWebhook    = "Webhook"
	CallWebhookHTTPReq = "HTTPRequest"

	ProviderGitLab                 = "GitLab"
	CallGitLabListProjectsGroups   = "ListProjectsGroups"
	CallGitLabProjectVariableGet   = "ProjectVariableGet"
	CallGitLabProjectListVariables = "ProjectVariablesList"
	CallGitLabGroupGetVariable     = "GroupVariableGet"
	CallGitLabGroupListVariables   = "GroupVariablesList"

	ProviderAKEYLESSSM                  = "AKEYLESSLESS/SecretsManager"
	CallAKEYLESSSMGetSecretValue        = "GetSecretValue"
	CallAKEYLESSSMDescribeItem          = "DescribeItem"
	CallAKEYLESSSMListItems             = "ListItems"
	CallAKEYLESSSMAuth                  = "Auth"
	CallAKEYLESSSMGetRotatedSecretValue = "GetRotatedSecretValue"
	CallAKEYLESSSMGetCertificateValue   = "GetCertificateValue"
	CallAKEYLESSSMGetDynamicSecretValue = "GetDynamicSecretsValue"

	StatusError   = "error"
	StatusSuccess = "success"

	WellKnownLabelKey             = "external-secrets.io/component"
	WellKnownLabelValueController = "controller"
	WellKnownLabelValueWebhook    = "webhook"
)
