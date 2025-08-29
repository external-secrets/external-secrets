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

package util

import (
	"encoding/json"
	"errors"
	"fmt"

	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errNilStore         = "found nil store"
	errMissingStoreSpec = "store is missing spec"
	errMissingProvider  = "storeSpec is missing provider"
	errInvalidProvider  = "invalid provider spec. Missing AWS field in store %s"
)

// GetAWSProvider does the necessary nil checks on the generic store
// it returns the aws provider or an error.
func GetAWSProvider(store esv1.GenericStore) (*esv1.AWSProvider, error) {
	if store == nil {
		return nil, errors.New(errNilStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errMissingStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errMissingProvider)
	}
	prov := spc.Provider.AWS
	if prov == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}
	return prov, nil
}

func IsReferentSpec(prov esv1.AWSAuth) bool {
	if prov.JWTAuth != nil && prov.JWTAuth.ServiceAccountRef != nil && prov.JWTAuth.ServiceAccountRef.Namespace == nil {
		return true
	}
	if prov.SecretRef != nil &&
		(prov.SecretRef.AccessKeyID.Namespace == nil ||
			prov.SecretRef.SecretAccessKey.Namespace == nil ||
			(prov.SecretRef.SessionToken != nil && prov.SecretRef.SessionToken.Namespace == nil)) {
		return true
	}

	return false
}

func SecretTagsToJSONString(tags []awssm.Tag) (string, error) {
	tagMap := make(map[string]string, len(tags))
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	byteArr, err := json.Marshal(tagMap)
	if err != nil {
		return "", err
	}

	return string(byteArr), nil
}

func ParameterTagsToJSONString(tags map[string]string) (string, error) {
	byteArr, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}

	return string(byteArr), nil
}

// FindTagKeysToRemove returns a slice of tag keys that exist in the current tags
// but are not present in the desired metaTags. These keys should be removed to
// synchronize the tags with the desired state.
func FindTagKeysToRemove(tags, metaTags map[string]string) []string {
	var diff []string
	for key, _ := range tags {
		if _, ok := metaTags[key]; !ok {
			diff = append(diff, key)
		}
	}
	return diff
}
