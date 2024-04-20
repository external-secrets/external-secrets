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

package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	api "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
)

// Fake implementation of LockboxClient.
type fakeLockboxClient struct {
	fakeLockboxServer *FakeLockboxServer
}

func NewFakeLockboxClient(fakeLockboxServer *FakeLockboxServer) LockboxClient {
	return &fakeLockboxClient{fakeLockboxServer}
}

func (c *fakeLockboxClient) GetPayloadEntries(_ context.Context, iamToken, secretID, versionID string) ([]*api.Payload_Entry, error) {
	return c.fakeLockboxServer.getEntries(iamToken, secretID, versionID)
}

// Fakes Yandex Lockbox service backend.
type FakeLockboxServer struct {
	secretMap  map[secretKey]secretValue   // secret specific data
	versionMap map[versionKey]versionValue // version specific data
	tokenMap   map[tokenKey]tokenValue     // token specific data

	tokenExpirationDuration time.Duration
	clock                   clock.Clock
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
	entries []*api.Payload_Entry
}

type tokenKey struct {
	token string
}

type tokenValue struct {
	authorizedKey *iamkey.Key
	expiresAt     time.Time
}

func NewFakeLockboxServer(clock clock.Clock, tokenExpirationDuration time.Duration) *FakeLockboxServer {
	return &FakeLockboxServer{
		secretMap:               make(map[secretKey]secretValue),
		versionMap:              make(map[versionKey]versionValue),
		tokenMap:                make(map[tokenKey]tokenValue),
		tokenExpirationDuration: tokenExpirationDuration,
		clock:                   clock,
	}
}

func (s *FakeLockboxServer) CreateSecret(authorizedKey *iamkey.Key, entries ...*api.Payload_Entry) (string, string) {
	secretID := uuid.NewString()
	versionID := uuid.NewString()

	s.secretMap[secretKey{secretID}] = secretValue{authorizedKey}
	s.versionMap[versionKey{secretID, ""}] = versionValue{entries} // empty versionID corresponds to the latest version
	s.versionMap[versionKey{secretID, versionID}] = versionValue{entries}

	return secretID, versionID
}

func (s *FakeLockboxServer) AddVersion(secretID string, entries ...*api.Payload_Entry) string {
	versionID := uuid.NewString()

	s.versionMap[versionKey{secretID, ""}] = versionValue{entries} // empty versionID corresponds to the latest version
	s.versionMap[versionKey{secretID, versionID}] = versionValue{entries}

	return versionID
}

func (s *FakeLockboxServer) NewIamToken(authorizedKey *iamkey.Key) *common.IamToken {
	token := uuid.NewString()
	expiresAt := s.clock.CurrentTime().Add(s.tokenExpirationDuration)
	s.tokenMap[tokenKey{token}] = tokenValue{authorizedKey, expiresAt}
	return &common.IamToken{Token: token, ExpiresAt: expiresAt}
}

func (s *FakeLockboxServer) getEntries(iamToken, secretID, versionID string) ([]*api.Payload_Entry, error) {
	if _, ok := s.secretMap[secretKey{secretID}]; !ok {
		return nil, fmt.Errorf("secret not found")
	}
	if _, ok := s.versionMap[versionKey{secretID, versionID}]; !ok {
		return nil, fmt.Errorf("version not found")
	}
	if _, ok := s.tokenMap[tokenKey{iamToken}]; !ok {
		return nil, fmt.Errorf("unauthenticated")
	}

	if s.tokenMap[tokenKey{iamToken}].expiresAt.Before(s.clock.CurrentTime()) {
		return nil, fmt.Errorf("iam token expired")
	}
	if !cmp.Equal(s.tokenMap[tokenKey{iamToken}].authorizedKey, s.secretMap[secretKey{secretID}].expectedAuthorizedKey, cmpopts.IgnoreUnexported(iamkey.Key{})) {
		return nil, fmt.Errorf("permission denied")
	}

	return s.versionMap[versionKey{secretID, versionID}].entries, nil
}
