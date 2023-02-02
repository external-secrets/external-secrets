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
}

type fakeSecret struct {
	id       string
	name     string
	versions []*fakeSecretVersion
	tags     []string
}

type fakeProject struct {
	id      string
	secrets []*fakeSecret
}

type fakeSecretApi struct {
	projects []*fakeProject
	_secrets map[string]*fakeSecret
}

func buildDb(f *fakeSecretApi) *fakeSecretApi {

	f._secrets = map[string]*fakeSecret{}

	for _, project := range f.projects {

		for _, secret := range project.secrets {

			if secret.id == "" {
				secret.id = uuid.NewString()
			}

			// TODO: check for duplicates
			sort.Sort(fakeSecretVersionSlice(secret.versions))

			for _, version := range secret.versions {
				if len(version.data) == 0 && !version.dontFillData {
					version.data = []byte(fmt.Sprintf("some data for secret %s version %d: %s", secret.id, version.revision, uuid.NewString()))
				}
			}

			f._secrets[secret.id] = secret
		}
	}

	return f
}

type fakeSecretVersionSlice []*fakeSecretVersion

func (vs fakeSecretVersionSlice) Len() int {
	return len(vs)
}

func (vs fakeSecretVersionSlice) Less(i, j int) bool {
	return vs[i].revision < vs[j].revision
}

func (vs fakeSecretVersionSlice) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
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

func (f *fakeSecretApi) AccessSecretVersion(request *smapi.AccessSecretVersionRequest, _ ...scw.RequestOption) (*smapi.AccessSecretVersionResponse, error) {

	// TODO: check region

	secret, ok := f._secrets[request.SecretID]
	if !ok {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "", // TODO
			ResourceID: request.SecretID,
		}
	}

	version, ok := secret.getVersion(request.Revision)
	if !ok {
		return nil, &scw.ResourceNotFoundError{
			Resource:   "",               // TODO
			ResourceID: request.SecretID, // TODO
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

	// TODO: order correctly
	// TODO: scope by orga/project

	for _, project := range f.projects {
		for _, secret := range project.secrets {
			if matchListSecretFilter(secret, request) {
				matches = append(matches, secret)
			}
		}
	}

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
			ProjectID:    "", // TODO
			Name:         secret.name,
			Status:       "", // TODO
			Tags:         secret.tags,
			Region:       "", // TODO
			VersionCount: uint32(len(secret.versions)),
		})
	}

	return &response, nil
}
