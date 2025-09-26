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

package secretmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/tidwall/gjson"
	"google.golang.org/api/iterator"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/util/locks"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/metadata"
)

const (
	CloudPlatformRole               = "https://www.googleapis.com/auth/cloud-platform"
	defaultVersion                  = "latest"
	errGCPSMStore                   = "received invalid GCPSM SecretStore resource"
	errUnableGetCredentials         = "unable to get credentials: %w"
	errClientClose                  = "unable to close SecretManager client: %w"
	errUnableProcessJSONCredentials = "failed to process the provided JSON credentials: %w"
	errUnableCreateGCPSMClient      = "failed to create GCP secretmanager client: %w"
	errUninitalizedGCPProvider      = "provider GCP is not initialized"
	errClientGetSecretAccess        = "unable to access Secret from SecretManager Client: %w"
	errJSONSecretUnmarshal          = "unable to unmarshal secret from JSON: %w"

	errInvalidStore           = "invalid store"
	errInvalidStoreSpec       = "invalid store spec"
	errInvalidStoreProv       = "invalid store provider"
	errInvalidGCPProv         = "invalid gcp secrets manager provider"
	errInvalidAuthSecretRef   = "invalid auth secret data: %w"
	errInvalidWISARef         = "invalid workload identity service account reference: %w"
	errUnexpectedFindOperator = "unexpected find operator"

	managedByKey   = "managed-by"
	managedByValue = "external-secrets"

	providerName               = "GCPSecretManager"
	topicsKey                  = "topics"
	globalSecretPath           = "projects/%s/secrets/%s"
	globalSecretParentPath     = "projects/%s"
	regionalSecretParentPath   = "projects/%s/locations/%s"
	regionalSecretPath         = "projects/%s/locations/%s/secrets/%s"
	globalSecretVersionsPath   = "projects/%s/secrets/%s/versions/%s"
	regionalSecretVersionsPath = "projects/%s/locations/%s/secrets/%s/versions/%s"
)

type Client struct {
	smClient  GoogleSecretManagerClient
	kube      kclient.Client
	store     *esv1.GCPSMProvider
	storeKind string

	// namespace of the external secret
	namespace        string
	workloadIdentity *workloadIdentity
}

type GoogleSecretManagerClient interface {
	DeleteSecret(ctx context.Context, req *secretmanagerpb.DeleteSecretRequest, opts ...gax.CallOption) error
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	ListSecrets(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, opts ...gax.CallOption) *secretmanager.SecretIterator
	AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
	CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	Close() error
	GetSecret(ctx context.Context, req *secretmanagerpb.GetSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	UpdateSecret(context.Context, *secretmanagerpb.UpdateSecretRequest, ...gax.CallOption) (*secretmanagerpb.Secret, error)
	ListSecretVersions(ctx context.Context, req *secretmanagerpb.ListSecretVersionsRequest, opts ...gax.CallOption) *secretmanager.SecretVersionIterator
}

var log = ctrl.Log.WithName("provider").WithName("gcp").WithName("secretsmanager")

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	name := getName(c.store.ProjectID, c.store.Location, remoteRef.GetRemoteKey())
	gcpSecret, err := c.smClient.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: name,
	})
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMGetSecret, err)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}

		return err
	}

	if manager, ok := gcpSecret.Labels[managedByKey]; !ok || manager != managedByValue {
		return nil
	}
	deleteSecretVersionReq := &secretmanagerpb.DeleteSecretRequest{
		Name: name,
		Etag: gcpSecret.Etag,
	}
	err = c.smClient.DeleteSecret(ctx, deleteSecretVersionReq)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMDeleteSecret, err)
	return err
}

func parseError(err error) error {
	var gerr *apierror.APIError
	if errors.As(err, &gerr) && gerr.GRPCStatus().Code() == codes.NotFound {
		return esv1.NoSecretError{}
	}
	return err
}

