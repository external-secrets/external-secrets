package infisical

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
)

type MachineTokenManager struct {
	accessTokenTTL           time.Duration
	accessTokenMaxTTL        time.Duration
	accessTokenFetchedTime   time.Time
	accessTokenRefreshedTime time.Time

	mutex sync.Mutex

	accessToken  string
	clientSecret string
	clientId     string

	apiClient api.InfisicalApis
}

func NewMachineTokenManger(apiClient api.InfisicalApis, clientId string, clientSecret string) *MachineTokenManager {
	token := &MachineTokenManager{
		clientId:     clientId,
		clientSecret: clientSecret,
		apiClient:    apiClient,
	}

	return token
}

func (t *MachineTokenManager) StartLifeCycle() error {
	if err := t.FetchNewAccessToken(); err != nil {
		return err
	}
	go t.HandleTokenLifecycle()
	return nil
}

func (t *MachineTokenManager) HandleTokenLifecycle() error {
	// this is skip doing things in first iteration
	// the refresh part is not needed to trigger at first because access token is just received as on top
	init := false
	for {
		accessTokenMaxTTLExpiresInTime := t.accessTokenFetchedTime.Add(t.accessTokenMaxTTL - (5 * time.Second))
		accessTokenRefreshedTime := t.accessTokenRefreshedTime

		if accessTokenRefreshedTime.IsZero() {
			accessTokenRefreshedTime = t.accessTokenFetchedTime
		}

		nextAccessTokenExpiresInTime := accessTokenRefreshedTime.Add(t.accessTokenTTL - (5 * time.Second))

		if time.Now().After(accessTokenMaxTTLExpiresInTime) {
			Logger.Info("Infisical Authentication: machine identity access token has reached max ttl, attempting to re authenticate...")
			err := t.FetchNewAccessToken()
			if err != nil {
				Logger.Error(err, "Infisical Authentication: unable to authenticate universal auth. Will retry in 30 seconds")

				// wait a bit before trying again
				time.Sleep((30 * time.Second))
				continue
			}
		} else if init == true {
			err := t.RefreshAccessToken()
			if err != nil {
				Logger.Error(err, "Infisical Authentication: unable to refresh universal auth token because %v. Will retry in 30 seconds")

				// wait a bit before trying again
				time.Sleep((30 * time.Second))
				continue
			}
		}

		if accessTokenRefreshedTime.IsZero() {
			accessTokenRefreshedTime = t.accessTokenFetchedTime
		} else {
			accessTokenRefreshedTime = t.accessTokenRefreshedTime
		}

		nextAccessTokenExpiresInTime = accessTokenRefreshedTime.Add(t.accessTokenTTL - (5 * time.Second))
		accessTokenMaxTTLExpiresInTime = t.accessTokenFetchedTime.Add(t.accessTokenMaxTTL - (5 * time.Second))

		if nextAccessTokenExpiresInTime.After(accessTokenMaxTTLExpiresInTime) {
			// case: Refreshed so close that the next refresh would occur beyond max ttl (this is because currently, token renew tries to add +access-token-ttl amount of time)
			// example: access token ttl is 11 sec and max ttl is 30 sec. So it will start with 11 seconds, then 22 seconds but the next time you call refresh it would try to extend it to 33 but max ttl only allows 30, so the token will be valid until 30 before we need to reauth
			time.Sleep(t.accessTokenTTL - nextAccessTokenExpiresInTime.Sub(accessTokenMaxTTLExpiresInTime))
		} else {
			time.Sleep(t.accessTokenTTL - (5 * time.Second))
		}

		init = true
	}
}

func (t *MachineTokenManager) RefreshAccessToken() error {
	resp, err := t.apiClient.RefreshMachineIdentityAccessToken(api.MachineIdentityUniversalAuthRefreshRequest{AccessToken: t.accessToken})
	if err != nil {
		return err
	}

	accessTokenTTL := time.Duration(resp.ExpiresIn * int(time.Second))
	accessTokenMaxTTL := time.Duration(resp.AccessTokenMaxTTL * int(time.Second))
	t.accessTokenRefreshedTime = time.Now()

	t.SetAccessToken(resp.AccessToken, accessTokenTTL, accessTokenMaxTTL)

	return nil
}

// Fetches a new access token using client credentials
func (t *MachineTokenManager) FetchNewAccessToken() error {
	loginResponse, err := t.apiClient.MachineIdentityLoginViaUniversalAuth(api.MachineIdentityUniversalAuthLoginRequest{
		ClientId:     t.clientId,
		ClientSecret: t.clientSecret,
	})
	if err != nil {
		return err
	}

	accessTokenTTL := time.Duration(loginResponse.ExpiresIn * int(time.Second))
	accessTokenMaxTTL := time.Duration(loginResponse.AccessTokenMaxTTL * int(time.Second))

	if accessTokenTTL <= time.Duration(5)*time.Second {
		Logger.Info("Infisical Authentication: At this time, k8 operator does not support refresh of tokens with 5 seconds or less ttl. Please increase access token ttl and try again")
		os.Exit(1)
	}

	t.accessTokenFetchedTime = time.Now()
	t.SetAccessToken(loginResponse.AccessToken, accessTokenTTL, accessTokenMaxTTL)

	return nil
}

func (t *MachineTokenManager) SetAccessToken(token string, accessTokenTTL time.Duration, accessTokenMaxTTL time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.accessToken = token
	t.accessTokenTTL = accessTokenTTL
	t.accessTokenMaxTTL = accessTokenMaxTTL
}

func (t *MachineTokenManager) GetAccessToken() (string, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.accessToken == "" {
		return "", errors.New("Missing access token")
	}

	return t.accessToken, nil
}
