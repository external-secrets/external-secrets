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

package scaleway

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/google/uuid"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type fakeSecretVersion struct {
	revision     int
	data         []byte
	dontFillData bool
	status       string
}

type fakeSecret struct {
	id       string
	name     string
	versions []*fakeSecretVersion
	tags     []string
	status   string
	path     string
}

var _ secretAPI = (*fakeSecretAPI)(nil)

type fakeSecretAPI struct {
	secrets        []*fakeSecret
	_secretsByID   map[string]*fakeSecret
	_secretsByName map[string]*fakeSecret
}

func buildDB(f *fakeSecretAPI) *fakeSecretAPI {
	f._secretsByID = map[string]*fakeSecret{}
	f._secretsByName = map[string]*fakeSecret{}

	for _, secret := range f.secrets {
		if secret.id == "" {
			secret.id = uuid.NewString()
		}

		if secret.path == "" {
			secret.path = "/"
		}

		sort.Slice(secret.versions, func(i, j int) bool {
			return secret.versions[i].revision < secret.versions[j].revision
		})

		for index, version := range secret.versions {
			if version.revision != index+1 {
				panic("bad revision number in fixtures")
			}

			if version.status == "" {
				version.status = "enabled"
			}
		}

		for _, version := range secret.versions {
			if len(version.data) == 0 && !version.dontFillData {
				version.data = []byte(fmt.Sprintf("some data for secret %s version %d: %s", secret.id, version.revision, uuid.NewString()))
			}
		}

		if secret.status == "" {
			secret.status = "ready"
		}

		f._secretsByID[secret.id] = secret
		f._secretsByName[secret.name] = secret
	}

	return f
}

func (s *fakeSecret) getVersion(revision string) (*fakeSecretVersion, bool) {
	if len(s.versions) == 0 {
		return nil, false
	}

	if revision == "latest" {
		return s.versions[len(s.versions)-1], true
	}

	if revision == "latest_enabled" {
		for i := len(s.versions) - 1; i >= 0; i-- {
			if s.versions[i].status == "enabled" {
				return s.versions[i], true
			}
		}
		return nil, false
	}

	revisionNumber, err := strconv.Atoi(revision)
	if err != nil {
		return nil, false
	}

	i, found := sort.Find(len(s.versions), func(i int) int {
		if revisionNumber < s.versions[i].revision {
			return -1
		} else if revisionNumber > s.versions[i].revision {
			return 1
		} else {
			return 0
		}
	})
	if found {
		return s.versions[i], true
	}
	return nil, false
}

func (s *fakeSecret) mustGetVersion(revision string) *fakeSecretVersion {
	version, ok := s.getVersion(revision)
	if !ok {
		panic("no such version")
	}

	return version
}

func (f *fakeSecretAPI) secret(name string) *fakeSecret {
	return f._secretsByName[name]
}

func (f *fakeSecretAPI) getSecretByID(secretID string) (*fakeSecret, error) {
	secret, foundSecret := f._secretsByID[secretID]

	if !foundSecret {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "secret",
			ResourceID: secretID,
		}
	}

	return secret, nil
}

func (f *fakeSecretAPI) GetSecret(request *smapi.GetSecretRequest, _ ...scw.RequestOption) (*smapi.Secret, error) {
	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	secret, err := f.getSecretByID(request.SecretID)
	if err != nil {
		return nil, err
	}

	return &smapi.Secret{
		ID:           secret.id,
		Name:         secret.name,
		Status:       smapi.SecretStatus(secret.status),
		Tags:         secret.tags,
		VersionCount: uint32(len(secret.versions)),
		Path:         secret.path,
	}, nil
}

func (f *fakeSecretAPI) GetSecretVersion(request *smapi.GetSecretVersionRequest, _ ...scw.RequestOption) (*smapi.SecretVersion, error) {
	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	secret, err := f.getSecretByID(request.SecretID)
	if err != nil {
		return nil, err
	}

	version, ok := secret.getVersion(request.Revision)
	if !ok {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "secret_version",
			ResourceID: request.Revision,
		}
	}

	return &smapi.SecretVersion{
		SecretID: secret.id,
		Revision: uint32(version.revision),
		Status:   smapi.SecretVersionStatus(secret.status),
	}, nil
}

func (f *fakeSecretAPI) AccessSecretVersion(request *smapi.AccessSecretVersionRequest, _ ...scw.RequestOption) (*smapi.AccessSecretVersionResponse, error) {
	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	secret, err := f.getSecretByID(request.SecretID)
	if err != nil {
		return nil, err
	}

	version, ok := secret.getVersion(request.Revision)
	if !ok {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "secret_version",
			ResourceID: request.Revision,
		}
	}

	return &smapi.AccessSecretVersionResponse{
		SecretID: secret.id,
		Revision: uint32(version.revision),
		Data:     version.data,
	}, nil
}