func (c *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	secretName := fmt.Sprintf(globalSecretPath, c.store.ProjectID, ref.GetRemoteKey())
	gcpSecret, err := c.smClient.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: secretName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}

		return false, err
	}

	return gcpSecret != nil, nil
}

// PushSecret pushes a kubernetes secret key into gcp provider Secret.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, pushSecretData esv1.PushSecretData) error {
	var (
		payload []byte
		err     error
	)
	if pushSecretData.GetSecretKey() == "" {
		// Must convert secret values to string, otherwise data will be sent as base64 to Vault
		secretStringVal := make(map[string]string)
		for k, v := range secret.Data {
			secretStringVal[k] = string(v)
		}
		payload, err = utils.JSONMarshal(secretStringVal)
		if err != nil {
			return fmt.Errorf("failed to serialize secret content as JSON: %w", err)
		}
	} else {
		payload = secret.Data[pushSecretData.GetSecretKey()]
	}
	secretName := getName(c.store.ProjectID, c.store.Location, pushSecretData.GetRemoteKey())
	gcpSecret, err := c.smClient.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: secretName,
	})
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMGetSecret, err)

	if err != nil {
		if status.Code(err) != codes.NotFound {
			return err
		}

		var replication = &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{},
			},
		}

		if pushSecretData.GetMetadata() != nil {
			replica := &secretmanagerpb.Replication_UserManaged_Replica{}
			var err error
			meta, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](pushSecretData.GetMetadata())
			if err != nil {
				return fmt.Errorf("failed to parse PushSecret metadata: %w", err)
			}
			if meta != nil && meta.Spec.ReplicationLocation != "" {
				replica.Location = meta.Spec.ReplicationLocation
			}
			if meta != nil && meta.Spec.CMEKKeyName != "" {
				replica.CustomerManagedEncryption = &secretmanagerpb.CustomerManagedEncryption{
					KmsKeyName: meta.Spec.CMEKKeyName,
				}
			}
			replication = &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_UserManaged_{
					UserManaged: &secretmanagerpb.Replication_UserManaged{
						Replicas: []*secretmanagerpb.Replication_UserManaged_Replica{
							replica,
						},
					},
				},
			}
		}
		parent := getParentName(c.store.ProjectID, c.store.Location)
		scrt := &secretmanagerpb.Secret{
			Labels: map[string]string{
				managedByKey: managedByValue,
			},
		}
		// fix: cannot set Replication at all when using regional Secrets.
		if c.store.Location == "" {
			scrt.Replication = replication
		}

		topics, err := utils.FetchValueFromMetadata(topicsKey, pushSecretData.GetMetadata(), []any{})
		if err != nil {
			return fmt.Errorf("failed to fetch topics from metadata: %w", err)
		}

		for _, t := range topics {
			name, ok := t.(string)
			if !ok {
				return fmt.Errorf("invalid topic type")
			}

			scrt.Topics = append(scrt.Topics, &secretmanagerpb.Topic{
				Name: name,
			})
		}

		gcpSecret, err = c.smClient.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
			Parent:   parent,
			SecretId: pushSecretData.GetRemoteKey(),
			Secret:   scrt,
		})
		metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMCreateSecret, err)
		if err != nil {
			return err
		}
	}

	builder, err := newPushSecretBuilder(payload, pushSecretData)
	if err != nil {
		return err
	}

	annotations, labels, topics, err := builder.buildMetadata(gcpSecret.Annotations, gcpSecret.Labels, gcpSecret.Topics)
	if err != nil {
		return err
	}

	// Comparing with a pointer based slice doesn't work so we are converting
	// it to a string slice.
	existingTopics := make([]string, 0, len(gcpSecret.Topics))
	for _, t := range gcpSecret.Topics {
		existingTopics = append(existingTopics, t.Name)
	}

	if !maps.Equal(gcpSecret.Annotations, annotations) || !maps.Equal(gcpSecret.Labels, labels) || !slices.Equal(existingTopics, topics) {
		scrt := &secretmanagerpb.Secret{
			Name:        gcpSecret.Name,
			Etag:        gcpSecret.Etag,
			Labels:      labels,
			Annotations: annotations,
			Topics:      gcpSecret.Topics,
		}

		if c.store.Location != "" {
			scrt.Replication = &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_UserManaged_{
					UserManaged: &secretmanagerpb.Replication_UserManaged{
						Replicas: []*secretmanagerpb.Replication_UserManaged_Replica{
							{
								Location: c.store.Location,
							},
						},
					},
				},
			}
		}

		_, err = c.smClient.UpdateSecret(ctx, &secretmanagerpb.UpdateSecretRequest{
			Secret: scrt,
			UpdateMask: &field_mask.FieldMask{
				Paths: []string{"labels", "annotations"},
			},
		})
		metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMUpdateSecret, err)
		if err != nil {
			return err
		}
	}

	unlock, err := locks.TryLock(providerName, secretName)
	if err != nil {
		return err
	}
	defer unlock()

	gcpVersion, err := c.smClient.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/latest", secretName),
	})
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMAccessSecretVersion, err)

	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	if gcpVersion != nil && gcpVersion.Payload != nil && !builder.needUpdate(gcpVersion.Payload.Data) {
		return nil
	}

	var original []byte
	if gcpVersion != nil && gcpVersion.Payload != nil {
		original = gcpVersion.Payload.Data
	}

	data, err := builder.buildData(original)
	if err != nil {
		return err
	}

	parent := getName(c.store.ProjectID, c.store.Location, pushSecretData.GetRemoteKey())

	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: parent,
		Payload: &secretmanagerpb.SecretPayload{
			Data: data,
		},
	}

	_, err = c.smClient.AddSecretVersion(ctx, addSecretVersionReq)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMAddSecretVersion, err)
	return err
}

