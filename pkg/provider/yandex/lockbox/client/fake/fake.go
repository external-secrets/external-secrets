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
package fake

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
)

// Fake implementation of YandexCloudCreator.
type YandexCloudCreator struct {
	Backend *LockboxBackend
}

func (c *YandexCloudCreator) CreateLockboxClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (client.LockboxClient, error) {
	return &LockboxClient{c.Backend}, nil
}

func (c *YandexCloudCreator) CreateIamToken(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key) (*client.IamToken, error) {
	return c.Backend.getToken(authorizedKey)
}

func (c *YandexCloudCreator) Now() time.Time {
	return c.Backend.now
}

// Fake implementation of LockboxClient.
type LockboxClient struct {
	fakeLockboxBackend *LockboxBackend
}

func (c *LockboxClient) GetPayloadEntries(ctx context.Context, iamToken, secretID, versionID string) ([]*lockbox.Payload_Entry, error) {
	return c.fakeLockboxBackend.getEntries(iamToken, secretID, versionID)
}

// Fakes Yandex Lockbox service backend.
type LockboxBackend struct {
	secretMap  map[secretKey]secretValue   // secret specific data
	versionMap map[versionKey]versionValue // version specific data
	tokenMap   map[tokenKey]tokenValue     // token specific data

	tokenExpirationDuration time.Duration
	now                     time.Time // fakes the current time
}

type secretKey struct {
	secretID string
}

type secretValue struct {
	expectedAuthorizedKey *iamkey.Key // authorized key expected to access the secret
}

type versionKey struct {
	secretID  string
	versionID string
}

type versionValue struct {
	entries []*lockbox.Payload_Entry
}

type tokenKey struct {
	token string
}

type tokenValue struct {
	authorizedKey *iamkey.Key
	expiresAt     time.Time
}

func NewLockboxBackend(tokenExpirationDuration time.Duration) *LockboxBackend {
	return &LockboxBackend{
		secretMap:               make(map[secretKey]secretValue),
		versionMap:              make(map[versionKey]versionValue),
		tokenMap:                make(map[tokenKey]tokenValue),
		tokenExpirationDuration: tokenExpirationDuration,
		now:                     time.Time{},
	}
}

func (lb *LockboxBackend) CreateSecret(authorizedKey *iamkey.Key, entries ...*lockbox.Payload_Entry) (string, string) {
	secretID := uuid.NewString()
	versionID := uuid.NewString()

	lb.secretMap[secretKey{secretID}] = secretValue{authorizedKey}
	lb.versionMap[versionKey{secretID, ""}] = versionValue{entries} // empty versionID corresponds to the latest version
	lb.versionMap[versionKey{secretID, versionID}] = versionValue{entries}

	return secretID, versionID
}

func (lb *LockboxBackend) AddVersion(secretID string, entries ...*lockbox.Payload_Entry) string {
	versionID := uuid.NewString()

	lb.versionMap[versionKey{secretID, ""}] = versionValue{entries} // empty versionID corresponds to the latest version
	lb.versionMap[versionKey{secretID, versionID}] = versionValue{entries}

	return versionID
}

func (lb *LockboxBackend) AdvanceClock(duration time.Duration) {
	lb.now = lb.now.Add(duration)
}

func (lb *LockboxBackend) getToken(authorizedKey *iamkey.Key) (*client.IamToken, error) {
	token := uuid.NewString()
	expiresAt := lb.now.Add(lb.tokenExpirationDuration)
	lb.tokenMap[tokenKey{token}] = tokenValue{authorizedKey, expiresAt}
	return &client.IamToken{Token: token, ExpiresAt: expiresAt}, nil
}

func (lb *LockboxBackend) getEntries(iamToken, secretID, versionID string) ([]*lockbox.Payload_Entry, error) {
	if _, ok := lb.secretMap[secretKey{secretID}]; !ok {
		return nil, fmt.Errorf("secret not found")
	}
	if _, ok := lb.versionMap[versionKey{secretID, versionID}]; !ok {
		return nil, fmt.Errorf("version not found")
	}
	if _, ok := lb.tokenMap[tokenKey{iamToken}]; !ok {
		return nil, fmt.Errorf("unauthenticated")
	}

	if lb.tokenMap[tokenKey{iamToken}].expiresAt.Before(lb.now) {
		return nil, fmt.Errorf("iam token expired")
	}
	if !cmp.Equal(lb.tokenMap[tokenKey{iamToken}].authorizedKey, lb.secretMap[secretKey{secretID}].expectedAuthorizedKey, cmpopts.IgnoreUnexported(iamkey.Key{})) {
		return nil, fmt.Errorf("permission denied")
	}

	return lb.versionMap[versionKey{secretID, versionID}].entries, nil
}
