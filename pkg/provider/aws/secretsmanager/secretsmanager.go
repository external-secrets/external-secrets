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

package secretsmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/tidwall/gjson"
	utilpointer "k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/provider/metrics"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &SecretsManager{}

// SecretsManager is a provider for AWS SecretsManager.
type SecretsManager struct {
	sess         *session.Session
	client       SMInterface
	referentAuth bool
	cache        map[string]*awssm.GetSecretValueOutput
}

// SMInterface is a subset of the smiface api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/secretsmanageriface/
type SMInterface interface {
	ListSecrets(*awssm.ListSecretsInput) (*awssm.ListSecretsOutput, error)
	GetSecretValue(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
	CreateSecretWithContext(aws.Context, *awssm.CreateSecretInput, ...request.Option) (*awssm.CreateSecretOutput, error)
	GetSecretValueWithContext(aws.Context, *awssm.GetSecretValueInput, ...request.Option) (*awssm.GetSecretValueOutput, error)
	PutSecretValueWithContext(aws.Context, *awssm.PutSecretValueInput, ...request.Option) (*awssm.PutSecretValueOutput, error)
	DescribeSecretWithContext(aws.Context, *awssm.DescribeSecretInput, ...request.Option) (*awssm.DescribeSecretOutput, error)
	DeleteSecretWithContext(ctx aws.Context, input *awssm.DeleteSecretInput, opts ...request.Option) (*awssm.DeleteSecretOutput, error)
}

const (
	errUnexpectedFindOperator = "unexpected find operator"
	managedBy                 = "managed-by"
	externalSecrets           = "external-secrets"
)

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("secretsmanager")

// New creates a new SecretsManager client.
func New(sess *session.Session, cfg *aws.Config, referentAuth bool) (*SecretsManager, error) {
	return &SecretsManager{
		sess:         sess,
		client:       awssm.New(sess, cfg),
		referentAuth: referentAuth,
		cache:        make(map[string]*awssm.GetSecretValueOutput),
	}, nil
}

func (sm *SecretsManager) fetch(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (*awssm.GetSecretValueOutput, error) {
	ver := "AWSCURRENT"
	valueFrom := "SECRET"
	if ref.Version != "" {
		ver = ref.Version
	}
	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		valueFrom = "TAG"
	}

	log.Info("fetching secret value", "key", ref.Key, "version", ver, "value", valueFrom)

	cacheKey := fmt.Sprintf("%s#%s#%s", ref.Key, ver, valueFrom)
	if secretOut, found := sm.cache[cacheKey]; found {
		log.Info("found secret in cache", "key", ref.Key, "version", ver)
		return secretOut, nil
	}

	var secretOut *awssm.GetSecretValueOutput
	var err error

	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		describeSecretInput := &awssm.DescribeSecretInput{
			SecretId: &ref.Key,
		}

		descOutput, err := sm.client.DescribeSecretWithContext(ctx, describeSecretInput)
		if err != nil {
			return nil, err
		}
		log.Info("found metadata secret", "key", ref.Key, "output", descOutput)

		jsonTags, err := TagsToJSONString(descOutput.Tags)
		if err != nil {
			return nil, err
		}
		secretOut = &awssm.GetSecretValueOutput{
			ARN:          descOutput.ARN,
			CreatedDate:  descOutput.CreatedDate,
			Name:         descOutput.Name,
			SecretString: &jsonTags,
			VersionId:    &ver,
		}
	} else {
		var getSecretValueInput *awssm.GetSecretValueInput
		if strings.HasPrefix(ver, "uuid/") {
			versionID := strings.TrimPrefix(ver, "uuid/")
			getSecretValueInput = &awssm.GetSecretValueInput{
				SecretId:  &ref.Key,
				VersionId: &versionID,
			}
		} else {
			getSecretValueInput = &awssm.GetSecretValueInput{
				SecretId:     &ref.Key,
				VersionStage: &ver,
			}
		}
		secretOut, err = sm.client.GetSecretValue(getSecretValueInput)
		metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMGetSecretValue, err)
		var nf *awssm.ResourceNotFoundException
		if errors.As(err, &nf) {
			return nil, esv1beta1.NoSecretErr
		}
		if err != nil {
			return nil, err
		}
	}
	sm.cache[cacheKey] = secretOut

	return secretOut, nil
}