// GetAllSecrets syncs multiple secrets from gcp provider into a single Kubernetes Secret.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		return c.findByName(ctx, ref)
	}
	if len(ref.Tags) > 0 {
		return c.findByTags(ctx, ref)
	}

	return nil, errors.New(errUnexpectedFindOperator)
}

func (c *Client) findByName(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	// regex matcher
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	parent := getParentName(c.store.ProjectID, c.store.Location)
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: parent,
	}
	if ref.Path != nil {
		req.Filter = fmt.Sprintf("name:%s", *ref.Path)
	}
	// Call the API.
	it := c.smClient.ListSecrets(ctx, req)
	secretMap := make(map[string][]byte)
	var resp *secretmanagerpb.Secret
	defer metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMListSecrets, err)
	for {
		resp, err = it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}
		log.V(1).Info("gcp sm findByName found", "secrets", strconv.Itoa(it.PageInfo().Remaining()))
		key := c.trimName(resp.Name)
		// If we don't match we skip.
		// Also, if we have path, and it is not at the beguining we skip.
		// We have to check if path is at the beguining of the key because
		// there is no way to create a `name:%s*` (starts with) filter
		// At https://cloud.google.com/secret-manager/docs/filtering you can use `*`
		// but not like that it seems.
		if !matcher.MatchName(key) || (ref.Path != nil && !strings.HasPrefix(key, *ref.Path)) {
			continue
		}
		log.V(1).Info("gcp sm findByName matches", "name", resp.Name)
		secretMap[key], err = c.getData(ctx, key)
		if err != nil {
			return nil, err
		}
	}

	return utils.ConvertKeys(ref.ConversionStrategy, secretMap)
}

func (c *Client) getData(ctx context.Context, key string) ([]byte, error) {
	dataRef := esv1.ExternalSecretDataRemoteRef{
		Key: key,
	}
	data, err := c.GetSecret(ctx, dataRef)
	if err != nil {
		return []byte(""), err
	}
	return data, nil
}

