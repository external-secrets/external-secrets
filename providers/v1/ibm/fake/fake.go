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
	"errors"
	"fmt"
	"net/http"

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type IBMMockClient struct {
	getSecretWithContext           func(ctx context.Context, getSecretOptions *sm.GetSecretOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)
	getSecretByNameTypeWithContext func(ctx context.Context, getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions) (result sm.SecretIntf, response *core.DetailedResponse, err error)

	// ListPages are returned in order, one per ListSecretsWithContext call, so a
	// test can exercise paging. ListCalled records the options of each call.
	ListPages  []*sm.SecretMetadataPaginatedCollection
	listCalls  int
	Groups     []sm.SecretGroup
	ListCalled []*sm.ListSecretsOptions

	// ByName serves GetSecretByNameTypeWithContext keyed by secret name, for
	// tests that read several secrets and do not care about exact options
	// matching.
	ByName map[string]sm.SecretIntf

	// ByID serves GetSecretWithContext keyed by secret ID. GetAllSecrets reads
	// by ID, so this is the hook its tests use. NotFoundIDs are reported as a
	// 404 so the not-found translation can be exercised.
	ByID        map[string]sm.SecretIntf
	NotFoundIDs map[string]bool
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
	if mc.ByID != nil || mc.NotFoundIDs != nil {
		id := *getSecretOptions.ID
		if mc.NotFoundIDs[id] {
			return nil, &core.DetailedResponse{StatusCode: http.StatusNotFound}, errors.New("not found")
		}
		if secret, ok := mc.ByID[id]; ok {
			return secret, &core.DetailedResponse{StatusCode: http.StatusOK}, nil
		}
		return nil, nil, fmt.Errorf("fake: no secret configured for id %q", id)
	}
	return mc.getSecretWithContext(ctx, getSecretOptions)
}

func (mc *IBMMockClient) GetSecretByNameTypeWithContext(
	ctx context.Context,
	getSecretByNameTypeOptions *sm.GetSecretByNameTypeOptions,
) (result sm.SecretIntf, response *core.DetailedResponse, err error) {
	if mc.ByName != nil {
		secret, ok := mc.ByName[*getSecretByNameTypeOptions.Name]
		if !ok || secret == nil {
			return nil, nil, fmt.Errorf("fake: no secret configured for name %q", *getSecretByNameTypeOptions.Name)
		}
		return secret, nil, nil
	}
	return mc.getSecretByNameTypeWithContext(ctx, getSecretByNameTypeOptions)
}

// ListSecretsWithContext returns the configured pages in order.
func (mc *IBMMockClient) ListSecretsWithContext(_ context.Context, options *sm.ListSecretsOptions) (*sm.SecretMetadataPaginatedCollection, *core.DetailedResponse, error) {
	mc.ListCalled = append(mc.ListCalled, options)
	if mc.listCalls >= len(mc.ListPages) {
		return &sm.SecretMetadataPaginatedCollection{TotalCount: new(int64(0))}, nil, nil
	}
	page := mc.ListPages[mc.listCalls]
	mc.listCalls++
	return page, nil, nil
}

// ListSecretGroupsWithContext returns the configured secret groups.
func (mc *IBMMockClient) ListSecretGroupsWithContext(_ context.Context, _ *sm.ListSecretGroupsOptions) (*sm.SecretGroupCollection, *core.DetailedResponse, error) {
	return &sm.SecretGroupCollection{
		SecretGroups: mc.Groups,
		TotalCount:   new(int64(len(mc.Groups))),
	}, nil, nil
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
