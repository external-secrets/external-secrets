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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/tidwall/gjson"
	"google.golang.org/api/iterator"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/grpc/codes"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	CloudPlatformRole                         = "https://www.googleapis.com/auth/cloud-platform"
	defaultVersion                            = "latest"
	errGCPSMStore                             = "received invalid GCPSM SecretStore resource"
	errUnableGetCredentials                   = "unable to get credentials: %w"
	errClientClose                            = "unable to close SecretManager client: %w"
	errMissingStoreSpec                       = "invalid: missing store spec"
	errInvalidClusterStoreMissingSAKNamespace = "invalid ClusterSecretStore: missing GCP SecretAccessKey Namespace"
	errInvalidClusterStoreMissingSANamespace  = "invalid ClusterSecretStore: missing GCP Service Account Namespace"
	errFetchSAKSecret                         = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                             = "missing SecretAccessKey"
	errUnableProcessJSONCredentials           = "failed to process the provided JSON credentials: %w"
	errUnableCreateGCPSMClient                = "failed to create GCP secretmanager client: %w"
	errUninitalizedGCPProvider                = "provider GCP is not initialized"
	errClientGetSecretAccess                  = "unable to access Secret from SecretManager Client: %w"
	errJSONSecretUnmarshal                    = "unable to unmarshal secret: %w"

	errInvalidStore           = "invalid store"
	errInvalidStoreSpec       = "invalid store spec"
	errInvalidStoreProv       = "invalid store provider"
	errInvalidGCPProv         = "invalid gcp secrets manager provider"
	errInvalidAuthSecretRef   = "invalid auth secret ref: %w"
	errInvalidWISARef         = "invalid workload identity service account reference: %w"
	errUnexpectedFindOperator = "unexpected find operator"
)

type Client struct {
	smClient GoogleSecretManagerClient
	kube     kclient.Client
	store    *esv1beta1.GCPSMProvider

	// namespace of the external secret
	namespace        string
	workloadIdentity *workloadIdentity
}

type GoogleSecretManagerClient interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	ListSecrets(ctx context.Context, req *secretmanagerpb.ListSecretsRequest, opts ...gax.CallOption) *secretmanager.SecretIterator
	AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
	CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	Close() error
	GetSecret(ctx context.Context, req *secretmanagerpb.GetSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
}

var log = ctrl.Log.WithName("provider").WithName("gcp").WithName("secretsmanager")

// SetSecret pushes a kubernetes secret key into gcp provider Secret.
func (c *Client) SetSecret(ctx context.Context, payload []byte, remoteRef esv1beta1.PushRemoteRef) error {
	createSecretReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", c.store.ProjectID),
		SecretId: remoteRef.GetRemoteKey(),
		Secret: &secretmanagerpb.Secret{
			Labels: map[string]string{
				"managed-by": "external-secrets",
			},
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}

	var gcpSecret *secretmanagerpb.Secret
	var err error

	gcpSecret, err = c.smClient.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", c.store.ProjectID, remoteRef.GetRemoteKey()),
	})

	var gErr *apierror.APIError

	if err != nil && errors.As(err, &gErr) {
		if gErr.GRPCStatus().Code() == codes.NotFound {
			gcpSecret, err = c.smClient.CreateSecret(ctx, createSecretReq)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	manager, ok := gcpSecret.Labels["managed-by"]

	if !ok || manager != "external-secrets" {
		return fmt.Errorf("secret %v is not managed by external secrets", remoteRef.GetRemoteKey())
	}

	gcpVersion, err := c.smClient.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", c.store.ProjectID, remoteRef.GetRemoteKey()),
	})

	if errors.As(err, &gErr) {
		if err != nil && gErr.GRPCStatus().Code() != codes.NotFound {
			return err
		}
	} else if err != nil {
		return err
	}

	if gcpVersion != nil && gcpVersion.Payload != nil && bytes.Equal(payload, gcpVersion.Payload.Data) {
		return nil
	}

	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: fmt.Sprintf("projects/%s/secrets/%s", c.store.ProjectID, remoteRef.GetRemoteKey()),
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	}

	_, err = c.smClient.AddSecretVersion(ctx, addSecretVersionReq)

	if err != nil {
		return err
	}

	return nil
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
	for {
		resp, err := it.Next()
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
	secretMap := make(map[string][]byte)
	for {
		resp, err := it.Next()
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

	version := ref.Version
	if version == "" {
		version = defaultVersion
	}

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", c.store.ProjectID, ref.Key, version),
	}
	result, err := c.smClient.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(errClientGetSecretAccess, err)
	}

	if ref.Property == "" {
		if result.Payload.Data != nil {
			return result.Payload.Data, nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string for key: %s", ref.Key)
	}

	var payload string
	if result.Payload.Data != nil {
		payload = string(result.Payload.Data)
	}
	idx := strings.Index(ref.Property, ".")
	refProperty := ref.Property
	if idx > 0 {
		refProperty = strings.ReplaceAll(refProperty, ".", "\\.")
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

func (c *Client) Close(ctx context.Context) error {
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
	return esv1beta1.ValidationResultReady, nil
}
