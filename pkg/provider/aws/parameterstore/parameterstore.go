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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/metadata"
)

// Tier defines policy details for PushSecret.
type Tier struct {
	Type     ssmTypes.ParameterTier `json:"type"`
	Policies *apiextensionsv1.JSON  `json:"policies"`
}

// PushSecretMetadataSpec defines the spec for the metadata for PushSecret.
type PushSecretMetadataSpec struct {
	SecretType      ssmTypes.ParameterType `json:"secretType,omitempty"`
	KMSKeyID        string                 `json:"kmsKeyID,omitempty"`
	Tier            Tier                   `json:"tier,omitempty"`
	EncodeAsDecoded bool                   `json:"encodeAsDecoded,omitempty"`
	Tags            map[string]string      `json:"tags,omitempty"`
	Description     string                 `json:"description,omitempty"`
}

// https://github.com/external-secrets/external-secrets/issues/644
var (
	_               esv1.SecretsClient = &ParameterStore{}
	managedBy                          = "managed-by"
	externalSecrets                    = "external-secrets"
	logger                             = ctrl.Log.WithName("provider").WithName("parameterstore")
)

// ParameterStore is a provider for AWS ParameterStore.
type ParameterStore struct {
	cfg          *aws.Config
	client       PMInterface
	referentAuth bool
	prefix       string
}