func TagsToJSONString(tags []*awssm.Tag) (string, error) {
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

func (sm *SecretsManager) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	secretName := remoteRef.GetRemoteKey()
	secretValue := awssm.GetSecretValueInput{
		SecretId: &secretName,
	}
	secretInput := awssm.DescribeSecretInput{
		SecretId: &secretName,
	}
	awsSecret, err := sm.client.GetSecretValueWithContext(ctx, &secretValue)
	metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMGetSecretValue, err)
	var aerr awserr.Error
	if err != nil {
		if ok := errors.As(err, &aerr); !ok {
			return err
		}
		if aerr.Code() == awssm.ErrCodeResourceNotFoundException {
			return nil
		}
		return err
	}
	data, err := sm.client.DescribeSecretWithContext(ctx, &secretInput)
	metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMDescribeSecret, err)
	if err != nil {
		return err
	}
	if !isManagedByESO(data) {
		return nil
	}
	deleteInput := &awssm.DeleteSecretInput{
		SecretId: awsSecret.ARN,
	}
	_, err = sm.client.DeleteSecretWithContext(ctx, deleteInput)
	metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMDeleteSecret, err)
	return err
}

func (sm *SecretsManager) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	secretName := remoteRef.GetRemoteKey()
	managedBy := managedBy
	externalSecrets := externalSecrets
	externalSecretsTag := []*awssm.Tag{
		{
			Key:   &managedBy,
			Value: &externalSecrets,
		},
	}
	secretRequest := awssm.CreateSecretInput{
		Name:         &secretName,
		SecretBinary: value,
		Tags:         externalSecretsTag,
	}

	secretValue := awssm.GetSecretValueInput{
		SecretId: &secretName,
	}

	secretInput := awssm.DescribeSecretInput{
		SecretId: &secretName,
	}

	awsSecret, err := sm.client.GetSecretValueWithContext(ctx, &secretValue)
	metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMGetSecretValue, err)
	var aerr awserr.Error
	if err != nil {
		if ok := errors.As(err, &aerr); !ok {
			return err
		}
		if aerr.Code() == awssm.ErrCodeResourceNotFoundException {
			_, err = sm.client.CreateSecretWithContext(ctx, &secretRequest)
			metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMCreateSecret, err)
			return err
		}
		return err
	}
	data, err := sm.client.DescribeSecretWithContext(ctx, &secretInput)
	metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMDescribeSecret, err)
	if err != nil {
		return err
	}
	if !isManagedByESO(data) {
		return fmt.Errorf("secret not managed by external-secrets")
	}
	if awsSecret != nil && bytes.Equal(awsSecret.SecretBinary, value) {
		return nil
	}
	input := &awssm.PutSecretValueInput{
		SecretId:     awsSecret.ARN,
		SecretBinary: value,
	}
	_, err = sm.client.PutSecretValueWithContext(ctx, input)
	metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMPutSecretValue, err)
	return err
}

func isManagedByESO(data *awssm.DescribeSecretOutput) bool {
	managedBy := managedBy
	externalSecrets := externalSecrets
	for _, tag := range data.Tags {
		if *tag.Key == managedBy && *tag.Value == externalSecrets {
			return true
		}
	}
	return false
}

// GetAllSecrets syncs multiple secrets from aws provider into a single Kubernetes Secret.
func (sm *SecretsManager) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		return sm.findByName(ctx, ref)
	}
	if len(ref.Tags) > 0 {
		return sm.findByTags(ctx, ref)
	}
	return nil, errors.New(errUnexpectedFindOperator)
}

