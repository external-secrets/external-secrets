package conjur

import (
	"fmt"
	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	"github.com/cyberark/conjur-api-go/conjurapi/response"
	"net/http"
	"strings"
	"time"
)

// Client is an interface for the Conjur client.
type Client interface {
	RetrieveSecret(secret string) (result []byte, err error)
}

// ClientApi is an interface for creating a Conjur client.
type ClientApi interface {
	NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (Client, error)
	NewClientFromJWT(config conjurapi.Config, jwtToken string, jwtServiceId string) (Client, error)
}

// ConjurClientApi is an implementation of the ClientApi interface.
type ConjurClientApi struct{}

func (c *ConjurClientApi) NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (Client, error) {
	return conjurapi.NewClientFromKey(config, loginPair)
}

// NewClientFromJWT creates a new Conjur client from a JWT token.
// cannot use the built-in function "conjurapi.NewClientFromJwt" because it requires environment variables
// see: https://github.com/cyberark/conjur-api-go/blob/b698692392a38e5d38b8440f32ab74206544848a/conjurapi/client.go#L130
func (c *ConjurClientApi) NewClientFromJWT(config conjurapi.Config, jwtToken string, jwtServiceId string) (Client, error) {
	var jwtTokenString string
	jwtTokenString = fmt.Sprintf("jwt=%s", jwtToken)

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

	authnJwtUrl := strings.Join([]string{config.ApplianceURL, "authn-jwt", jwtServiceId, config.Account, "authenticate"}, "/")

	req, err := http.NewRequest("POST", authnJwtUrl, strings.NewReader(jwtTokenString))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	tokenBytes, err := response.DataResponse(resp)
	if err != nil {
		return nil, err
	}

	return conjurapi.NewClientFromToken(config, string(tokenBytes))
}
