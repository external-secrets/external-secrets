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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	ExternalSecretSubsystem = "externalsecret"
	providerAPICalls        = "provider_api_calls_count"
)

const (
	ProviderAWSSM           = "AWS/SecretsManager"
	CallAWSSMGetSecretValue = "GetSecretValue"
	CallAWSSMDescribeSecret = "DescribeSecret"
	CallAWSSMDeleteSecret   = "DeleteSecret"
	CallAWSSMCreateSecret   = "CreateSecret"
	CallAWSSMPutSecretValue = "PutSecretValue"
	CallAWSSMListSecrets    = "ListSecrets"

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
	CallAzureKVDeleteSecret      = "DeleteSecret"
	CallAzureKVGetCertificate    = "GetCertificate"
	CallAzureKVDeleteCertificate = "DeleteCertificate"
	CallAzureKVImportCertificate = "ImportCertificate"

	ProviderGCPSM                = "GCP/SecretManager"
	CallGCPSMGetSecret           = "GetSecret"
	CallGCPSMDeleteSecret        = "DeleteSecret"
	CallGCPSMCreateSecret        = "CreateSecret"
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
	CallKubernetesCreateSelfSubjectRulesReview = "CreateSelfSubjectRulesReview"

	ProviderIBMSM      = "IBM/SecretsManager"
	CallIBMSMGetSecret = "GetSecret"

	ProviderWebhook    = "Webhook"
	CallWebhookHTTPReq = "HTTPRequest"

	ProviderGitLab                 = "GitLab"
	CallGitLabListProjectsGroups   = "ListProjectsGroups"
	CallGitLabProjectVariableGet   = "ProjectVariableGet"
	CallGitLabProjectListVariables = "ProjectVariablesList"
	CallGitLabGroupGetVariable     = "GroupVariableGet"
	CallGitLabGroupListVariables   = "GroupVariablesList"

	StatusError   = "error"
	StatusSuccess = "success"
)

var (
	syncCallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      providerAPICalls,
		Help:      "Number of API calls towards the secret provider",
	}, []string{"provider", "call", "status"})
)

func ObserveAPICall(provider, call string, err error) {
	syncCallsTotal.WithLabelValues(provider, call, deriveStatus(err)).Inc()
}

func deriveStatus(err error) string {
	if err != nil {
		return StatusError
	}
	return StatusSuccess
}

func init() {
	metrics.Registry.MustRegister(syncCallsTotal)
}
