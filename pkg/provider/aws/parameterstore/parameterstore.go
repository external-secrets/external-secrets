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
package parameterstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/tidwall/gjson"
	utilpointer "k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &ParameterStore{}

// ParameterStore is a provider for AWS ParameterStore.
type ParameterStore struct {
	sess   *session.Session
	client PMInterface
}

// PMInterface is a subset of the parameterstore api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/ssm/ssmiface/
type PMInterface interface {
	GetParameter(*ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
	DescribeParameters(*ssm.DescribeParametersInput) (*ssm.DescribeParametersOutput, error)
}

const (
	errUnexpectedFindOperator = "unexpected find operator"
)

// New constructs a ParameterStore Provider that is specific to a store.
func New(sess *session.Session) (*ParameterStore, error) {
	return &ParameterStore{
		sess:   sess,
		client: ssm.New(sess),
	}, nil
}

// Empty GetAllSecrets.
func (pm *ParameterStore) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		return pm.findByName(ref)
	}
	if ref.Tags != nil {
		return pm.findByTags(ref)
	}
	return nil, errors.New(errUnexpectedFindOperator)
}

func (pm *ParameterStore) findByName(ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	pathFilter := make([]*ssm.ParameterStringFilter, 0)
	if ref.Path != nil {
		pathFilter = append(pathFilter, &ssm.ParameterStringFilter{
			Key:    aws.String("Path"),
			Option: aws.String("Recursive"),
			Values: []*string{ref.Path},
		})
	}
	data := make(map[string][]byte)
	var nextToken *string
	for {
		it, err := pm.client.DescribeParameters(&ssm.DescribeParametersInput{
			NextToken:        nextToken,
			ParameterFilters: pathFilter,
		})
		if err != nil {
			return nil, err
		}
		for _, param := range it.Parameters {
			if !matcher.MatchName(*param.Name) {
				continue
			}
			err = pm.fetchAndSet(data, *param.Name)
			if err != nil {
				return nil, err
			}
		}
		nextToken = it.NextToken
		if nextToken == nil {
			break
		}
	}

	return utils.ConvertKeys(ref.ConversionStrategy, data)
}

func (pm *ParameterStore) findByTags(ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	filters := make([]*ssm.ParameterStringFilter, 0)
	for k, v := range ref.Tags {
		filters = append(filters, &ssm.ParameterStringFilter{
			Key:    utilpointer.StringPtr(fmt.Sprintf("tag:%s", k)),
			Values: []*string{utilpointer.StringPtr(v)},
			Option: utilpointer.StringPtr("Equals"),
		})
	}

	if ref.Path != nil {
		filters = append(filters, &ssm.ParameterStringFilter{
			Key:    aws.String("Path"),
			Option: aws.String("Recursive"),
			Values: []*string{ref.Path},
		})
	}

	data := make(map[string][]byte)
	var nextToken *string
	for {
		it, err := pm.client.DescribeParameters(&ssm.DescribeParametersInput{
			ParameterFilters: filters,
			NextToken:        nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, param := range it.Parameters {
			err = pm.fetchAndSet(data, *param.Name)
			if err != nil {
				return nil, err
			}
		}
		nextToken = it.NextToken
		if nextToken == nil {
			break
		}
	}

	return utils.ConvertKeys(ref.ConversionStrategy, data)
}

func (pm *ParameterStore) fetchAndSet(data map[string][]byte, name string) error {
	out, err := pm.client.GetParameter(&ssm.GetParameterInput{
		Name:           utilpointer.StringPtr(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return util.SanitizeErr(err)
	}

	data[name] = []byte(*out.Parameter.Value)
	return nil
}

// GetSecret returns a single secret from the provider.
func (pm *ParameterStore) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	out, err := pm.client.GetParameter(&ssm.GetParameterInput{
		Name:           &ref.Key,
		WithDecryption: aws.Bool(true),
	})

	var nf *ssm.ParameterNotFound
	if errors.As(err, &nf) {
		return nil, esv1beta1.NoSecretErr
	}
	if err != nil {
		return nil, util.SanitizeErr(err)
	}
	if ref.Property == "" {
		if out.Parameter.Value != nil {
			return []byte(*out.Parameter.Value), nil
		}
		return nil, fmt.Errorf("invalid secret received. parameter value is nil for key: %s", ref.Key)
	}
	idx := strings.Index(ref.Property, ".")
	if idx > 0 {
		refProperty := strings.ReplaceAll(ref.Property, ".", "\\.")
		val := gjson.Get(*out.Parameter.Value, refProperty)
		if val.Exists() {
			return []byte(val.String()), nil
		}
	}
	val := gjson.Get(*out.Parameter.Value, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (pm *ParameterStore) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := pm.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
	}
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

func (pm *ParameterStore) Close(ctx context.Context) error {
	return nil
}

func (pm *ParameterStore) Validate() (esv1beta1.ValidationResult, error) {
	_, err := pm.sess.Config.Credentials.Get()
	if err != nil {
		return esv1beta1.ValidationResultError, err
	}
	return esv1beta1.ValidationResultReady, nil
}
