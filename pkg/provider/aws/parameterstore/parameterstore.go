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
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/tidwall/gjson"
	utilpointer "k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
)

// https://github.com/external-secrets/external-secrets/issues/644
var (
	_               esv1beta1.SecretsClient = &ParameterStore{}
	managedBy                               = "managed-by"
	externalSecrets                         = "external-secrets"
)

// ParameterStore is a provider for AWS ParameterStore.
type ParameterStore struct {
	sess   *session.Session
	client PMInterface
}

// PMInterface is a subset of the parameterstore api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/ssm/ssmiface/
type PMInterface interface {
	GetParameterWithContext(aws.Context, *ssm.GetParameterInput, ...request.Option) (*ssm.GetParameterOutput, error)
	PutParameterWithContext(aws.Context, *ssm.PutParameterInput, ...request.Option) (*ssm.PutParameterOutput, error)
	DescribeParametersWithContext(aws.Context, *ssm.DescribeParametersInput, ...request.Option) (*ssm.DescribeParametersOutput, error)
	ListTagsForResourceWithContext(aws.Context, *ssm.ListTagsForResourceInput, ...request.Option) (*ssm.ListTagsForResourceOutput, error)
}

const (
	errUnexpectedFindOperator = "unexpected find operator"
)

// New constructs a ParameterStore Provider that is specific to a store.
func New(sess *session.Session, cfg *aws.Config) (*ParameterStore, error) {
	return &ParameterStore{
		sess:   sess,
		client: ssm.New(sess, cfg),
	}, nil
}

func (pm *ParameterStore) getTagsByName(ctx aws.Context, ref *ssm.GetParameterOutput) ([]*ssm.Tag, error) {
	parameterTags := ssm.ListTagsForResourceInput{
		ResourceId:   ref.Parameter.ARN,
		ResourceType: ref.Parameter.Type,
	}

	data, err := pm.client.ListTagsForResourceWithContext(ctx, &parameterTags)

	if err != nil {
		return nil, fmt.Errorf("error listing tags %w", err)
	}

	return data.TagList, nil
}

func (pm *ParameterStore) SetSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	parameterType := "String"

	stringValue := string(value)
	secretName := remoteRef.GetRemoteKey()

	secretRequest := ssm.PutParameterInput{
		Name:  &secretName,
		Value: &stringValue,
		Type:  &parameterType,
	}

	secretValue := ssm.GetParameterInput{
		Name: &secretName,
	}

	existing, err := pm.client.GetParameterWithContext(ctx, &secretValue)

	if err != nil && err.Error() != ssm.ErrCodeParameterNotFound {
		// TODO: give some more context to the error
		return err
	}

	// If we have a valid parameter returned to us, check its tags
	if existing != nil && existing.Parameter != nil {
		fmt.Println("The existing value contains data:", existing.String())
		tags, err := pm.getTagsByName(ctx, existing)
		if err != nil {
			return fmt.Errorf("error getting the existing tags for the parameter: %w", err)
		}

		isManaged := isManagedByESO(tags)

		if !isManaged {
			// TODO Can we refactor this error message to a higher scope to stop duplicates
			return fmt.Errorf("secret not managed by external-secrets")
		}
	}

	// let's set the secret
	// Do we need to delete the existing parameter on the remote?
	return pm.setManagedRemoteParameter(ctx, secretRequest)
}

func isManagedByESO(tags []*ssm.Tag) (isManaged bool) {
	for _, tag := range tags {
		if *tag.Key == managedBy && *tag.Value == externalSecrets {
			isManaged = true
		}
	}
	return
}

func (pm *ParameterStore) setManagedRemoteParameter(ctx context.Context, secretRequest ssm.PutParameterInput) error {
	externalSecretsTag := ssm.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}

	isManaged := isManagedByESO(secretRequest.Tags)

	if !isManaged {
		secretRequest.Tags = append(secretRequest.Tags, &externalSecretsTag)
	}

	_, err := pm.client.PutParameterWithContext(ctx, &secretRequest)
	if err != nil {
		// TODO: Edit test so that we can pass more context and have the test not fail because it's not exactly the error `err`
		// return fmt.fmErrorf("failed to push the parameter: %w", err)
		return err
	}
	return nil
}

// GetAllSecrets fetches information from multiple secrets into a single kubernetes secret.
func (pm *ParameterStore) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		return pm.findByName(ctx, ref)
	}
	if ref.Tags != nil {
		return pm.findByTags(ctx, ref)
	}
	return nil, errors.New(errUnexpectedFindOperator)
}

func (pm *ParameterStore) findByName(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
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
		it, err := pm.client.DescribeParametersWithContext(
			ctx,
			&ssm.DescribeParametersInput{
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
			err = pm.fetchAndSet(ctx, data, *param.Name)
			if err != nil {
				return nil, err
			}
		}
		nextToken = it.NextToken
		if nextToken == nil {
			break
		}
	}

	return data, nil
}

func (pm *ParameterStore) findByTags(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
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
		it, err := pm.client.DescribeParametersWithContext(
			ctx,
			&ssm.DescribeParametersInput{
				ParameterFilters: filters,
				NextToken:        nextToken,
			})
		if err != nil {
			return nil, err
		}
		for _, param := range it.Parameters {
			err = pm.fetchAndSet(ctx, data, *param.Name)
			if err != nil {
				return nil, err
			}
		}
		nextToken = it.NextToken
		if nextToken == nil {
			break
		}
	}

	return data, nil
}

func (pm *ParameterStore) fetchAndSet(ctx context.Context, data map[string][]byte, name string) error {
	out, err := pm.client.GetParameterWithContext(ctx, &ssm.GetParameterInput{
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
	out, err := pm.client.GetParameterWithContext(ctx, &ssm.GetParameterInput{
		Name:           &ref.Key,
		WithDecryption: aws.Bool(true),
	})

	nsf := esv1beta1.NoSecretError{}
	var nf *ssm.ParameterNotFound
	if errors.As(err, &nf) || errors.As(err, &nsf) {
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
	if idx > -1 {
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
	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
	}
	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		err = json.Unmarshal(v, &strVal)
		if err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
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
