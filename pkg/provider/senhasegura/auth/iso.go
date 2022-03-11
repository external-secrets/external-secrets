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

package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/*
	SenhaseguraIsoSession contains information about senhasegura ISO API for any request
*/
type SenhaseguraIsoSession struct {
	Url   string
	Token string
}

/*
	isoGetTokenResponse contains response from OAuth2 authentication endpoint in senhasegura API
*/
type isoGetTokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

var (
	errCannotCreateRequest  = errors.New("Cannot create request to senhasegura resource /iso/oauth2/token")
	errCannotDoRequest      = errors.New("Cannot do request in senhasegura, SSL certificate is valid ?")
	errInvalidResponseBody  = errors.New("Invalid HTTP response body received from senhasegura")
	errInvalidHttpCode      = errors.New("Received invalid HTTP code from senhasegura")
	errRequiredIsoSecretRef = errors.New("Required auth.isoSecretRef not found")
)

/*
	Authenticate check required authentication method based on provider spec and initialize ISO OAuth2 session
*/
func Authenticate(ctx context.Context, store esv1beta1.GenericStore, provider *esv1beta1.SenhaseguraProvider, kube client.Client, namespace string) (isoSession *SenhaseguraIsoSession, err error) {
	if provider.Auth.IsoSecretRef != nil {
		isoSession, err = isoSessionFromSecretRef(ctx, provider, store, kube, namespace)
		if err != nil {
			return nil, err
		}
		return isoSession, nil
	} else {
		return &SenhaseguraIsoSession{}, errRequiredIsoSecretRef
	}

}

/*
	isoSessionFromSecretRef initialize an ISO OAuth2 flow with .spec.provider.senhasegura.auth.isoSecretRef parameters
*/
func isoSessionFromSecretRef(ctx context.Context, provider *esv1beta1.SenhaseguraProvider, store esv1beta1.GenericStore, kube client.Client, namespace string) (*SenhaseguraIsoSession, error) {

	clientId, err := getKubernetesSecret(ctx, provider.Auth.IsoSecretRef.ClientId, store, kube, namespace)
	if err != nil {
		return &SenhaseguraIsoSession{}, err
	}
	clientSecret, err := getKubernetesSecret(ctx, provider.Auth.IsoSecretRef.ClientSecret, store, kube, namespace)
	if err != nil {
		return &SenhaseguraIsoSession{}, err
	}
	systemUrl, err := getKubernetesSecret(ctx, provider.Auth.IsoSecretRef.Url, store, kube, namespace)
	if err != nil {
		return &SenhaseguraIsoSession{}, err
	}

	isoToken, err := getIsoToken(ctx, clientId, clientSecret, systemUrl)
	if err != nil {
		return &SenhaseguraIsoSession{}, err
	}

	return &SenhaseguraIsoSession{
		Url:   systemUrl,
		Token: isoToken,
	}, nil

}

/*
	getIsoToken calls senhasegura OAuth2 endpoint to get a token
*/
func getIsoToken(ctx context.Context, clientId, clientSecret, systemUrl string) (token string, err error) {

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)

	u, _ := url.ParseRequestURI(systemUrl)
	u.Path = "/iso/oauth2/token"

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	client := &http.Client{Transport: tr}

	r, err := http.NewRequest("POST", u.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return "", errCannotCreateRequest
	}

	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := client.Do(r)
	if err != nil {
		return "", errCannotDoRequest
	}

	if resp.StatusCode != 200 {
		return "", errInvalidHttpCode
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errInvalidResponseBody
	}

	var respObj isoGetTokenResponse
	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return "", errInvalidResponseBody
	}

	return respObj.AccessToken, nil
}