// PMInterface is a subset of the parameterstore api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/ssm/ssmiface/
type PMInterface interface {
	GetParameter(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParametersByPath(ctx context.Context, input *ssm.GetParametersByPathInput, opts ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
	PutParameter(ctx context.Context, input *ssm.PutParameterInput, opts ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
	DescribeParameters(ctx context.Context, input *ssm.DescribeParametersInput, opts ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
	ListTagsForResource(ctx context.Context, input *ssm.ListTagsForResourceInput, opts ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error)
	RemoveTagsFromResource(ctx context.Context, params *ssm.RemoveTagsFromResourceInput, optFns ...func(*ssm.Options)) (*ssm.RemoveTagsFromResourceOutput, error)
	AddTagsToResource(ctx context.Context, params *ssm.AddTagsToResourceInput, optFns ...func(*ssm.Options)) (*ssm.AddTagsToResourceOutput, error)
	DeleteParameter(ctx context.Context, input *ssm.DeleteParameterInput, opts ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
}

const (
	errUnexpectedFindOperator    = "unexpected find operator"
	errCodeAccessDeniedException = "AccessDeniedException"
)

// New constructs a ParameterStore Provider that is specific to a store.
func New(_ context.Context, cfg *aws.Config, prefix string, referentAuth bool) (*ParameterStore, error) {
	return &ParameterStore{
		cfg:          cfg,
		referentAuth: referentAuth,
		client: ssm.NewFromConfig(*cfg, func(o *ssm.Options) {
			o.EndpointResolverV2 = customEndpointResolver{}
		}),
		prefix: prefix,
	}, nil
}

func (pm *ParameterStore) getTagsByName(ctx context.Context, ref *ssm.GetParameterOutput) (map[string]string, error) {
	parameterType := "Parameter"

	parameterTags := ssm.ListTagsForResourceInput{
		ResourceId:   ref.Parameter.Name,
		ResourceType: ssmTypes.ResourceTypeForTagging(parameterType),
	}

	data, err := pm.client.ListTagsForResource(ctx, &parameterTags)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSListTagsForResource, err)
	if err != nil {
		return nil, fmt.Errorf("error listing tags %w", err)
	}

	tags := map[string]string{}
	for _, tag := range data.TagList {
		tags[*tag.Key] = *tag.Value
	}
	return tags, nil
}

func (pm *ParameterStore) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	secretName := pm.prefix + remoteRef.GetRemoteKey()
	secretValue := ssm.GetParameterInput{
		Name: &secretName,
	}
	existing, err := pm.client.GetParameter(ctx, &secretValue)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParameter, err)
	var parameterNotFoundErr *ssmTypes.ParameterNotFound
	ok := errors.As(err, &parameterNotFoundErr)
	if err != nil && !ok {
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
		_, err = pm.client.DeleteParameter(ctx, deleteInput)
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSDeleteParameter, err)
		if err != nil {
			return fmt.Errorf("could not delete parameter %v: %w", secretName, err)
		}
	}
	return nil
}

func (pm *ParameterStore) SecretExists(ctx context.Context, pushSecretRef esv1.PushSecretRemoteRef) (bool, error) {
	secretName := pm.prefix + pushSecretRef.GetRemoteKey()

	secretValue := ssm.GetParameterInput{
		Name: &secretName,
	}

	_, err := pm.client.GetParameter(ctx, &secretValue)

	var resourceNotFoundErr *ssmTypes.ResourceNotFoundException
	var parameterNotFoundErr *ssmTypes.ParameterNotFound

	if err != nil {
		if errors.As(err, &resourceNotFoundErr) {
			return false, nil
		}
		if errors.As(err, &parameterNotFoundErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (pm *ParameterStore) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
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

	tags := make([]ssmTypes.Tag, 0, len(meta.Spec.Tags))

	for k, v := range meta.Spec.Tags {
		tags = append(tags, ssmTypes.Tag{
			Key:   ptr.To(k),
			Value: ptr.To(v),
		})
	}

	secretName := pm.prefix + data.GetRemoteKey()
	secretRequest := ssm.PutParameterInput{
		Name:        ptr.To(pm.prefix + data.GetRemoteKey()),
		Value:       ptr.To(string(value)),
		Type:        meta.Spec.SecretType,
		Overwrite:   ptr.To(true),
		Description: ptr.To(meta.Spec.Description),
	}

	if meta.Spec.SecretType == "SecureString" {
		secretRequest.KeyId = &meta.Spec.KMSKeyID
	}

	if meta.Spec.Tier.Type == ssmTypes.ParameterTierAdvanced {
		secretRequest.Tier = meta.Spec.Tier.Type
		if meta.Spec.Tier.Policies != nil {
			secretRequest.Policies = ptr.To(string(meta.Spec.Tier.Policies.Raw))
		}
	}

	secretValue := ssm.GetParameterInput{
		Name:           &secretName,
		WithDecryption: aws.Bool(true),
	}

	existing, err := pm.client.GetParameter(ctx, &secretValue)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParameter, err)
	var parameterNotFoundErr *ssmTypes.ParameterNotFound
	ok := errors.As(err, &parameterNotFoundErr)
	if err != nil && !ok {
		return fmt.Errorf("unexpected error getting parameter %v: %w", secretName, err)
	}

	// If we have a valid parameter returned to us, check its tags
	if existing != nil && existing.Parameter != nil {
		return pm.setExisting(ctx, existing, secretName, value, secretRequest, meta.Spec.Tags)
	}

	// let's set the secret
	// Do we need to delete the existing parameter on the remote?
	return pm.setManagedRemoteParameter(ctx, secretRequest, tags, true)
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

func (pm *ParameterStore) setExisting(ctx context.Context, existing *ssm.GetParameterOutput, secretName string, value []byte, secretRequest ssm.PutParameterInput, metaTags map[string]string) error {
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

	err = pm.setManagedRemoteParameter(ctx, secretRequest, []ssmTypes.Tag{}, false)
	if err != nil {
		return err
	}

	tagKeysToRemove := util.FindTagKeysToRemove(tags, metaTags)
	if len(tagKeysToRemove) > 0 {
		_, err = pm.client.RemoveTagsFromResource(ctx, &ssm.RemoveTagsFromResourceInput{
			ResourceId:   existing.Parameter.Name,
			ResourceType: ssmTypes.ResourceTypeForTaggingParameter,
			TagKeys:      tagKeysToRemove,
		})
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSRemoveTagsParameter, err)
		if err != nil {
			return err
		}
	}

	tagsToUpdate, isModified := computeTagsToUpdate(tags, metaTags)
	if isModified {
		_, err = pm.client.AddTagsToResource(ctx, &ssm.AddTagsToResourceInput{
			ResourceId:   existing.Parameter.Name,
			ResourceType: ssmTypes.ResourceTypeForTaggingParameter,
			Tags:         tagsToUpdate,
		})
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSAddTagsParameter, err)
		if err != nil {
			return err
		}
	}

	return nil
}

func isManagedByESO(tags map[string]string) bool {
	return tags[managedBy] == externalSecrets
}

func (pm *ParameterStore) setManagedRemoteParameter(ctx context.Context, secretRequest ssm.PutParameterInput, tags []ssmTypes.Tag, createManagedByTags bool) error {
	overwrite := true
	secretRequest.Overwrite = &overwrite
	if createManagedByTags {
		secretRequest.Tags = append(secretRequest.Tags, tags...)
		overwrite = false
	}

	_, err := pm.client.PutParameter(ctx, &secretRequest)
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSPutParameter, err)
	if err != nil {
		return fmt.Errorf("unexpected error pushing parameter %v: %w", secretRequest.Name, err)
	}
	return nil
}

// GetAllSecrets fetches information from multiple secrets into a single kubernetes secret.
func (pm *ParameterStore) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		return pm.findByName(ctx, ref)
	}
	if ref.Tags != nil {
		return pm.findByTags(ctx, ref)
	}
	return nil, errors.New(errUnexpectedFindOperator)
}