func (c *Client) findByTags(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	var tagFilter string
	for k, v := range ref.Tags {
		tagFilter = fmt.Sprintf("%slabels.%s=%s ", tagFilter, k, v)
	}
	tagFilter = strings.TrimSuffix(tagFilter, " ")
	if ref.Path != nil {
		tagFilter = fmt.Sprintf("%s name:%s", tagFilter, *ref.Path)
	}
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", c.store.ProjectID),
	}
	log.V(1).Info("gcp sm findByTags", "tagFilter", tagFilter)
	req.Filter = tagFilter
	// Call the API.
	it := c.smClient.ListSecrets(ctx, req)
	var resp *secretmanagerpb.Secret
	var err error
	defer metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMListSecrets, err)
	secretMap := make(map[string][]byte)
	for {
		resp, err = it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}
		key := c.trimName(resp.Name)
		if ref.Path != nil && !strings.HasPrefix(key, *ref.Path) {
			continue
		}
		log.V(1).Info("gcp sm findByTags matches tags", "name", resp.Name)
		secretMap[key], err = c.getData(ctx, key)
		if err != nil {
			return nil, err
		}
	}

	return utils.ConvertKeys(ref.ConversionStrategy, secretMap)
}

func (c *Client) trimName(name string) string {
	projectIDNumuber := c.extractProjectIDNumber(name)
	prefix := getParentName(projectIDNumuber, c.store.Location)
	key := strings.TrimPrefix(name, fmt.Sprintf("%s/secrets/", prefix))
	return key
}

// extractProjectIDNumber grabs the project id from the full name returned by gcp api
// gcp api seems to always return the number and not the project name
// (and users would always use the name, while requests accept both).
func (c *Client) extractProjectIDNumber(secretFullName string) string {
	s := strings.Split(secretFullName, "/")
	ProjectIDNumuber := s[1]
	return ProjectIDNumuber
}

// GetSecret returns a single secret from the provider.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(c.smClient) || c.store.ProjectID == "" {
		return nil, errors.New(errUninitalizedGCPProvider)
	}

	if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
		return c.getSecretMetadata(ctx, ref)
	}

	version := ref.Version
	if version == "" {
		version = defaultVersion
	}
	name := fmt.Sprintf(globalSecretVersionsPath, c.store.ProjectID, ref.Key, version)
	if c.store.Location != "" {
		name = fmt.Sprintf(regionalSecretVersionsPath, c.store.ProjectID, c.store.Location, ref.Key, version)
	}
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}
	result, err := c.smClient.AccessSecretVersion(ctx, req)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMAccessSecretVersion, err)
	if err != nil && c.store.SecretVersionSelectionPolicy == esv1.SecretVersionSelectionPolicyLatestOrFetch &&
		ref.Version == "" && isErrSecretDestroyedOrDisabled(err) {
		// if the secret is destroyed or disabled, and we are configured to get the latest enabled secret,
		// we need to get the latest enabled secret
		// Extract the secret name from the version name for ListSecretVersions
		secretName := fmt.Sprintf(globalSecretPath, c.store.ProjectID, ref.Key)
		if c.store.Location != "" {
			secretName = fmt.Sprintf(regionalSecretPath, c.store.ProjectID, c.store.Location, ref.Key)
		}
		result, err = getLatestEnabledVersion(ctx, c.smClient, secretName)
	}
	if err != nil {
		err = parseError(err)
		return nil, fmt.Errorf(errClientGetSecretAccess, err)
	}

	if ref.Property == "" {
		if result.Payload.Data != nil {
			return result.Payload.Data, nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string for key: %s", ref.Key)
	}

	val := getDataByProperty(result.Payload.Data, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

func (c *Client) getSecretMetadata(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	name := getName(c.store.ProjectID, c.store.Location, ref.Key)
	secret, err := c.smClient.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: name,
	})

	err = parseError(err)
	if err != nil {
		return nil, fmt.Errorf(errClientGetSecretAccess, err)
	}

	const (
		annotations = "annotations"
		labels      = "labels"
	)

	if annotation := c.extractMetadataKey(ref.Property, annotations); annotation != "" {
		v, ok := secret.GetAnnotations()[annotation]
		if !ok {
			return nil, fmt.Errorf("annotation with key %s does not exist in secret %s", annotation, ref.Key)
		}

		return []byte(v), nil
	}

	if label := c.extractMetadataKey(ref.Property, labels); label != "" {
		v, ok := secret.GetLabels()[label]
		if !ok {
			return nil, fmt.Errorf("label with key %s does not exist in secret %s", label, ref.Key)
		}

		return []byte(v), nil
	}

	if ref.Property == annotations {
		j, err := json.Marshal(secret.GetAnnotations())
		if err != nil {
			return nil, fmt.Errorf("faild marshaling annotations into json: %w", err)
		}

		return j, nil
	}

	if ref.Property == labels {
		j, err := json.Marshal(secret.GetLabels())
		if err != nil {
			return nil, fmt.Errorf("faild marshaling labels into json: %w", err)
		}

		return j, nil
	}

	if ref.Property != "" {
		return nil, fmt.Errorf("invalid property %s: metadata property should start with either %s or %s", ref.Property, annotations, labels)
	}

	j, err := json.Marshal(map[string]map[string]string{
		"annotations": secret.GetAnnotations(),
		"labels":      secret.GetLabels(),
	})
	if err != nil {
		return nil, fmt.Errorf("faild marshaling metadata map into json: %w", err)
	}

	return j, nil
}

