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

package dsm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	senhaseguraAuth "github.com/external-secrets/external-secrets/pkg/provider/senhasegura/auth"
)

type clientDSMInterface interface {
	FetchSecrets() (respObj IsoDappResponse, err error)
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &DSM{}

/*
DSM service for SenhaseguraProvider.
*/
type DSM struct {
	isoSession *senhaseguraAuth.SenhaseguraIsoSession
	dsmClient  clientDSMInterface
}

/*
IsoDappResponse is a response object from senhasegura /iso/dapp/response (DevOps Secrets Management API endpoint)
Contains information about API request and Secrets linked with authorization.
*/
type IsoDappResponse struct {
	Response struct {
		Status    int    `json:"status"`
		Message   string `json:"message"`
		Error     bool   `json:"error"`
		ErrorCode int    `json:"error_code"`
	} `json:"response"`
	Application struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		System      string   `json:"system"`
		Environment string   `json:"Environment"`
		Secrets     []struct {
			SecretID       string              `json:"secret_id"`
			SecretName     string              `json:"secret_name"`
			Identity       string              `json:"identity"`
			Version        string              `json:"version"`
			ExpirationDate string              `json:"expiration_date"`
			Engine         string              `json:"engine"`
			Data           []map[string]string `json:"data"`
		} `json:"secrets"`
	} `json:"application"`
}

var (
	errCannotCreateRequest = errors.New("cannot create request to senhasegura resource /iso/dapp/application")
	errCannotDoRequest     = errors.New("cannot do request in senhasegura, SSL certificate is valid ?")
	errInvalidResponseBody = errors.New("invalid HTTP response body received from senhasegura")
	errInvalidHTTPCode     = errors.New("received invalid HTTP code from senhasegura")
	errApplicationError    = errors.New("received application error from senhasegura")
	errNotImplemented      = errors.New("not implemented")
)

/*
New creates an senhasegura DSM client based on ISO session.
*/
func New(isoSession *senhaseguraAuth.SenhaseguraIsoSession) (*DSM, error) {
	return &DSM{
		isoSession: isoSession,
		dsmClient:  &DSM{},
	}, nil
}

func (dsm *DSM) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return errNotImplemented
}

func (dsm *DSM) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errNotImplemented
}

// Not Implemented PushSecret.
func (dsm *DSM) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return errNotImplemented
}

/*
GetSecret implements ESO interface and get a single secret from senhasegura provider with DSM service.
*/
func (dsm *DSM) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (resp []byte, err error) {
	appSecrets, err := dsm.FetchSecrets()
	if err != nil {
		return []byte(""), err
	}

	for _, v := range appSecrets.Application.Secrets {
		if ref.Key == v.Identity {
			// Return whole data content in json-encoded when ref.Property is empty
			if ref.Property == "" {
				jsonStr, err := json.Marshal(v.Data)
				if err != nil {
					return nil, err
				}
				return jsonStr, nil
			}

			// Return raw data content when ref.Property is provided
			for _, v2 := range v.Data {
				for k, v3 := range v2 {
					if k == ref.Property {
						resp = []byte(v3)
						return resp, nil
					}
				}
			}
		}
	}

	return []byte(""), esv1beta1.NoSecretErr
}

/*
GetSecretMap implements ESO interface and returns miltiple k/v pairs from senhasegura provider with DSM service.
*/
func (dsm *DSM) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (secretData map[string][]byte, err error) {
	secretData = make(map[string][]byte)
	appSecrets, err := dsm.FetchSecrets()
	if err != nil {
		return secretData, err
	}

	for _, v := range appSecrets.Application.Secrets {
		if v.Identity == ref.Key {
			for _, v2 := range v.Data {
				for k, v3 := range v2 {
					secretData[k] = []byte(v3)
				}
			}
		}
	}
	return secretData, nil
}

/*
GetAllSecrets implements ESO interface and returns multiple secrets from senhasegura provider with DSM service

TODO: GetAllSecrets functionality is to get secrets from either regexp-matching against the names or via metadata label matching.
https://github.com/external-secrets/external-secrets/pull/830#discussion_r858657107
*/
func (dsm *DSM) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (secretData map[string][]byte, err error) {
	return nil, errNotImplemented
}

/*
fetchSecrets calls senhasegura DSM /iso/dapp/application API endpoint
Return an IsoDappResponse with all related information from senhasegura provider with DSM service and error.
*/
func (dsm *DSM) FetchSecrets() (respObj IsoDappResponse, err error) {
	u, _ := url.ParseRequestURI(dsm.isoSession.URL)
	u.Path = "/iso/dapp/application"

	tr := &http.Transport{
		//nolint
		TLSClientConfig: &tls.Config{InsecureSkipVerify: dsm.isoSession.IgnoreSslCertificate},
	}

	client := &http.Client{Transport: tr}

	r, err := http.NewRequest("GET", u.String(), http.NoBody)
	if err != nil {
		return respObj, errCannotCreateRequest
	}

	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Authorization", "Bearer "+dsm.isoSession.Token)

	resp, err := client.Do(r)
	if err != nil {
		return respObj, errCannotDoRequest
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return respObj, errInvalidHTTPCode
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return respObj, errInvalidResponseBody
	}

	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return respObj, errInvalidResponseBody
	}

	if respObj.Response.Error {
		return respObj, errApplicationError
	}

	return respObj, nil
}

/*
Close implements ESO interface and do nothing in senhasegura.
*/
func (dsm *DSM) Close(_ context.Context) error {
	return nil
}

// Validate if has valid connection with senhasegura, credentials, authorization using fetchSecrets method
// fetchSecrets method implement required check about request
// https://github.com/external-secrets/external-secrets/pull/830#discussion_r833275463
func (dsm *DSM) Validate() (esv1beta1.ValidationResult, error) {
	_, err := dsm.FetchSecrets()
	if err != nil {
		return esv1beta1.ValidationResultError, err
	}

	return esv1beta1.ValidationResultReady, nil
}
