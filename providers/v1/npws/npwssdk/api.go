// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package npwssdk implements a Go SDK for Netwrix Password Secure (NPWS) with E2E encryption.
package npwssdk

import (
	"net/http"
	"strings"
)

// PsrAPI is the main entry point for the NPWS SDK.
// It provides access to authentication, container management, and encryption.
type PsrAPI struct {
	Auth              *AuthenticationManager
	UserKeys          *UserKeyManager
	Containers        *ContainerManager
	Encryption        *EncryptionManager
	Rights            *RightManager
	Seals             *SealManager
	OrganisationUnits *OrganisationUnitManager
	GenericRights     *GenericRightManager

	authClient    *HTTPClient
	serviceClient *HTTPClient
}

// NewPsrAPI creates a new PsrAPI instance connected to the given endpoint.
// The endpoint should be the base URL of the NPWS server (e.g., "https://npws.example.com").
func NewPsrAPI(endpoint string) *PsrAPI {
	// Ensure https prefix
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}

	authClient := NewHTTPClient(endpoint + "auth/")
	serviceClient := NewHTTPClient(endpoint + "WebService/")
	encryption := NewEncryptionManager(EncryptionV1) // Default to V1, server may upgrade
	rights := NewRightManager(serviceClient)
	seals := NewSealManager(serviceClient)
	orgUnits := NewOrganisationUnitManager(serviceClient)
	userKeys := NewUserKeyManager(serviceClient, encryption, rights, seals)
	auth := NewAuthenticationManager(authClient, serviceClient, encryption, userKeys)
	genericRights := NewGenericRightManager(rights, seals, userKeys)
	containers := NewContainerManager(serviceClient, userKeys, encryption, orgUnits, rights, genericRights)

	return &PsrAPI{
		Auth:              auth,
		UserKeys:          userKeys,
		Containers:        containers,
		Encryption:        encryption,
		Rights:            rights,
		Seals:             seals,
		OrganisationUnits: orgUnits,
		GenericRights:     genericRights,
		authClient:        authClient,
		serviceClient:     serviceClient,
	}
}

// SetClientType overrides the ClientType sent to NPWS during authentication.
// Default is "PsrAPIGo". Must be called before LoginWithAPIKey.
func (api *PsrAPI) SetClientType(clientType string) {
	api.Auth.clientInfo.ClientType = clientType
}

// SetClientVersion overrides the ClientVersion sent to NPWS during authentication.
// Default is "1.0.0". Must be called before LoginWithAPIKey.
func (api *PsrAPI) SetClientVersion(clientVersion string) {
	api.Auth.clientInfo.ClientVersion = clientVersion
}

// SetHTTPClient overrides the underlying http.Client on both internal clients.
// Useful for custom TLS settings (e.g. skipping certificate verification for localhost).
func (api *PsrAPI) SetHTTPClient(hc *http.Client) {
	api.authClient.SetHTTPClient(hc)
	api.serviceClient.SetHTTPClient(hc)
}
