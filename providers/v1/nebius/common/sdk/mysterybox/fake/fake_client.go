// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package fake

import (
	"context"
	"sync/atomic"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox"
)

type FakeMysteryboxClient struct {
	mysteryboxService *MysteryboxService
	Closed            int32
}

func (f *FakeMysteryboxClient) Close() error {
	atomic.AddInt32(&f.Closed, 1)
	return nil
}

func (f FakeMysteryboxClient) GetSecret(_ context.Context, _, secretId, versionId string) (*mysterybox.Payload, error) {
	secret, err := f.mysteryboxService.GetSecret(secretId, versionId)
	if err != nil {
		return nil, err
	}
	return &mysterybox.Payload{
		VersionID: secret.VersionId,
		Entries:   secret.Entries,
	}, nil
}

func (f FakeMysteryboxClient) GetSecretByKey(_ context.Context, _, secretID, versionID, key string) (*mysterybox.PayloadEntry, error) {
	secret, err := f.mysteryboxService.GetSecret(secretID, versionID)
	if err != nil {
		return nil, err
	}
	for _, entry := range secret.Entries {
		if entry.Key == key {
			return &mysterybox.PayloadEntry{
				VersionID: versionID,
				Entry:     entry,
			}, nil
		}
	}

	return nil, notFoundError()
}

type MysteryboxService struct {
	secretData map[string]map[string][]mysterybox.Entry
}

func InitMysteryboxService() *MysteryboxService {
	return &MysteryboxService{
		secretData: make(map[string]map[string][]mysterybox.Entry),
	}
}

func (s *MysteryboxService) GetSecret(secretId, versionId string) (*Secret, error) {
	data, ok := s.secretData[secretId]
	if !ok {
		return nil, notFoundError()
	}
	dataByVersion, ok := data[versionId] // if version is empty -> "" (latest/primary) version will be taken
	if !ok {
		return nil, notFoundError()
	}
	return &Secret{
		Id:        secretId,
		VersionId: versionId,
		Entries:   dataByVersion,
	}, nil
}

func (s *MysteryboxService) CreateSecret(payloadEntries []mysterybox.Entry) *Secret {
	if len(payloadEntries) == 0 {
		return nil
	}
	secretId := uuid.NewString()
	versionId := uuid.NewString()

	versionData := make(map[string][]mysterybox.Entry)
	versionData[versionId] = payloadEntries
	versionData[""] = payloadEntries // latest version is primary
	s.secretData[secretId] = versionData

	return &Secret{
		Id:        secretId,
		VersionId: versionId,
		Entries:   payloadEntries,
	}
}

func (s *MysteryboxService) CreateNewSecretVersion(secretId string, payloadEntries []mysterybox.Entry) (string, error) {
	versions, ok := s.secretData[secretId]
	if !ok {
		return "", notFoundError()
	}
	versionId := uuid.NewString()
	versions[versionId] = payloadEntries
	versions[""] = payloadEntries // latest version is primary
	return versionId, nil
}

func NewFakeMysteryboxClient(service *MysteryboxService) *FakeMysteryboxClient {
	return &FakeMysteryboxClient{
		mysteryboxService: service,
	}
}

type Secret struct {
	Id        string
	VersionId string
	Entries   []mysterybox.Entry
}

func notFoundError() error {
	return status.Error(codes.NotFound, "not found")
}

var _ mysterybox.Client = &FakeMysteryboxClient{}