// findByName requires `ssm:GetParametersByPath` IAM permission, but the `Resource` scope can be limited.
func (pm *ParameterStore) findByName(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
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
		it, err := pm.client.GetParametersByPath(
			ctx,
			&ssm.GetParametersByPathInput{
				NextToken:      nextToken,
				Path:           ref.Path,
				Recursive:      aws.Bool(true),
				WithDecryption: aws.Bool(true),
			})
		metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParametersByPath, err)
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == errCodeAccessDeniedException {
				logger.Info("GetParametersByPath: access denied. using fallback to describe parameters. It is recommended to add ssm:GetParametersByPath permissions", "path", ref.Path)
				return pm.fallbackFindByName(ctx, ref)
			}

			return nil, fmt.Errorf("fetching parameters by path %s: %w", *ref.Path, err)
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
func (pm *ParameterStore) fallbackFindByName(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	pathFilter := make([]ssmTypes.ParameterStringFilter, 0)
	if ref.Path != nil {
		pathFilter = append(pathFilter, ssmTypes.ParameterStringFilter{
			Key:    aws.String("Path"),
			Option: aws.String("Recursive"),
			Values: []string{*ref.Path},
		})
	}
	data := make(map[string][]byte)
	var nextToken *string
	for {
		it, err := pm.client.DescribeParameters(
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
func (pm *ParameterStore) findByTags(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	filters := make([]ssmTypes.ParameterStringFilter, 0)
	for k, v := range ref.Tags {
		filters = append(filters, ssmTypes.ParameterStringFilter{
			Key:    ptr.To(fmt.Sprintf("tag:%s", k)),
			Values: []string{v},
			Option: ptr.To("Equals"),
		})
	}

	if ref.Path != nil {
		filters = append(filters, ssmTypes.ParameterStringFilter{
			Key:    aws.String("Path"),
			Option: aws.String("Recursive"),
			Values: []string{*ref.Path},
		})
	}

	data := make(map[string][]byte)
	var nextToken *string
	for {
		it, err := pm.client.DescribeParameters(
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
	out, err := pm.client.GetParameter(ctx, &ssm.GetParameterInput{
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
func (pm *ParameterStore) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var out *ssm.GetParameterOutput
	var err error
	if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
		out, err = pm.getParameterTags(ctx, ref)
	} else {
		out, err = pm.getParameterValue(ctx, ref)
	}
	metrics.ObserveAPICall(constants.ProviderAWSPS, constants.CallAWSPSGetParameter, err)
	nsf := esv1.NoSecretError{}
	var nf *ssmTypes.ParameterNotFound
	if errors.As(err, &nf) || errors.As(err, &nsf) {
		return nil, esv1.NoSecretErr
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

func (pm *ParameterStore) getParameterTags(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (*ssm.GetParameterOutput, error) {
	param := ssm.GetParameterOutput{
		Parameter: &ssmTypes.Parameter{
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
		Parameter: &ssmTypes.Parameter{
			Value: &json,
		},
	}
	return out, nil
}

func (pm *ParameterStore) getParameterValue(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (*ssm.GetParameterOutput, error) {
	out, err := pm.client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           pm.parameterNameWithVersion(ref),
		WithDecryption: aws.Bool(true),
	})

	return out, err
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (pm *ParameterStore) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

func (pm *ParameterStore) parameterNameWithVersion(ref esv1.ExternalSecretDataRemoteRef) *string {
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

func (pm *ParameterStore) Validate() (esv1.ValidationResult, error) {
	// skip validation stack because it depends on the namespace
	// of the ExternalSecret
	if pm.referentAuth {
		return esv1.ValidationResultUnknown, nil
	}
	_, err := pm.cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
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

	if meta.Spec.Description == "" {
		meta.Spec.Description = fmt.Sprintf("secret '%s:%s'", managedBy, externalSecrets)
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

	if len(meta.Spec.Tags) > 0 {
		if _, exists := meta.Spec.Tags[managedBy]; exists {
			return nil, fmt.Errorf("error parsing tags in metadata: Cannot specify a '%s' tag", managedBy)
		}
	} else {
		meta.Spec.Tags = make(map[string]string)
	}
	// always add the managedBy tag
	meta.Spec.Tags[managedBy] = externalSecrets

	return meta, nil
}

// computeTagsToUpdate compares the current tags with the desired metaTags and returns a slice of ssmTypes.Tag
// that should be set on the resource. It also returns a boolean indicating if any tag was added or modified.
func computeTagsToUpdate(tags, metaTags map[string]string) ([]ssmTypes.Tag, bool) {
	result := make([]ssmTypes.Tag, 0, len(metaTags))
	modified := false
	for k, v := range metaTags {
		if _, exists := tags[k]; !exists || tags[k] != v {
			modified = true
		}
		result = append(result, ssmTypes.Tag{
			Key:   ptr.To(k),
			Value: ptr.To(v),
		})
	}
	return result, modified
}