func (c *Client) extractMetadataKey(s, p string) string {
	prefix := p + "."
	if !strings.HasPrefix(s, prefix) {
		return ""
	}
	return strings.TrimPrefix(s, prefix)
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if c.smClient == nil || c.store.ProjectID == "" {
		return nil, errors.New(errUninitalizedGCPProvider)
	}

	data, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
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

func (c *Client) Close(_ context.Context) error {
	var err error
	if c.smClient != nil {
		err = c.smClient.Close()
	}
	if c.workloadIdentity != nil {
		err = c.workloadIdentity.Close()
	}
	useMu.Unlock()
	if err != nil {
		return fmt.Errorf(errClientClose, err)
	}
	return nil
}

func (c *Client) Validate() (esv1.ValidationResult, error) {
	if c.storeKind == esv1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1.ValidationResultUnknown, nil
	}
	return esv1.ValidationResultReady, nil
}

func getDataByProperty(data []byte, property string) gjson.Result {
	var payload string
	if data != nil {
		payload = string(data)
	}
	idx := strings.Index(property, ".")
	refProperty := property
	if idx > 0 {
		refProperty = strings.ReplaceAll(refProperty, ".", "\\.")
		val := gjson.Get(payload, refProperty)
		if val.Exists() {
			return val
		}
	}
	return gjson.Get(payload, property)
}

func getName(projectID, location, key string) string {
	if location != "" {
		return fmt.Sprintf(regionalSecretPath, projectID, location, key)
	}
	return fmt.Sprintf(globalSecretPath, projectID, key)
}

func getParentName(projectID, location string) string {
	if location != "" {
		return fmt.Sprintf(regionalSecretParentPath, projectID, location)
	}
	return fmt.Sprintf(globalSecretParentPath, projectID)
}

func isErrSecretDestroyedOrDisabled(err error) bool {
	st, _ := status.FromError(err)
	return st.Code() == codes.FailedPrecondition &&
		(strings.Contains(st.Message(), "DESTROYED state") || strings.Contains(st.Message(), "DISABLED state"))
}

func getLatestEnabledVersion(ctx context.Context, client GoogleSecretManagerClient, name string) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	iter := client.ListSecretVersions(ctx, &secretmanagerpb.ListSecretVersionsRequest{
		Parent: name,
		Filter: "state:ENABLED",
	})
	latestCreateTime := time.Unix(0, 0)
	latestVersion := &secretmanagerpb.SecretVersion{}
	for {
		version, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if version.CreateTime.AsTime().After(latestCreateTime) {
			latestCreateTime = version.CreateTime.AsTime()
			latestVersion = version
		}
	}
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/%s", name, latestVersion.Name),
	}
	return client.AccessSecretVersion(ctx, req)
}
