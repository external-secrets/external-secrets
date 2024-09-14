//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

/*
SenhaseguraAuth tells the controller how to do auth in senhasegura.
*/
type SenhaseguraAuth struct {
	ClientID     string                   `json:"clientId"`
	ClientSecret esmeta.SecretKeySelector `json:"clientSecretSecretRef"`
}

/*
SenhaseguraModuleType enum defines senhasegura target module to fetch secrets
+kubebuilder:validation:Enum=DSM
*/
type SenhaseguraModuleType string

const (
	/*
		SenhaseguraModuleDSM is the senhasegura DevOps Secrets Management module
		see: https://senhasegura.com/devops
	*/
	SenhaseguraModuleDSM SenhaseguraModuleType = "DSM"
)

/*
SenhaseguraProvider setup a store to sync secrets with senhasegura.
*/
type SenhaseguraProvider struct {
	/* URL of senhasegura */
	URL string `json:"url"`

	/* Module defines which senhasegura module should be used to get secrets */
	Module SenhaseguraModuleType `json:"module"`

	/* Auth defines parameters to authenticate in senhasegura */
	Auth SenhaseguraAuth `json:"auth"`

	// IgnoreSslCertificate defines if SSL certificate must be ignored
	// +kubebuilder:default=false
	IgnoreSslCertificate bool `json:"ignoreSslCertificate,omitempty"`
}
