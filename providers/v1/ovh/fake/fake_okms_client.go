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

package fake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go"
	"github.com/ovh/okms-sdk-go/types"
)

type FakeOkmsClient struct {
	TestCase string
}

func (kmsClient FakeOkmsClient) GetSecretV2(_ context.Context, _ uuid.UUID, path string, _ *uint32, _ *bool) (*types.GetSecretV2Response, error) {
	// Called by GetSecret() & GetSecretMap()

	// Metadata
	CasRequired := true
	CreatedAt := "string"
	DeactivateVersionAfter := "string"
	MaxVersions := uint32(10)
	UpdatedAt := "string"

	// Version
	Data := map[string]any{
		"key": "value",
	}
	NestedData := map[string]any{
		"projects": map[string]any{
			"project1": "Name",
			"project2": "Name",
		},
		"test": "value",
	}
	DeactivatedAt := "string"
	Warnings := []string{}
	dataSecretV2Response := &types.GetSecretV2Response{
		Metadata: &types.SecretV2Metadata{
			CasRequired:            &CasRequired,
			CreatedAt:              &CreatedAt,
			DeactivateVersionAfter: &DeactivateVersionAfter,
			MaxVersions:            &MaxVersions,
			UpdatedAt:              &UpdatedAt,
		},
		Path: &kmsClient.TestCase,
		Version: &types.SecretV2Version{
			CreatedAt:     "string",
			Data:          &Data,
			DeactivatedAt: &DeactivatedAt,
			Id:            1,
			State:         "string",
			Warnings:      &Warnings,
		},
	}
	nestedDataSecretV2Response := &types.GetSecretV2Response{
		Metadata: &types.SecretV2Metadata{
			CasRequired:            &CasRequired,
			CreatedAt:              &CreatedAt,
			DeactivateVersionAfter: &DeactivateVersionAfter,
			MaxVersions:            &MaxVersions,
			UpdatedAt:              &UpdatedAt,
		},
		Path: &kmsClient.TestCase,
		Version: &types.SecretV2Version{
			CreatedAt:     "string",
			Data:          &NestedData,
			DeactivatedAt: &DeactivatedAt,
			Id:            1,
			State:         "string",
			Warnings:      &Warnings,
		},
	}

	// Test Cases
	switch kmsClient.TestCase {
	case "Valid Secret":
		return dataSecretV2Response, nil
	case "MetaDataPolicy: Fetch":
		return dataSecretV2Response, nil
	case "Non-existent Secret":
		kmsError := okms.NewKmsErrorFromBytes([]byte("{\"error_code\":17125377}"))
		return nil, kmsError
	case "Secret without data":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &map[string]interface{}{},
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "Valid property that gets Nested Json":
		return nestedDataSecretV2Response, nil
	case "Valid property that gets non_Nested Json":
		return nestedDataSecretV2Response, nil
	case "Invalid property":
		return dataSecretV2Response, nil
	case "Empty property":
		return dataSecretV2Response, nil
	case "Secret Version":
		return dataSecretV2Response, nil
	case "Invalid Secret Version":
		kmsError := okms.NewKmsErrorFromBytes([]byte("{\"error_code\":17125378}"))
		return nil, kmsError
	case "Error case":
		return &types.GetSecretV2Response{}, errors.New("SecretExists error")
	}

	if path == "" {
		return &types.GetSecretV2Response{}, errors.New("unknown case")
	}

	// Called by GetAllSecrets()
	data1 := map[string]any{
		"projects": map[string]any{
			"project1": "Name",
			"project2": "Name",
		},
	}
	data2 := map[string]any{
		"key": "value",
	}
	data3 := map[string]any{
		"root": map[string]any{
			"sub1": map[string]any{
				"value": "string",
			},
			"sub2": "Name",
		},
		"test":  "value",
		"test1": "value1",
	}
	data4 := map[string]any{
		"test4": "value4",
	}
	data5 := map[string]any{
		"test5": "value5",
	}
	data6 := map[string]any{
		"test6": "value6",
	}
	data7 := map[string]any{
		"test7": "value7",
	}
	data8 := map[string]any{
		"test8": "value8",
	}
	switch path {
	case "pattern1/path1":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data1,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "pattern1/path2":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data2,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "pattern1/path3":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data3,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "pattern2/test/test-secret":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data4,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "pattern2/test/test.secret":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data5,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "pattern2/secret":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data6,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "1secret":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data7,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "pattern2/test/test;secret":
		return &types.GetSecretV2Response{
			Metadata: &types.SecretV2Metadata{
				CasRequired:            &CasRequired,
				CreatedAt:              &CreatedAt,
				DeactivateVersionAfter: &DeactivateVersionAfter,
				MaxVersions:            &MaxVersions,
				UpdatedAt:              &UpdatedAt,
			},
			Path: &kmsClient.TestCase,
			Version: &types.SecretV2Version{
				CreatedAt:     "string",
				Data:          &data8,
				DeactivatedAt: &DeactivatedAt,
				Id:            1,
				State:         "string",
				Warnings:      &Warnings,
			},
		}, nil
	case "non-existent":
		kmsError := okms.NewKmsErrorFromBytes([]byte("{\"error_code\":17125377}"))
		return &types.GetSecretV2Response{}, kmsError
	}
	return &types.GetSecretV2Response{}, errors.New("unknown path")
}

