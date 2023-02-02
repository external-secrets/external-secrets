package scaleway

import (
	"fmt"
	"github.com/google/uuid"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"sort"
	"strconv"
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
	_project *fakeProject
}

type fakeProject struct {
	id      string
	secrets []*fakeSecret
}

type fakeSecretApi struct {
	projects       []*fakeProject
	_secrets       map[string]*fakeSecret
	_secretsByName map[string]*fakeSecret
}

func buildDb(f *fakeSecretApi) *fakeSecretApi {

	f._secrets = map[string]*fakeSecret{}
	f._secretsByName = map[string]*fakeSecret{}

	for _, project := range f.projects {

		if project.id == "" {
			project.id = uuid.NewString()
		}

		for _, secret := range project.secrets {

			if secret.id == "" {
				secret.id = uuid.NewString()
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

			secret._project = project

			f._secrets[secret.id] = secret
			f._secretsByName[secret.name] = secret
		}
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
	} else {
		return nil, false
	}
}

func (s *fakeSecret) mustGetVersion(revision string) *fakeSecretVersion {

	version, ok := s.getVersion(revision)
	if !ok {
		panic("no such version")
	}

	return version
}

func (f *fakeSecretApi) secret(name string) *fakeSecret {
	return f._secretsByName[name]
}

func (f *fakeSecretApi) AccessSecretVersion(request *smapi.AccessSecretVersionRequest, _ ...scw.RequestOption) (*smapi.AccessSecretVersionResponse, error) {

	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	secret, ok := f._secrets[request.SecretID]
	if !ok {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "secret",
			ResourceID: request.SecretID,
		}
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

func matchListSecretFilter(secret *fakeSecret, filter *smapi.ListSecretsRequest) bool {

	// TODO

	for _, requiredTag := range filter.Tags {

		found := false
		for _, secretTag := range secret.tags {
			if requiredTag == secretTag {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func (f *fakeSecretApi) ListSecrets(request *smapi.ListSecretsRequest, _ ...scw.RequestOption) (*smapi.ListSecretsResponse, error) {

	var matches []*fakeSecret

	// filtering

	for _, project := range f.projects {

		if request.ProjectID != nil && *request.ProjectID != project.id {
			continue
		}

		for _, secret := range project.secrets {
			if matchListSecretFilter(secret, request) {
				matches = append(matches, secret)
			}
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
		TotalCount: uint32(len(matches)),
	}

	if request.Page == nil || request.PageSize == nil {
		panic("list secrets without explicit pagination not implemented")
	}
	page := int(*request.Page)
	pageSize := int(*request.PageSize)

	startOffset := (page - 1) * pageSize
	if startOffset > len(matches) {
		startOffset = len(matches)
	}

	endOffset := page * pageSize
	if endOffset > len(matches) {
		endOffset = len(matches)
	}

	for _, secret := range matches[startOffset:endOffset] {
		response.Secrets = append(response.Secrets, &smapi.Secret{
			ID:           secret.id,
			ProjectID:    secret._project.id,
			Name:         secret.name,
			Status:       smapi.SecretStatus(secret.status),
			Tags:         secret.tags,
			VersionCount: uint32(len(secret.versions)),
		})
	}

	return &response, nil
}

func (f *fakeSecretApi) CreateSecretVersion(request *smapi.CreateSecretVersionRequest, _ ...scw.RequestOption) (*smapi.SecretVersion, error) {

	if request.Region != "" {
		panic("explicit region in request is not supported")
	}

	secret, ok := f._secrets[request.SecretID]
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
