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

package secretmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"

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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/util/locks"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	CloudPlatformRole               = "https://www.googleapis.com/auth/cloud-platform"
	defaultVersion                  = "latest"
	errGCPSMStore                   = "received invalid GCPSM SecretStore resource"
	errUnableGetCredentials         = "unable to get credentials: %w"
	errClientClose                  = "unable to close SecretManager client: %w"
	errMissingStoreSpec             = "invalid: missing store spec"
	errFetchSAKSecret               = "could not fetch SecretAccessKey secret: %w"
	errUnableProcessJSONCredentials = "failed to process the provided JSON credentials: %w"
	errUnableCreateGCPSMClient      = "failed to create GCP secretmanager client: %w"
	errUninitalizedGCPProvider      = "provider GCP is not initialized"
	errClientGetSecretAccess        = "unable to access Secret from SecretManager Client: %w"
	errJSONSecretUnmarshal          = "unable to unmarshal secret: %w"

	errInvalidStore           = "invalid store"
	errInvalidStoreSpec       = "invalid store spec"
	errInvalidStoreProv       = "invalid store provider"
	errInvalidGCPProv         = "invalid gcp secrets manager provider"
	errInvalidAuthSecretRef   = "invalid auth secret data: %w"
	errInvalidWISARef         = "invalid workload identity service account reference: %w"
	errUnexpectedFindOperator = "unexpected find operator"

	managedByKey   = "managed-by"
	managedByValue = "external-secrets"

	providerName = "GCPSecretManager"
)

type Client struct {
	smClient  GoogleSecretManagerClient
	kube      kclient.Client
	store     *esv1beta1.GCPSMProvider
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
}

var log = ctrl.Log.WithName("provider").WithName("gcp").WithName("secretsmanager")

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	gcpSecret, err := c.smClient.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", c.store.ProjectID, remoteRef.GetRemoteKey()),
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
		Name: fmt.Sprintf("projects/%s/secrets/%s", c.store.ProjectID, remoteRef.GetRemoteKey()),
		Etag: gcpSecret.Etag,
	}
	err = c.smClient.DeleteSecret(ctx, deleteSecretVersionReq)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMDeleteSecret, err)
	return err
}

func parseError(err error) error {
	var gerr *apierror.APIError
	if errors.As(err, &gerr) && gerr.GRPCStatus().Code() == codes.NotFound {
		return esv1beta1.NoSecretError{}
	}
	return err
}

func (c *Client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// PushSecret pushes a kubernetes secret key into gcp provider Secret.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, pushSecretData esv1beta1.PushSecretData) error {
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
	secretName := fmt.Sprintf("projects/%s/secrets/%s", c.store.ProjectID, pushSecretData.GetRemoteKey())
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

		if c.store.Location != "" {
			replication = &secretmanagerpb.Replication{
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

		gcpSecret, err = c.smClient.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
			Parent:   fmt.Sprintf("projects/%s", c.store.ProjectID),
			SecretId: pushSecretData.GetRemoteKey(),
			Secret: &secretmanagerpb.Secret{
				Labels: map[string]string{
					managedByKey: managedByValue,
				},
				Replication: replication,
			},
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

	annotations, labels, err := builder.buildMetadata(gcpSecret.Annotations, gcpSecret.Labels)
	if err != nil {
		return err
	}

	if !maps.Equal(gcpSecret.Annotations, annotations) || !maps.Equal(gcpSecret.Labels, labels) {
		scrt := &secretmanagerpb.Secret{
			Name:        gcpSecret.Name,
			Etag:        gcpSecret.Etag,
			Labels:      labels,
			Annotations: annotations,
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

	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: fmt.Sprintf("projects/%s/secrets/%s", c.store.ProjectID, pushSecretData.GetRemoteKey()),
		Payload: &secretmanagerpb.SecretPayload{
			Data: data,
		},
	}

	_, err = c.smClient.AddSecretVersion(ctx, addSecretVersionReq)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMAddSecretVersion, err)
	return err
}

// GetAllSecrets syncs multiple secrets from gcp provider into a single Kubernetes Secret.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		return c.findByName(ctx, ref)
	}
	if len(ref.Tags) > 0 {
		return c.findByTags(ctx, ref)
	}

	return nil, errors.New(errUnexpectedFindOperator)
}

func (c *Client) findByName(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// regex matcher
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", c.store.ProjectID),
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
	dataRef := esv1beta1.ExternalSecretDataRemoteRef{
		Key: key,
	}
	data, err := c.GetSecret(ctx, dataRef)
	if err != nil {
		return []byte(""), err
	}
	return data, nil
}

func (c *Client) findByTags(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
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
	key := strings.TrimPrefix(name, fmt.Sprintf("projects/%s/secrets/", projectIDNumuber))
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
func (c *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(c.smClient) || c.store.ProjectID == "" {
		return nil, fmt.Errorf(errUninitalizedGCPProvider)
	}

	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		return c.getSecretMetadata(ctx, ref)
	}

	version := ref.Version
	if version == "" {
		version = defaultVersion
	}

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", c.store.ProjectID, ref.Key, version),
	}
	result, err := c.smClient.AccessSecretVersion(ctx, req)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMAccessSecretVersion, err)
	err = parseError(err)
	if err != nil {
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

func (c *Client) getSecretMetadata(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.smClient.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", c.store.ProjectID, ref.Key),
	})

	err = parseError(err)
	if err != nil {
		return nil, fmt.Errorf(errClientGetSecretAccess, err)
	}

	const (
		annotations = "annotations"
		labels      = "labels"
	)

	extractMetadataKey := func(s string, p string) string {
		prefix := p + "."
		if !strings.HasPrefix(s, prefix) {
			return ""
		}
		return strings.TrimPrefix(s, prefix)
	}

	if annotation := extractMetadataKey(ref.Property, annotations); annotation != "" {
		v, ok := secret.GetAnnotations()[annotation]
		if !ok {
			return nil, fmt.Errorf("annotation with key %s does not exist in secret %s", annotation, ref.Key)
		}

		return []byte(v), nil
	}

	if label := extractMetadataKey(ref.Property, labels); label != "" {
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

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if c.smClient == nil || c.store.ProjectID == "" {
		return nil, fmt.Errorf(errUninitalizedGCPProvider)
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

func (c *Client) Validate() (esv1beta1.ValidationResult, error) {
	if c.storeKind == esv1beta1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1beta1.ValidationResultUnknown, nil
	}
	return esv1beta1.ValidationResultReady, nil
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
