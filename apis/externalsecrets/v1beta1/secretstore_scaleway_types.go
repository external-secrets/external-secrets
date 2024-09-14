//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

type ScalewayProviderSecretRef struct {

	// Value can be specified directly to set a value without using a secret.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretRef references a key in a secret that will be used as value.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

type ScalewayProvider struct {

	// APIURL is the url of the api to use. Defaults to https://api.scaleway.com
	// +optional
	APIURL string `json:"apiUrl,omitempty"`

	// Region where your secrets are located: https://developers.scaleway.com/en/quickstart/#region-and-zone
	Region string `json:"region"`

	// ProjectID is the id of your project, which you can find in the console: https://console.scaleway.com/project/settings
	ProjectID string `json:"projectId"`

	// AccessKey is the non-secret part of the api key.
	AccessKey *ScalewayProviderSecretRef `json:"accessKey"`

	// SecretKey is the non-secret part of the api key.
	SecretKey *ScalewayProviderSecretRef `json:"secretKey"`
}
