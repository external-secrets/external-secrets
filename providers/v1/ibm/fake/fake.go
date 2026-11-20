/*
Copyright © The ESO Authors

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

package fake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

var errNotMocked = errors.New("not mocked")

type IBMMockClient struct {
	getSecretWithContext           func(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
	getSecretByNameTypeWithContext func(ctx context.Context, getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
}

type IBMMockClientParams struct {
	GetSecretOptions       *sm.GetSecretOptions
	GetSecretOutput        sm.SecretIntf
	GetSecretErr           error
	GetSecretByNameOptions *sm.GetSecretByNameTypeOptions
	GetSecretByNameOutput  sm.SecretIntf
	GetSecretByNameErr     error
}

func (mc *IBMMockClient) GetSecretWithContext(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error) {
	return mc.getSecretWithContext(ctx, getSecretOptions)
}

func (mc *IBMMockClient) GetSecretByNameTypeWithContext(
	ctx context.Context,
	getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions,
) (result sm.SecretIntf, response *core.DetailedResponse, err error) {
	return mc.getSecretByNameTypeWithContext(ctx, getSecretByNameTypeOptions)
}

func (mc *IBMMockClient) CreateSecretWithContext(context.Context, *sm.CreateSecretOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
	return nil, nil, errNotMocked
}

func (mc *IBMMockClient) CreateSecretVersionWithContext(context.Context, *sm.CreateSecretVersionOptions) (sm.SecretVersionIntf, *core.DetailedResponse, error) {
	return nil, nil, errNotMocked
}

func (mc *IBMMockClient) DeleteSecretWithContext(context.Context, *sm.DeleteSecretOptions) (*core.DetailedResponse, error) {
	return nil, errNotMocked
}

func (mc *IBMMockClient) ListSecretGroupsWithContext(context.Context, *sm.ListSecretGroupsOptions) (*sm.SecretGroupCollection, *core.DetailedResponse, error) {
	return nil, nil, errNotMocked
}

func (mc *IBMMockClient) WithValue(params IBMMockClientParams) {
	if mc != nil {
		mc.getSecretWithContext = func(ctx context.Context, paramReq *sm.GetSecretOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			if !cmp.Equal(paramReq, params.GetSecretOptions, cmpopts.IgnoreUnexported(sm.Secret{})) {
				return nil, nil, fmt.Errorf("unexpected test argument for GetSecret: %s, %s", *paramReq.ID, *params.GetSecretOptions.ID)
			}
			return params.GetSecretOutput, nil, params.GetSecretErr
		}
		mc.getSecretByNameTypeWithContext = func(ctx context.Context, paramReq *sm.GetSecretByNameTypeOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
			// type secretmanagerpb.AccessSecretVersionRequest contains unexported fields
			// use cmpopts.IgnoreUnexported to ignore all the unexported fields in the cmp.
			if !cmp.Equal(paramReq, params.GetSecretByNameOptions, cmpopts.IgnoreUnexported(sm.Secret{})) {
				return nil, nil, fmt.Errorf("unexpected test argument for GetSecretByNameType: %s, %s", *paramReq.Name, *params.GetSecretByNameOptions.Name)
			}
			return params.GetSecretByNameOutput, nil, params.GetSecretByNameErr
		}
	}
}

// InMemoryClient is a stateful Secrets Manager backend holding arbitrary and kv secrets.
type InMemoryClient struct {
	// Groups maps custom secret group names to IDs; the default group always exists.
	Groups  map[string]string
	secrets map[string]sm.SecretIntf
}

// NewInMemoryClient returns an empty backend with the given custom secret groups.
func NewInMemoryClient(groups map[string]string) *InMemoryClient {
	return &InMemoryClient{Groups: groups, secrets: make(map[string]sm.SecretIntf)}
}

func notFound() (*core.DetailedResponse, error) {
	return &core.DetailedResponse{StatusCode: http.StatusNotFound}, errors.New("Not Found")
}

// copySecret stops callers from changing stored secrets through returned maps.
func copySecret(secret sm.SecretIntf) sm.SecretIntf {
	b, err := json.Marshal(secret)
	if err != nil {
		panic(err)
	}
	var out sm.SecretIntf
	switch secret.(type) {
	case *sm.ArbitrarySecret:
		out = &sm.ArbitrarySecret{}
	case *sm.KVSecret:
		out = &sm.KVSecret{}
	}
	if err := json.Unmarshal(b, out); err != nil {
		panic(err)
	}
	return out
}

// Put stores a secret as is, for setting up secrets ESO did not create.
func (c *InMemoryClient) Put(secret sm.SecretIntf) {
	switch s := secret.(type) {
	case *sm.ArbitrarySecret:
		c.secrets[*s.ID] = copySecret(s)
	case *sm.KVSecret:
		c.secrets[*s.ID] = copySecret(s)
	default:
		panic(fmt.Sprintf("unsupported secret %T", secret))
	}
}

// Find returns a copy of the secret with the given name, type and group ID, or nil.
func (c *InMemoryClient) Find(name, secretType, groupID string) sm.SecretIntf {
	for _, secret := range c.secrets {
		switch s := secret.(type) {
		case *sm.ArbitrarySecret:
			if *s.Name == name && *s.SecretType == secretType && *s.SecretGroupID == groupID {
				return copySecret(s)
			}
		case *sm.KVSecret:
			if *s.Name == name && *s.SecretType == secretType && *s.SecretGroupID == groupID {
				return copySecret(s)
			}
		}
	}
	return nil
}

func (c *InMemoryClient) GetSecretWithContext(_ context.Context, opts *sm.GetSecretOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
	secret, ok := c.secrets[*opts.ID]
	if !ok {
		resp, err := notFound()
		return nil, resp, err
	}
	return copySecret(secret), nil, nil
}

func (c *InMemoryClient) GetSecretByNameTypeWithContext(_ context.Context, opts *sm.GetSecretByNameTypeOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
	groupID := *opts.SecretGroupName
	if groupID != "default" {
		var ok bool
		if groupID, ok = c.Groups[groupID]; !ok {
			resp, err := notFound()
			return nil, resp, err
		}
	}
	if secret := c.Find(*opts.Name, *opts.SecretType, groupID); secret != nil {
		return secret, nil, nil
	}
	resp, err := notFound()
	return nil, resp, err
}

func (c *InMemoryClient) CreateSecretWithContext(_ context.Context, opts *sm.CreateSecretOptions) (sm.SecretIntf, *core.DetailedResponse, error) {
	id := uuid.NewString()
	var secret sm.SecretIntf
	switch p := opts.SecretPrototype.(type) {
	case *sm.ArbitrarySecretPrototype:
		if c.Find(*p.Name, *p.SecretType, *p.SecretGroupID) != nil {
			return nil, &core.DetailedResponse{StatusCode: http.StatusConflict}, errors.New("Conflict")
		}
		secret = &sm.ArbitrarySecret{
			ID: &id, Name: p.Name, SecretType: p.SecretType, SecretGroupID: p.SecretGroupID,
			Payload: p.Payload, CustomMetadata: p.CustomMetadata, VersionsTotal: new(int64(1)),
		}
	case *sm.KVSecretPrototype:
		if c.Find(*p.Name, *p.SecretType, *p.SecretGroupID) != nil {
			return nil, &core.DetailedResponse{StatusCode: http.StatusConflict}, errors.New("Conflict")
		}
		secret = &sm.KVSecret{
			ID: &id, Name: p.Name, SecretType: p.SecretType, SecretGroupID: p.SecretGroupID,
			Data: p.Data, CustomMetadata: p.CustomMetadata, VersionsTotal: new(int64(1)),
		}
	default:
		return nil, nil, fmt.Errorf("unsupported prototype %T", opts.SecretPrototype)
	}
	c.Put(secret)
	return copySecret(secret), nil, nil
}

func (c *InMemoryClient) CreateSecretVersionWithContext(_ context.Context, opts *sm.CreateSecretVersionOptions) (sm.SecretVersionIntf, *core.DetailedResponse, error) {
	secret, ok := c.secrets[*opts.SecretID]
	if !ok {
		resp, err := notFound()
		return nil, resp, err
	}
	switch p := opts.SecretVersionPrototype.(type) {
	case *sm.ArbitrarySecretVersionPrototype:
		s, ok := secret.(*sm.ArbitrarySecret)
		if !ok {
			return nil, &core.DetailedResponse{StatusCode: http.StatusBadRequest}, errors.New("Bad Request")
		}
		s.Payload = p.Payload
		*s.VersionsTotal++
	case *sm.KVSecretVersionPrototype:
		s, ok := secret.(*sm.KVSecret)
		if !ok {
			return nil, &core.DetailedResponse{StatusCode: http.StatusBadRequest}, errors.New("Bad Request")
		}
		s.Data = p.Data
		*s.VersionsTotal++
	default:
		return nil, nil, fmt.Errorf("unsupported version prototype %T", opts.SecretVersionPrototype)
	}
	c.Put(secret)
	return nil, nil, nil
}

func (c *InMemoryClient) DeleteSecretWithContext(_ context.Context, opts *sm.DeleteSecretOptions) (*core.DetailedResponse, error) {
	if _, ok := c.secrets[*opts.ID]; !ok {
		return notFound()
	}
	delete(c.secrets, *opts.ID)
	return &core.DetailedResponse{StatusCode: http.StatusNoContent}, nil
}

func (c *InMemoryClient) ListSecretGroupsWithContext(context.Context, *sm.ListSecretGroupsOptions) (*sm.SecretGroupCollection, *core.DetailedResponse, error) {
	groups := make([]sm.SecretGroup, 0, len(c.Groups))
	for name, id := range c.Groups {
		groups = append(groups, sm.SecretGroup{ID: new(id), Name: new(name)})
	}
	return &sm.SecretGroupCollection{SecretGroups: groups, TotalCount: new(int64(len(groups)))}, nil, nil
}