func (kmsClient FakeOkmsClient) GetSecretsMetadata(_ context.Context, _ uuid.UUID, path string, _ bool) (*types.GetMetadataResponse, error) {
	switch path {
	case "nil resp":
		return nil, nil
	case "nil data struct":
		return &types.GetMetadataResponse{}, nil
	case "nil secrets list":
		return &types.GetMetadataResponse{
			Data: &types.SecretMetadata{},
		}, nil
	case "empty secrets list":
		return &types.GetMetadataResponse{
			Data: &types.SecretMetadata{
				Keys: &[]string{},
			},
		}, nil
	case "error response":
		return nil, errors.New("error response")
	}

	paths := []string{
		"pattern1/path1",
		"pattern1/path2",
		"pattern1/path3",
		"pattern2/test/test-secret",
		"pattern2/test/test.secret",
		"pattern2/secret",
		"1secret",
		"pattern2/test/test;secret",
	}
	resp := &types.GetMetadataResponse{
		Data: &types.SecretMetadata{
			Keys: &[]string{},
		},
	}
	if path == "" {
		resp.Data.Keys = &paths
		return resp, nil
	}

	for _, path_elem := range paths {
		posStart := strings.Index(path_elem, path)
		if posStart == 0 {
			if len(path) == len(path_elem) {
				*resp.Data.Keys = append(*resp.Data.Keys, path_elem)
			} else if len(path) < len(path_elem) && path_elem[len(path)] == '/' {
				path_elem = path_elem[len(path)+1:]
				posSlash := strings.Index(path_elem, "/")
				if posSlash >= 0 {
					*resp.Data.Keys = append(*resp.Data.Keys, path_elem[:posSlash])
				} else {
					*resp.Data.Keys = append(*resp.Data.Keys, path_elem)
				}
			}
		}
	}

	return resp, nil
}

func (kmsClient FakeOkmsClient) ListSecretV2(_ context.Context, _ uuid.UUID, _ *uint32, _ *string) (*types.ListSecretV2ResponseWithPagination, error) {
	return nil, nil
}

func (client FakeOkmsClient) WithCustomHeader(_, _ string) *okms.Client {
	return &okms.Client{}
}

func (client FakeOkmsClient) PostSecretV2(_ context.Context, _ uuid.UUID, body types.PostSecretV2Request) (*types.PostSecretV2Response, error) {
	secretDataByte, err := json.Marshal(body.Version.Data)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%s", string(secretDataByte))
}

func (client FakeOkmsClient) PutSecretV2(_ context.Context, _ uuid.UUID, _ string, _ *uint32, body types.PutSecretV2Request) (*types.PutSecretV2Response, error) {
	secretDataByte, err := json.Marshal(body.Version.Data)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%s", string(secretDataByte))
}

func (client FakeOkmsClient) DeleteSecretV2(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
