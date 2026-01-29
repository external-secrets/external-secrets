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

package ovh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go"
	"github.com/tidwall/gjson"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func getSecretWithOvhSDK(ctx context.Context, kmsClient OkmsClient, okmsID uuid.UUID, ref esv1.ExternalSecretDataRemoteRef) ([]byte, *uint32, error) {
	// Check if the remoteRef key is empty.
	if ref.Key == "" {
		return []byte{}, nil, errors.New("remote key cannot be empty (spec.data.remoteRef.key)")
	}

	// Check MetaDataPolicy (not supported).
	if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
		return []byte{}, nil, errors.New("fetch metadata policy not supported")
	}

	// Decode the secret version.
	versionAddr, err := decodeSecretVersion(ref.Version)
	if err != nil {
		return []byte{}, nil, err
	}

	// Retrieve the KMS secret.
	includeData := true
	secret, err := kmsClient.GetSecretV2(ctx, okmsID, ref.Key, versionAddr, &includeData)
	if err != nil {
		return []byte{}, nil, handleOkmsError(err)
	}
	if secret == nil {
		return []byte{}, nil, esv1.NoSecretErr
	}
	if secret.Version == nil || secret.Version.Data == nil {
		return []byte{}, nil, errors.New("secret version data is missing")
	}

	// Retrieve KMS Secret property if needed.
	var secretData []byte

	if ref.Property == "" {
		secretData, err = json.Marshal(secret.Version.Data)
	} else {
		secretData, err = getPropertyValue(*secret.Version.Data, ref.Property)
	}

	var currentVersion *uint32
	if secret.Metadata != nil {
		currentVersion = secret.Metadata.CurrentVersion
	}
	return secretData, currentVersion, err
}

// Decode a secret version.
//
// Returns nil if no version is provided; in that case, the OVH SDK uses the latest version.
func decodeSecretVersion(strVersion string) (*uint32, error) {
	var version uint32

	if strVersion != "" {
		v, err := strconv.Atoi(strVersion)
		if err != nil {
			return nil, err
		}
		if v < 0 || v > math.MaxUint32 {
			return nil, errors.New("overflow occurred while decoding secret version")
		}
		version = uint32(v)
	} else {
		return nil, nil
	}

	return &version, nil
}

// Retrieve the value of the secret property.
func getPropertyValue(data map[string]any, property string) ([]byte, error) {
	// Marshal data into bytes so it can be passed to gjson.Get.
	secretData, err := json.Marshal(data)
	if err != nil {
		return []byte{}, err
	}

	// Retrieve the property value if it exists.
	secretDataResult := gjson.Get(string(secretData), property)
	if !secretDataResult.Exists() {
		return []byte{}, fmt.Errorf("secret property \"%s\" not found", property)
	}

	return []byte(secretDataResult.String()), nil
}

// Returns an okms.KmsError struct representing the KMS response
// (error_code, error_id, errors, request_id).
func handleOkmsError(err error) error {
	okmsError := okms.AsKmsError(err)

	if okmsError == nil {
		return fmt.Errorf("failed to parse okms error: %w", err)
	} else if okmsError.ErrorCode == 17125377 { // 17125377: returned by OKMS when secret was not found
		return esv1.NoSecretErr
	}
	return okmsError
}
