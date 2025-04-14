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

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// AkeylessProvider Configures an store to sync secrets using Akeyless KV.
type AkeylessProvider struct {

	// Akeyless GW API Url from which the secrets to be fetched from.
	AkeylessGWApiURL *string `json:"akeylessGWApiURL"`

	// Auth configures how the operator authenticates with Akeyless.
	Auth *AkeylessAuth `json:"authSecretRef"`

	// PEM/base64 encoded CA bundle used to validate Akeyless Gateway certificate. Only used
	// if the AkeylessGWApiURL URL is using HTTPS protocol. If not set the system root certificates
	// are used to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The provider for the CA bundle to use to validate Akeyless Gateway certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}

type AkeylessAuth struct {

	// Reference to a Secret that contains the details
	// to authenticate with Akeyless.
	// +optional
	SecretRef AkeylessAuthSecretRef `json:"secretRef,omitempty"`

	// Kubernetes authenticates with Akeyless by passing the ServiceAccount
	// token stored in the named Secret resource.
	// +optional
	KubernetesAuth *AkeylessKubernetesAuth `json:"kubernetesAuth,omitempty"`
}

// AkeylessAuthSecretRef
// AKEYLESS_ACCESS_TYPE_PARAM: AZURE_OBJ_ID OR GCP_AUDIENCE OR ACCESS_KEY OR KUB_CONFIG_NAME.
type AkeylessAuthSecretRef struct {
	// The SecretAccessID is used for authentication
	AccessID        esmeta.SecretKeySelector `json:"accessID,omitempty"`
	AccessType      esmeta.SecretKeySelector `json:"accessType,omitempty"`
	AccessTypeParam esmeta.SecretKeySelector `json:"accessTypeParam,omitempty"`
}

// Authenticate with Kubernetes ServiceAccount token stored.
type AkeylessKubernetesAuth struct {

	// the Akeyless Kubernetes auth-method access-id
	AccessID string `json:"accessID"`

	// Kubernetes-auth configuration name in Akeyless-Gateway
	K8sConfName string `json:"k8sConfName"`

	// Optional service account field containing the name of a kubernetes ServiceAccount.
	// If the service account is specified, the service account secret token JWT will be used
	// for authenticating with Akeyless. If the service account selector is not supplied,
	// the secretRef will be used instead.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// Optional secret field containing a Kubernetes ServiceAccount JWT used
	// for authenticating with Akeyless. If a name is specified without a key,
	// `token` is the default. If one is not specified, the one bound to
	// the controller will be used.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}
