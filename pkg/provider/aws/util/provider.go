//Copyright External Secrets Inc. All Rights Reserved

package util

import (
	"encoding/json"
	"errors"
	"fmt"

	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ssm"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNilStore         = "found nil store"
	errMissingStoreSpec = "store is missing spec"
	errMissingProvider  = "storeSpec is missing provider"
	errInvalidProvider  = "invalid provider spec. Missing AWS field in store %s"
)

// GetAWSProvider does the necessary nil checks on the generic store
// it returns the aws provider or an error.
func GetAWSProvider(store esv1beta1.GenericStore) (*esv1beta1.AWSProvider, error) {
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

func IsReferentSpec(prov esv1beta1.AWSAuth) bool {
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

func SecretTagsToJSONString(tags []*awssm.Tag) (string, error) {
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

func ParameterTagsToJSONString(tags []*ssm.Tag) (string, error) {
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
