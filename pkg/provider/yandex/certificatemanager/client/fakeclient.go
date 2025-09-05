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
	"errors"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	api "github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
)

// Fake implementation of CertificateManagerClient.
type fakeCertificateManagerClient struct {
	fakeCertificateManagerServer *FakeCertificateManagerServer
}

func NewFakeCertificateManagerClient(fakeCertificateManagerServer *FakeCertificateManagerServer) CertificateManagerClient {
	return &fakeCertificateManagerClient{fakeCertificateManagerServer}
}

func (c *fakeCertificateManagerClient) GetCertificateContent(_ context.Context, iamToken, certificateID, versionID string) (*api.GetCertificateContentResponse, error) {
	return c.fakeCertificateManagerServer.getCertificateContent(iamToken, certificateID, versionID)
}

// Fakes Yandex Certificate Manager service backend.
type FakeCertificateManagerServer struct {
	certificateMap map[certificateKey]certificateValue // certificate specific data
	versionMap     map[versionKey]versionValue         // version specific data
	tokenMap       map[tokenKey]tokenValue             // token specific data

	tokenExpirationDuration time.Duration
	clock                   clock.Clock
}

type certificateKey struct {
	certificateID string
}

type certificateValue struct {
	expectedAuthorizedKey *iamkey.Key // authorized key expected to access the certificate
}

type versionKey struct {
	certificateID string
	versionID     string
}

type versionValue struct {
	content *api.GetCertificateContentResponse
}

type tokenKey struct {
	token string
}

type tokenValue struct {
	authorizedKey *iamkey.Key
	expiresAt     time.Time
}

func NewFakeCertificateManagerServer(clock clock.Clock, tokenExpirationDuration time.Duration) *FakeCertificateManagerServer {
	return &FakeCertificateManagerServer{
		certificateMap:          make(map[certificateKey]certificateValue),
		versionMap:              make(map[versionKey]versionValue),
		tokenMap:                make(map[tokenKey]tokenValue),
		tokenExpirationDuration: tokenExpirationDuration,
		clock:                   clock,
	}
}

func (s *FakeCertificateManagerServer) CreateCertificate(authorizedKey *iamkey.Key, content *api.GetCertificateContentResponse) (string, string) {
	certificateID := uuid.NewString()
	versionID := uuid.NewString()

	s.certificateMap[certificateKey{certificateID}] = certificateValue{authorizedKey}
	s.versionMap[versionKey{certificateID, ""}] = versionValue{content} // empty versionID corresponds to the latest version
	s.versionMap[versionKey{certificateID, versionID}] = versionValue{content}

	return certificateID, versionID
}

func (s *FakeCertificateManagerServer) AddVersion(certificateID string, content *api.GetCertificateContentResponse) string {
	versionID := uuid.NewString()

	s.versionMap[versionKey{certificateID, ""}] = versionValue{content} // empty versionID corresponds to the latest version
	s.versionMap[versionKey{certificateID, versionID}] = versionValue{content}

	return versionID
}

func (s *FakeCertificateManagerServer) NewIamToken(authorizedKey *iamkey.Key) *common.IamToken {
	token := uuid.NewString()
	expiresAt := s.clock.CurrentTime().Add(s.tokenExpirationDuration)
	s.tokenMap[tokenKey{token}] = tokenValue{authorizedKey, expiresAt}
	return &common.IamToken{Token: token, ExpiresAt: expiresAt}
}

func (s *FakeCertificateManagerServer) getCertificateContent(iamToken, certificateID, versionID string) (*api.GetCertificateContentResponse, error) {
	if _, ok := s.certificateMap[certificateKey{certificateID}]; !ok {
		return nil, errors.New("certificate not found")
	}
	if _, ok := s.versionMap[versionKey{certificateID, versionID}]; !ok {
		return nil, errors.New("version not found")
	}
	if _, ok := s.tokenMap[tokenKey{iamToken}]; !ok {
		return nil, errors.New("unauthenticated")
	}

	if s.tokenMap[tokenKey{iamToken}].expiresAt.Before(s.clock.CurrentTime()) {
		return nil, errors.New("iam token expired")
	}
	if !cmp.Equal(s.tokenMap[tokenKey{iamToken}].authorizedKey, s.certificateMap[certificateKey{certificateID}].expectedAuthorizedKey, cmpopts.IgnoreUnexported(iamkey.Key{})) {
		return nil, errors.New("permission denied")
	}

	return s.versionMap[versionKey{certificateID, versionID}].content, nil
}
