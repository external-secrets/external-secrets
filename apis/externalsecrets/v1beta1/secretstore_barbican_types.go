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

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type BarbicanAuthUserPass struct {
	UserName    string                 `json:"userName"`
	PasswordRef *BarbicanAuthSecretRef `json:"passwordRef"`
}

type BarbicanAuthAppCredentials struct {
	ApplicationID        string                 `json:"applicationID"`
	ApplicationSecretRef *BarbicanAuthSecretRef `json:"applicationSecretRef"`
}

type BarbicanAuth struct {
	// Authentication strategy with username and password
	// +optional
	UserPass *BarbicanAuthUserPass `json:"userPass,omitempty"`

	// Authentication strategy with application credentials
	// +optional
	AppCredentials *BarbicanAuthAppCredentials `json:"appCredentials,omitempty"`
}

type BarbicanAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

// BarbicanProvider Configures a store to sync secrets using the Barbican Secret Manager provider.
type BarbicanProvider struct {
	// Auth defines the information necessary to authenticate against Barbican
	// +optional
	Auth BarbicanAuth `json:"auth,omitempty"`

	// OpenStack Auth Url
	AuthUrl string `json:"authURL"`

	// The Domain of the user.
	// +optional
	UserDomain string `json:"userDomain"`

	// The project name. Both the Project ID and Project Name are optional.
	// +optional
	ProjectName string `json:"projectName"`

	// ServiceName [optional] is the service name for the client (e.g., "nova") as it
	// appears in the service catalog. Services can have the same Type but a
	// different Name, which is why both Type and Name are sometimes needed.
	// +optional
	// +kubebuilder:default=barbican
	ServiceName string `json:"serviceName,omitempty"`

	// Region [required] is the geographic region in which the endpoint resides,
	// generally specifying which datacenter should house your resources.
	// Required only for services that span multiple regions.
	// +optional
	Region string `json:"region"`
}
