/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"errors"
	"time"

	"github.com/go-logr/logr"

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

// NewFakeLockboxClient creates a new fake LockboxClient.
func NewFakeLockboxClient(fakeLockboxServer *FakeLockboxServer) LockboxClient {
	return &fakeLockboxClient{fakeLockboxServer}
}

func (c *fakeLockboxClient) GetPayloadEntries(_ context.Context, iamToken, secretID, versionID string) ([]*api.Payload_Entry, error) {
	return c.fakeLockboxServer.getEntries(iamToken, secretID, versionID)
}

func (c *fakeLockboxClient) GetExPayload(_ context.Context, iamToken, folderID, name, versionID string) (map[string][]byte, error) {
	return c.fakeLockboxServer.getExPayload(iamToken, folderID, name, versionID)
}

// FakeLockboxServer fakes Yandex Lockbox service backend.
type FakeLockboxServer struct {
	secretMap        map[secretKey]secretValue               // secret specific data
	versionMap       map[versionKey]versionValue             // version specific data
	tokenMap         map[tokenKey]tokenValue                 // token specific data
	folderAndNameMap map[folderAndNameKey]folderAndNameValue // folderAndName specific data

	tokenExpirationDuration time.Duration
	clock                   clock.Clock
	logger                  logr.Logger
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

type folderAndNameKey struct {
	folderID string
	name     string
}

type folderAndNameValue struct {
	secretID string
}

type tokenKey struct {
	token string
}

type tokenValue struct {
	authorizedKey *iamkey.Key
	expiresAt     time.Time
}

// NewFakeLockboxServer creates a new FakeLockboxServer.
func NewFakeLockboxServer(clock clock.Clock, tokenExpirationDuration time.Duration) *FakeLockboxServer {
	return &FakeLockboxServer{
		secretMap:               make(map[secretKey]secretValue),
		versionMap:              make(map[versionKey]versionValue),
		tokenMap:                make(map[tokenKey]tokenValue),
		folderAndNameMap:        make(map[folderAndNameKey]folderAndNameValue),
		tokenExpirationDuration: tokenExpirationDuration,
		clock:                   clock,
	}
}

// CreateSecret creates a new secret with the given entries in the fake server.
func (s *FakeLockboxServer) CreateSecret(authorizedKey *iamkey.Key, folderID, name string, entries ...*api.Payload_Entry) (string, string) {
	secretID := uuid.NewString()
	versionID := uuid.NewString()

	s.secretMap[secretKey{secretID}] = secretValue{authorizedKey}
	s.versionMap[versionKey{secretID, ""}] = versionValue{entries} // empty versionID corresponds to the latest version
	s.versionMap[versionKey{secretID, versionID}] = versionValue{entries}

	if _, exists := s.folderAndNameMap[folderAndNameKey{folderID, name}]; exists {
		s.logger.Error(nil, "On the fake server, you cannot add two certificates with the same name in the same folder")
	}
	s.folderAndNameMap[folderAndNameKey{folderID, name}] = folderAndNameValue{secretID}

	return secretID, versionID
}

// AddVersion adds a new version with the given entries to an existing secret in the fake server.
func (s *FakeLockboxServer) AddVersion(secretID string, entries ...*api.Payload_Entry) string {
	versionID := uuid.NewString()

	s.versionMap[versionKey{secretID, ""}] = versionValue{entries} // empty versionID corresponds to the latest version
	s.versionMap[versionKey{secretID, versionID}] = versionValue{entries}

	return versionID
}

// NewIamToken creates a new IAM token for the given authorized key.
// The token is valid for the duration configured in FakeLockboxServer.
func (s *FakeLockboxServer) NewIamToken(authorizedKey *iamkey.Key) *ydxcommon.IamToken {
	token := uuid.NewString()
	expiresAt := s.clock.CurrentTime().Add(s.tokenExpirationDuration)
	s.tokenMap[tokenKey{token}] = tokenValue{authorizedKey, expiresAt}
	return &ydxcommon.IamToken{Token: token, ExpiresAt: expiresAt}
}

func (s *FakeLockboxServer) getEntries(iamToken, secretID, versionID string) ([]*api.Payload_Entry, error) {
	if _, ok := s.secretMap[secretKey{secretID}]; !ok {
		return nil, errors.New("secret not found")
	}
	if _, ok := s.versionMap[versionKey{secretID, versionID}]; !ok {
		return nil, errors.New("version not found")
	}
	if _, ok := s.tokenMap[tokenKey{iamToken}]; !ok {
		return nil, errors.New("unauthenticated")
	}

	if s.tokenMap[tokenKey{iamToken}].expiresAt.Before(s.clock.CurrentTime()) {
		return nil, errors.New("iam token expired")
	}
	if !cmp.Equal(s.tokenMap[tokenKey{iamToken}].authorizedKey, s.secretMap[secretKey{secretID}].expectedAuthorizedKey, cmpopts.IgnoreUnexported(iamkey.Key{})) {
		return nil, errors.New("permission denied")
	}

	return s.versionMap[versionKey{secretID, versionID}].entries, nil
}

func (s *FakeLockboxServer) getExPayload(iamToken, folderID, name, versionID string) (map[string][]byte, error) {
	if _, ok := s.folderAndNameMap[folderAndNameKey{folderID, name}]; !ok {
		return nil, errors.New("secret not found")
	}
	secretID := s.folderAndNameMap[folderAndNameKey{folderID, name}].secretID
	if _, ok := s.versionMap[versionKey{secretID, versionID}]; !ok {
		return nil, errors.New("version not found")
	}

	if s.tokenMap[tokenKey{iamToken}].expiresAt.Before(s.clock.CurrentTime()) {
		return nil, errors.New("iam token expired")
	}
	if !cmp.Equal(s.tokenMap[tokenKey{iamToken}].authorizedKey, s.secretMap[secretKey{secretID}].expectedAuthorizedKey, cmpopts.IgnoreUnexported(iamkey.Key{})) {
		return nil, errors.New("permission denied")
	}
	entries := s.versionMap[versionKey{secretID, versionID}].entries
	out := make(map[string][]byte, len(entries))
	for _, e := range entries {
		switch e.Value.(type) {
		case *api.Payload_Entry_TextValue:
			out[e.Key] = []byte(e.GetTextValue())
		case *api.Payload_Entry_BinaryValue:
			out[e.Key] = e.GetBinaryValue()
		}
	}
	return out, nil
}