func (sm *SecretsManager) findByName(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}

	filters := make([]*awssm.Filter, 0)
	if ref.Path != nil {
		filters = append(filters, &awssm.Filter{
			Key: utilpointer.String(awssm.FilterNameStringTypeName),
			Values: []*string{
				ref.Path,
			},
		})
	}

	data := make(map[string][]byte)
	var nextToken *string

	for {
		it, err := sm.client.ListSecrets(&awssm.ListSecretsInput{
			Filters:   filters,
			NextToken: nextToken,
		})
		metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMListSecrets, err)
		if err != nil {
			return nil, err
		}
		log.V(1).Info("aws sm findByName found", "secrets", len(it.SecretList))
		for _, secret := range it.SecretList {
			if !matcher.MatchName(*secret.Name) {
				continue
			}
			log.V(1).Info("aws sm findByName matches", "name", *secret.Name)
			err = sm.fetchAndSet(ctx, data, *secret.Name)
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

func (sm *SecretsManager) findByTags(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	filters := make([]*awssm.Filter, 0)
	for k, v := range ref.Tags {
		filters = append(filters, &awssm.Filter{
			Key: utilpointer.String(awssm.FilterNameStringTypeTagKey),
			Values: []*string{
				utilpointer.String(k),
			},
		}, &awssm.Filter{
			Key: utilpointer.String(awssm.FilterNameStringTypeTagValue),
			Values: []*string{
				utilpointer.String(v),
			},
		})
	}

	if ref.Path != nil {
		filters = append(filters, &awssm.Filter{
			Key: utilpointer.String(awssm.FilterNameStringTypeName),
			Values: []*string{
				ref.Path,
			},
		})
	}

	data := make(map[string][]byte)
	var nextToken *string
	for {
		log.V(1).Info("aws sm findByTag", "nextToken", nextToken)
		it, err := sm.client.ListSecrets(&awssm.ListSecretsInput{
			Filters:   filters,
			NextToken: nextToken,
		})
		metrics.ObserveAPICall(metrics.ProviderAWSSM, metrics.CallAWSSMListSecrets, err)
		if err != nil {
			return nil, err
		}
		log.V(1).Info("aws sm findByTag found", "secrets", len(it.SecretList))
		for _, secret := range it.SecretList {
			err = sm.fetchAndSet(ctx, data, *secret.Name)
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

func (sm *SecretsManager) fetchAndSet(ctx context.Context, data map[string][]byte, name string) error {
	sec, err := sm.fetch(ctx, esv1beta1.ExternalSecretDataRemoteRef{
		Key: name,
	})
	if err != nil {
		return err
	}
	if sec.SecretString != nil {
		data[name] = []byte(*sec.SecretString)
	}
	if sec.SecretBinary != nil {
		data[name] = sec.SecretBinary
	}
	return nil
}

// GetSecret returns a single secret from the provider.
func (sm *SecretsManager) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secretOut, err := sm.fetch(ctx, ref)
	if errors.Is(err, esv1beta1.NoSecretErr) {
		return nil, err
	}
	if err != nil {
		return nil, util.SanitizeErr(err)
	}
	if ref.Property == "" {
		if secretOut.SecretString != nil {
			return []byte(*secretOut.SecretString), nil
		}
		if secretOut.SecretBinary != nil {
			return secretOut.SecretBinary, nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string nor binary for key: %s", ref.Key)
	}
	var payload string
	if secretOut.SecretString != nil {
		payload = *secretOut.SecretString
	}
	if secretOut.SecretBinary != nil {
		payload = string(secretOut.SecretBinary)
	}
	// We need to search if a given key with a . exists before using gjson operations.
	idx := strings.Index(ref.Property, ".")
	if idx > -1 {
		refProperty := strings.ReplaceAll(ref.Property, ".", "\\.")
		val := gjson.Get(payload, refProperty)
		if val.Exists() {
			return []byte(val.String()), nil
		}
	}
	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (sm *SecretsManager) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	log.Info("fetching secret map", "key", ref.Key)
	data, err := sm.GetSecret(ctx, ref)
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

func (sm *SecretsManager) Close(ctx context.Context) error {
	return nil
}

func (sm *SecretsManager) Validate() (esv1beta1.ValidationResult, error) {
	// skip validation stack because it depends on the namespace
	// of the ExternalSecret
	if sm.referentAuth {
		return esv1beta1.ValidationResultUnknown, nil
	}
	_, err := sm.sess.Config.Credentials.Get()
	if err != nil {
		return esv1beta1.ValidationResultError, err
	}
	return esv1beta1.ValidationResultReady, nil
}

func (sm *SecretsManager) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}