func (f *fakeSecretAPI) DisableSecretVersion(request *smapi.DisableSecretVersionRequest, _ ...scw.RequestOption) (*smapi.SecretVersion, error) {
	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	secret, err := f.getSecretByID(request.SecretID)
	if err != nil {
		return nil, err
	}

	version, ok := secret.getVersion(request.Revision)
	if !ok {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "secret_version",
			ResourceID: request.Revision,
		}
	}

	version.status = "disabled"

	return &smapi.SecretVersion{
		SecretID: secret.id,
		Revision: uint32(version.revision),
		Status:   smapi.SecretVersionStatus(version.status),
	}, nil
}

type secretFilter func(*fakeSecret) bool

func matchListSecretFilter(secret *fakeSecret, filter *smapi.ListSecretsRequest) bool {
	filters := make([]secretFilter, 0)

	if filter.Tags != nil {
		filters = append(filters, func(fs *fakeSecret) bool {
			for _, requiredTag := range filter.Tags {
				for _, secretTag := range fs.tags {
					if requiredTag == secretTag {
						return true
					}
				}
			}
			return false
		})
	}

	if filter.Name != nil {
		filters = append(filters, func(fs *fakeSecret) bool {
			return *filter.Name == fs.name
		})
	}

	if filter.Path != nil {
		filters = append(filters, func(fs *fakeSecret) bool {
			return *filter.Path == fs.path
		})
	}

	match := true

	for _, filterFn := range filters {
		match = match && filterFn(secret)
	}

	return match
}

func (f *fakeSecretAPI) ListSecrets(request *smapi.ListSecretsRequest, _ ...scw.RequestOption) (*smapi.ListSecretsResponse, error) {
	var matches []*fakeSecret

	// filtering

	for _, secret := range f.secrets {
		if matchListSecretFilter(secret, request) {
			matches = append(matches, secret)
		}
	}

	// ordering

	if request.OrderBy != "" {
		panic("explicit order by is not implemented")
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].id >= matches[j].id
	})

	// pagination

	response := smapi.ListSecretsResponse{
		TotalCount: uint64(len(matches)),
	}

	if request.Page == nil || request.PageSize == nil {
		panic("list secrets without explicit pagination not implemented")
	}
	page := int(*request.Page)
	pageSize := int(*request.PageSize)

	startOffset := (page - 1) * pageSize
	if startOffset > len(matches) {
		return nil, fmt.Errorf("invalid page offset (page = %d, page size = %d, total = %d)", page, pageSize, len(matches))
	}

	endOffset := page * pageSize
	if endOffset > len(matches) {
		endOffset = len(matches)
	}

	for _, secret := range matches[startOffset:endOffset] {
		response.Secrets = append(response.Secrets, &smapi.Secret{
			ID:           secret.id,
			Name:         secret.name,
			Status:       smapi.SecretStatus(secret.status),
			Tags:         secret.tags,
			VersionCount: uint32(len(secret.versions)),
			Path:         secret.path,
		})
	}

	return &response, nil
}

func (f *fakeSecretAPI) CreateSecret(request *smapi.CreateSecretRequest, _ ...scw.RequestOption) (*smapi.Secret, error) {
	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	path := "/"
	if request.Path != nil {
		path = *request.Path
	}

	secret := &fakeSecret{
		id:     uuid.NewString(),
		name:   request.Name,
		status: "ready",
		path:   path,
	}

	f.secrets = append(f.secrets, secret)
	f._secretsByID[secret.id] = secret
	f._secretsByName[secret.name] = secret

	return &smapi.Secret{
		ID:           secret.id,
		ProjectID:    request.ProjectID,
		Name:         secret.name,
		Status:       smapi.SecretStatus(secret.status),
		VersionCount: 0,
		Path:         secret.path,
	}, nil
}

func (f *fakeSecretAPI) CreateSecretVersion(request *smapi.CreateSecretVersionRequest, _ ...scw.RequestOption) (*smapi.SecretVersion, error) {
	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	secret, ok := f._secretsByID[request.SecretID]
	if !ok {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "secret",
			ResourceID: request.SecretID,
		}
	}

	newVersion := &fakeSecretVersion{
		revision: len(secret.versions) + 1,
		data:     request.Data,
	}

	secret.versions = append(secret.versions, newVersion)

	return &smapi.SecretVersion{
		SecretID: request.SecretID,
		Revision: uint32(newVersion.revision),
		Status:   smapi.SecretVersionStatus(newVersion.status),
	}, nil
}

func (f *fakeSecretAPI) DeleteSecret(request *smapi.DeleteSecretRequest, _ ...scw.RequestOption) error {
	secret, ok := f._secretsByID[request.SecretID]
	if !ok {
		return &scw.ResourceNotFoundError{
			Resource:   "secret",
			ResourceID: request.SecretID,
		}
	}
	delete(f._secretsByID, secret.id)

	return nil
}
