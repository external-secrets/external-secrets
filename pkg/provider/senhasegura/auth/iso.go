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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type ISOInterface interface {
	IsoSessionFromSecretRef(ctx context.Context, provider *esv1beta1.SenhaseguraProvider, store esv1beta1.GenericStore, kube client.Client, namespace string) (*SenhaseguraIsoSession, error)
	GetIsoToken(clientID, clientSecret, systemURL string, ignoreSslCertificate bool) (token string, err error)
}

/*
SenhaseguraIsoSession contains information about senhasegura ISO API for any request.
*/
type SenhaseguraIsoSession struct {
	URL                  string
	Token                string
	IgnoreSslCertificate bool
	isoClient            ISOInterface
}

/*
isoGetTokenResponse contains response from OAuth2 authentication endpoint in senhasegura API.
*/
type isoGetTokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

var (
	errCannotCreateRequest = errors.New("cannot create request to senhasegura resource /iso/oauth2/token")
	errCannotDoRequest     = errors.New("cannot do request in senhasegura, SSL certificate is valid ?")
	errInvalidResponseBody = errors.New("invalid HTTP response body received from senhasegura")
	errInvalidHTTPCode     = errors.New("received invalid HTTP code from senhasegura")
)

/*
Authenticate check required authentication method based on provider spec and initialize ISO OAuth2 session.
*/
func Authenticate(ctx context.Context, store esv1beta1.GenericStore, provider *esv1beta1.SenhaseguraProvider, kube client.Client, namespace string) (isoSession *SenhaseguraIsoSession, err error) {
	isoSession, err = isoSession.IsoSessionFromSecretRef(ctx, provider, store, kube, namespace)
	if err != nil {
		return nil, err
	}
	return isoSession, nil
}

/*
IsoSessionFromSecretRef initialize an ISO OAuth2 flow with .spec.provider.senhasegura.auth.isoSecretRef parameters.
*/
func (s *SenhaseguraIsoSession) IsoSessionFromSecretRef(ctx context.Context, provider *esv1beta1.SenhaseguraProvider, store esv1beta1.GenericStore, kube client.Client, namespace string) (*SenhaseguraIsoSession, error) {
	secret, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&provider.Auth.ClientSecret,
	)
	if err != nil {
		return &SenhaseguraIsoSession{}, err
	}

	isoToken, err := s.GetIsoToken(provider.Auth.ClientID, secret, provider.URL, provider.IgnoreSslCertificate)
	if err != nil {
		return &SenhaseguraIsoSession{}, err
	}

	return &SenhaseguraIsoSession{
		URL:                  provider.URL,
		Token:                isoToken,
		IgnoreSslCertificate: provider.IgnoreSslCertificate,
		isoClient:            &SenhaseguraIsoSession{},
	}, nil
}

/*
GetIsoToken calls senhasegura OAuth2 endpoint to get a token.
*/
func (s *SenhaseguraIsoSession) GetIsoToken(clientID, clientSecret, systemURL string, ignoreSslCertificate bool) (token string, err error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	u, _ := url.ParseRequestURI(systemURL)
	u.Path = "/iso/oauth2/token"

	tr := &http.Transport{
		//nolint
		TLSClientConfig: &tls.Config{InsecureSkipVerify: ignoreSslCertificate},
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errInvalidHTTPCode
	}

	respData, err := io.ReadAll(resp.Body)
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
