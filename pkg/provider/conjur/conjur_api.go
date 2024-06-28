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

package conjur

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	"github.com/cyberark/conjur-api-go/conjurapi/response"
)

// SecretsClient is an interface for the Conjur client.
type SecretsClient interface {
	RetrieveSecret(secret string) (result []byte, err error)
	RetrieveBatchSecrets(variableIDs []string) (map[string][]byte, error)
	Resources(filter *conjurapi.ResourceFilter) (resources []map[string]interface{}, err error)
}

// SecretsClientFactory is an interface for creating a Conjur client.
type SecretsClientFactory interface {
	NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (SecretsClient, error)
	NewClientFromJWT(config conjurapi.Config, jwtToken string, jwtServiceID, jwtHostID string) (SecretsClient, error)
}

// ClientAPIImpl is an implementation of the ClientAPI interface.
type ClientAPIImpl struct{}

func (c *ClientAPIImpl) NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (SecretsClient, error) {
	return conjurapi.NewClientFromKey(config, loginPair)
}

// NewClientFromJWT creates a new Conjur client from a JWT token.
// cannot use the built-in function "conjurapi.NewClientFromJwt" because it requires environment variables
// see: https://github.com/cyberark/conjur-api-go/blob/b698692392a38e5d38b8440f32ab74206544848a/conjurapi/client.go#L130
func (c *ClientAPIImpl) NewClientFromJWT(config conjurapi.Config, jwtToken, jwtServiceID, jwtHostID string) (SecretsClient, error) {
	jwtTokenString := fmt.Sprintf("jwt=%s", jwtToken)

	var httpClient *http.Client
	if config.IsHttps() {
		cert, err := config.ReadSSLCert()
		if err != nil {
			return nil, err
		}
		httpClient, err = newHTTPSClient(cert)
		if err != nil {
			return nil, err
		}
	} else {
		httpClient = &http.Client{Timeout: time.Second * 10}
	}

	var authnJwtURL string
	// If a hostID is provided, it must be included in the URL
	if jwtHostID != "" {
		authnJwtURL = strings.Join([]string{config.ApplianceURL, "authn-jwt", jwtServiceID, config.Account, url.PathEscape(jwtHostID), "authenticate"}, "/")
	} else {
		authnJwtURL = strings.Join([]string{config.ApplianceURL, "authn-jwt", jwtServiceID, config.Account, "authenticate"}, "/")
	}

	req, err := http.NewRequest("POST", authnJwtURL, strings.NewReader(jwtTokenString))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tokenBytes, err := response.DataResponse(resp)
	if err != nil {
		return nil, err
	}

	return conjurapi.NewClientFromToken(config, string(tokenBytes))
}
