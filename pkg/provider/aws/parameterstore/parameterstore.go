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
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/metadata"
)

// Tier defines policy details for PushSecret.
type Tier struct {
	Type     string                `json:"type"`
	Policies *apiextensionsv1.JSON `json:"policies"`
}

// PushSecretMetadataSpec defines the spec for the metadata for PushSecret.
type PushSecretMetadataSpec struct {
	SecretType      string `json:"secretType,omitempty"`
	KMSKeyID        string `json:"kmsKeyID,omitempty"`
	Tier            Tier   `json:"tier,omitempty"`
	EncodeAsDecoded bool   `json:"encodeAsDecoded,omitempty"`
}

// https://github.com/external-secrets/external-secrets/issues/644
var (
	_               esv1beta1.SecretsClient = &ParameterStore{}
	managedBy                               = "managed-by"
	externalSecrets                         = "external-secrets"
	logger                                  = ctrl.Log.WithName("provider").WithName("parameterstore")
)

// ParameterStore is a provider for AWS ParameterStore.
type ParameterStore struct {
	sess         *session.Session
	client       PMInterface
	referentAuth bool
	prefix       string
}

// PMInterface is a subset of the parameterstore api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/ssm/ssmiface/
type PMInterface interface {
	GetParameterWithContext(aws.Context, *ssm.GetParameterInput, ...request.Option) (*ssm.GetParameterOutput, error)
	GetParametersByPathWithContext(aws.Context, *ssm.GetParametersByPathInput, ...request.Option) (*ssm.GetParametersByPathOutput, error)
	PutParameterWithContext(aws.Context, *ssm.PutParameterInput, ...request.Option) (*ssm.PutParameterOutput, error)
	DescribeParametersWithContext(aws.Context, *ssm.DescribeParametersInput, ...request.Option) (*ssm.DescribeParametersOutput, error)
	ListTagsForResourceWithContext(aws.Context, *ssm.ListTagsForResourceInput, ...request.Option) (*ssm.ListTagsForResourceOutput, error)
	DeleteParameterWithContext(ctx aws.Context, input *ssm.DeleteParameterInput, opts ...request.Option) (*ssm.DeleteParameterOutput, error)
}

const (
	errUnexpectedFindOperator = "unexpected find operator"
	errAccessDeniedException  = "AccessDeniedException"
)

// New constructs a ParameterStore Provider that is specific to a store.
func New(sess *session.Session, cfg *aws.Config, prefix string, referentAuth bool) (*ParameterStore, error) {
	return &ParameterStore{
		sess:         sess,
		referentAuth: referentAuth,
		client:       ssm.New(sess, cfg),
		prefix:       prefix,
	}, nil
}

func (pm *ParameterStore) getTagsByName(ctx aws.Context, ref *ssm.GetParameterOutput) ([]*ssm.Tag, error) {
	parameterType := "Parameter"

	parameterTags := ssm.ListTagsForResourceInput{
		ResourceId:   ref.Parameter.Name,
		ResourceType: &parameterType,
	}

	data, err := pm.client.ListTagsForResourceWithContext(ctx, &parameterTags)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSListTagsForResource, err)
	if err != nil {
		return nil, fmt.Errorf("error listing tags %w", err)
	}

	return data.TagList, nil
}

func (pm *ParameterStore) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	secretName := pm.prefix + remoteRef.GetRemoteKey()
	secretValue := ssm.GetParameterInput{
		Name: &secretName,
	}
	existing, err := pm.client.GetParameterWithContext(ctx, &secretValue)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParameter, err)
	var awsError awserr.Error
	ok := errors.As(err, &awsError)
	if err != nil && (!ok || awsError.Code() != ssm.ErrCodeParameterNotFound) {
		return fmt.Errorf("unexpected error getting parameter %v: %w", secretName, err)
	}
	if existing != nil && existing.Parameter != nil {
		tags, err := pm.getTagsByName(ctx, existing)
		if err != nil {
			return fmt.Errorf("error getting the existing tags for the parameter %v: %w", secretName, err)
		}

		isManaged := isManagedByESO(tags)

		if !isManaged {
			// If the secret is not managed by external-secrets, it is "deleted" effectively by all means
			return nil
		}
		deleteInput := &ssm.DeleteParameterInput{
			Name: &secretName,
		}
		_, err = pm.client.DeleteParameterWithContext(ctx, deleteInput)
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSDeleteParameter, err)
		if err != nil {
			return fmt.Errorf("could not delete parameter %v: %w", secretName, err)
		}
	}
	return nil
}

func (pm *ParameterStore) SecretExists(ctx context.Context, pushSecretRef esv1beta1.PushSecretRemoteRef) (bool, error) {
	secretName := pm.prefix + pushSecretRef.GetRemoteKey()

	secretValue := ssm.GetParameterInput{
		Name: &secretName,
	}

	_, err := pm.client.GetParameterWithContext(ctx, &secretValue)

	if err != nil {
		var aerr awserr.Error
		if ok := errors.As(err, &aerr); !ok {
			return false, err
		}
		if aerr.Code() == ssm.ErrCodeResourceNotFoundException {
			return false, nil
		}
		if aerr.Code() == ssm.ErrCodeParameterNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (pm *ParameterStore) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	var (
		value []byte
		err   error
	)

	meta, err := pm.constructMetadataWithDefaults(data.GetMetadata())
	if err != nil {
		return err
	}

	key := data.GetSecretKey()

	if key == "" {
		value, err = pm.encodeSecretData(meta.Spec.EncodeAsDecoded, secret.Data)
		if err != nil {
			return fmt.Errorf("failed to serialize secret content as JSON: %w", err)
		}
	} else {
		value = secret.Data[key]
	}

	secretName := pm.prefix + data.GetRemoteKey()
	secretRequest := ssm.PutParameterInput{
		Name:      ptr.To(pm.prefix + data.GetRemoteKey()),
		Value:     ptr.To(string(value)),
		Type:      ptr.To(meta.Spec.SecretType),
		Overwrite: ptr.To(true),
	}

	if meta.Spec.SecretType == "SecureString" {
		secretRequest.KeyId = &meta.Spec.KMSKeyID
	}

	if meta.Spec.Tier.Type == "Advanced" {
		secretRequest.Tier = ptr.To(meta.Spec.Tier.Type)
		if meta.Spec.Tier.Policies != nil {
			secretRequest.Policies = ptr.To(string(meta.Spec.Tier.Policies.Raw))
		}
	}

	secretValue := ssm.GetParameterInput{
		Name:           &secretName,
		WithDecryption: aws.Bool(true),
	}

	existing, err := pm.client.GetParameterWithContext(ctx, &secretValue)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParameter, err)
	var awsError awserr.Error
	ok := errors.As(err, &awsError)
	if err != nil && (!ok || awsError.Code() != ssm.ErrCodeParameterNotFound) {
		return fmt.Errorf("unexpected error getting parameter %v: %w", secretName, err)
	}

	// If we have a valid parameter returned to us, check its tags
	if existing != nil && existing.Parameter != nil {
		return pm.setExisting(ctx, existing, secretName, value, secretRequest)
	}

	// let's set the secret
	// Do we need to delete the existing parameter on the remote?
	return pm.setManagedRemoteParameter(ctx, secretRequest, true)
}

func (pm *ParameterStore) encodeSecretData(encodeAsDecoded bool, data map[string][]byte) ([]byte, error) {
	if encodeAsDecoded {
		// This will result in map byte slices not being base64 encoded by json.Marshal.
		return utils.JSONMarshal(convertMap(data))
	}

	return utils.JSONMarshal(data)
}

func convertMap(in map[string][]byte) map[string]string {
	m := make(map[string]string)
	for k, v := range in {
		m[k] = string(v)
	}
	return m
}

func (pm *ParameterStore) setExisting(ctx context.Context, existing *ssm.GetParameterOutput, secretName string, value []byte, secretRequest ssm.PutParameterInput) error {
	tags, err := pm.getTagsByName(ctx, existing)
	if err != nil {
		return fmt.Errorf("error getting the existing tags for the parameter %v: %w", secretName, err)
	}

	isManaged := isManagedByESO(tags)

	if !isManaged {
		return errors.New("secret not managed by external-secrets")
	}

	// When fetching a remote SecureString parameter without decrypting, the default value will always be 'sensitive'
	// in this case, no updates will be pushed remotely
	if existing.Parameter.Value != nil && *existing.Parameter.Value == "sensitive" {
		return errors.New("unable to compare 'sensitive' result, ensure to request a decrypted value")
	}

	if existing.Parameter.Value != nil && *existing.Parameter.Value == string(value) {
		return nil
	}

	return pm.setManagedRemoteParameter(ctx, secretRequest, false)
}

func isManagedByESO(tags []*ssm.Tag) bool {
	return slices.ContainsFunc(tags, func(tag *ssm.Tag) bool {
		return *tag.Key == managedBy && *tag.Value == externalSecrets
	})
}

func (pm *ParameterStore) setManagedRemoteParameter(ctx context.Context, secretRequest ssm.PutParameterInput, createManagedByTags bool) error {
	externalSecretsTag := ssm.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}

	overwrite := true
	secretRequest.Overwrite = &overwrite
	if createManagedByTags {
		secretRequest.Tags = append(secretRequest.Tags, &externalSecretsTag)
		overwrite = false
	}

	_, err := pm.client.PutParameterWithContext(ctx, &secretRequest)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSPutParameter, err)
	if err != nil {
		return fmt.Errorf("unexpected error pushing parameter %v: %w", secretRequest.Name, err)
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

// findByName requires `ssm:GetParametersByPath` IAM permission, but the `Resource` scope can be limited.
func (pm *ParameterStore) findByName(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	if ref.Path == nil {
		ref.Path = aws.String("/")
	}
	data := make(map[string][]byte)
	var nextToken *string
	for {
		it, err := pm.client.GetParametersByPathWithContext(
			ctx,
			&ssm.GetParametersByPathInput{
				NextToken:      nextToken,
				Path:           ref.Path,
				Recursive:      aws.Bool(true),
				WithDecryption: aws.Bool(true),
			})
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParametersByPath, err)
		if err != nil {
			/*
				Check for AccessDeniedException when calling `GetParametersByPathWithContext`. If so,
				use fallbackFindByName and `DescribeParametersWithContext`.
				https://github.com/external-secrets/external-secrets/issues/1839#issuecomment-1489023522
			*/
			var awsError awserr.Error
			if errors.As(err, &awsError) && awsError.Code() == errAccessDeniedException {
				logger.Info("GetParametersByPath: access denied. using fallback to describe parameters. It is recommended to add ssm:GetParametersByPath permissions", "path", ref.Path)
				return pm.fallbackFindByName(ctx, ref)
			}

			return nil, err
		}

		for _, param := range it.Parameters {
			if !matcher.MatchName(*param.Name) {
				continue
			}
			data[*param.Name] = []byte(*param.Value)
		}

		nextToken = it.NextToken
		if nextToken == nil {
			break
		}
	}

	return data, nil
}

// fallbackFindByName requires `ssm:DescribeParameters` IAM permission on `"Resource": "*"`.
func (pm *ParameterStore) fallbackFindByName(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
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
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSDescribeParameter, err)
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

// findByTags requires ssm:DescribeParameters,tag:GetResources IAM permission on `"Resource": "*"`.
func (pm *ParameterStore) findByTags(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	filters := make([]*ssm.ParameterStringFilter, 0)
	for k, v := range ref.Tags {
		filters = append(filters, &ssm.ParameterStringFilter{
			Key:    ptr.To(fmt.Sprintf("tag:%s", k)),
			Values: []*string{ptr.To(v)},
			Option: ptr.To("Equals"),
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
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSDescribeParameter, err)
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
		Name:           ptr.To(name),
		WithDecryption: aws.Bool(true),
	})
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParameter, err)
	if err != nil {
		return util.SanitizeErr(err)
	}

	data[name] = []byte(*out.Parameter.Value)
	return nil
}

// GetSecret returns a single secret from the provider.
func (pm *ParameterStore) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var out *ssm.GetParameterOutput
	var err error
	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		out, err = pm.getParameterTags(ctx, ref)
	} else {
		out, err = pm.getParameterValue(ctx, ref)
	}
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParameter, err)
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

func (pm *ParameterStore) getParameterTags(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (*ssm.GetParameterOutput, error) {
	param := ssm.GetParameterOutput{
		Parameter: &ssm.Parameter{
			Name: pm.parameterNameWithVersion(ref),
		},
	}
	tags, err := pm.getTagsByName(ctx, &param)
	if err != nil {
		return nil, err
	}
	json, err := util.ParameterTagsToJSONString(tags)
	if err != nil {
		return nil, err
	}
	out := &ssm.GetParameterOutput{
		Parameter: &ssm.Parameter{
			Value: &json,
		},
	}
	return out, nil
}

func (pm *ParameterStore) getParameterValue(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (*ssm.GetParameterOutput, error) {
	out, err := pm.client.GetParameterWithContext(ctx, &ssm.GetParameterInput{
		Name:           pm.parameterNameWithVersion(ref),
		WithDecryption: aws.Bool(true),
	})

	return out, err
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

func (pm *ParameterStore) parameterNameWithVersion(ref esv1beta1.ExternalSecretDataRemoteRef) *string {
	name := pm.prefix + ref.Key
	if ref.Version != "" {
		// see docs: https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-paramstore-versions.html#reference-parameter-version
		name += ":" + ref.Version
	}
	return &name
}

func (pm *ParameterStore) Close(_ context.Context) error {
	return nil
}

func (pm *ParameterStore) Validate() (esv1beta1.ValidationResult, error) {
	// skip validation stack because it depends on the namespace
	// of the ExternalSecret
	if pm.referentAuth {
		return esv1beta1.ValidationResultUnknown, nil
	}
	_, err := pm.sess.Config.Credentials.Get()
	if err != nil {
		return esv1beta1.ValidationResultError, err
	}
	return esv1beta1.ValidationResultReady, nil
}

func (pm *ParameterStore) constructMetadataWithDefaults(data *apiextensionsv1.JSON) (*metadata.PushSecretMetadata[PushSecretMetadataSpec], error) {
	var (
		meta *metadata.PushSecretMetadata[PushSecretMetadataSpec]
		err  error
	)

	meta, err = metadata.ParseMetadataParameters[PushSecretMetadataSpec](data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	if meta == nil {
		meta = &metadata.PushSecretMetadata[PushSecretMetadataSpec]{}
	}

	if meta.Spec.Tier.Type == "" {
		meta.Spec.Tier.Type = "Standard"
	}

	if meta.Spec.SecretType == "" {
		meta.Spec.SecretType = "String"
	}

	if meta.Spec.KMSKeyID == "" {
		meta.Spec.KMSKeyID = "alias/aws/ssm"
	}

	return meta, nil
}
