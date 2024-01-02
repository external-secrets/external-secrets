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

package cerberus

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	cerberussdkapi "github.com/Nike-Inc/cerberus-go-client/v3/api"
	cerberussdk "github.com/Nike-Inc/cerberus-go-client/v3/cerberus"
	"github.com/aws/aws-sdk-go/aws"
	vaultapi "github.com/hashicorp/vault/api"
	v1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/cerberus/util"
)

const (
	versionIDKey = "versionId"
)

type cerberus struct {
	client AuthenticatedSecretReader
	sdb    *cerberussdkapi.SafeDepositBox
}

type AuthenticatedSecretReader interface {
	ReadSecret(string, map[string][]string) (*vaultapi.Secret, error)
	WriteSecret(string, map[string]interface{}) (*vaultapi.Secret, error)
	ListSecret(string) (*vaultapi.Secret, error)
	DeleteSecret(string) (*vaultapi.Secret, error)
	IsAuthenticated() bool
}

type cerberusClient struct {
	cerberussdk.Client
}

func (c *cerberusClient) ReadSecret(path string, data map[string][]string) (*vaultapi.Secret, error) {
	return c.Secret().ReadWithData(path, data)
}

func (c *cerberusClient) WriteSecret(path string, data map[string]interface{}) (*vaultapi.Secret, error) {
	return c.Secret().Write(path, data)
}

func (c *cerberusClient) ListSecret(path string) (*vaultapi.Secret, error) {
	return c.Secret().List(path)
}

func (c *cerberusClient) DeleteSecret(path string) (*vaultapi.Secret, error) {
	return c.Secret().Delete(path)
}

func (c *cerberusClient) IsAuthenticated() bool {
	return c.Authentication.IsAuthenticated()
}

var globalMutex = util.MutexMap{}

func (c *cerberus) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return c.readCerberusSecretProperties(ref)
}

func (c *cerberus) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	properties, err := c.GetSecretMap(ctx, ref)
	if err != nil {
		return nil, err
	}

	if ref.Property == "" {
		// workaround so that json.Marshal does not do base64 on []byte
		stringProps := map[string]string{}
		for k, v := range properties {
			stringProps[k] = string(v)
		}
		return json.Marshal(stringProps)
	}

	property, ok := properties[ref.Property]
	if !ok {
		return nil, fmt.Errorf("property %s does not exist in secret", ref.Property)
	}

	return property, nil
}

func (c *cerberus) PushSecret(_ context.Context, secret *v1.Secret, remoteRef esv1beta1.PushSecretData) error {
	if remoteRef.GetProperty() == "" {
		return fmt.Errorf("property must be set")
	}

	mu := globalMutex.GetLock(remoteRef.GetRemoteKey())
	mu.Lock()
	defer mu.Unlock()

	properties, err := c.readCerberusSecretProperties(esv1beta1.ExternalSecretDataRemoteRef{
		Key: remoteRef.GetRemoteKey(),
	})
	if err != nil {
		return err
	}

	properties[remoteRef.GetProperty()] = secret.Data[remoteRef.GetSecretKey()]

	return c.overwriteCerberusSecret(properties, remoteRef.GetRemoteKey())
}

func (c *cerberus) DeleteSecret(_ context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	if remoteRef.GetProperty() == "" {
		return fmt.Errorf("property must be set")
	}

	mu := globalMutex.GetLock(remoteRef.GetRemoteKey())
	mu.Lock()
	defer mu.Unlock()

	properties, err := c.readCerberusSecretProperties(esv1beta1.ExternalSecretDataRemoteRef{
		Key: remoteRef.GetRemoteKey(),
	})
	if err != nil {
		return err
	}

	delete(properties, remoteRef.GetProperty())

	if len(properties) > 0 {
		return c.overwriteCerberusSecret(properties, remoteRef.GetRemoteKey())
	}

	return c.deleteCerberusSecret(remoteRef.GetRemoteKey())
}

func (c *cerberus) Validate() (esv1beta1.ValidationResult, error) {
	if c.client == nil {
		return esv1beta1.ValidationResultError, nil
	}

	if c.client.IsAuthenticated() {
		return esv1beta1.ValidationResultReady, nil
	}

	return esv1beta1.ValidationResultError, nil
}

func (c *cerberus) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, fmt.Errorf("tags are not supported")
	}
	if ref.Path != nil && !strings.HasSuffix(*ref.Path, "/") {
		suffixed := fmt.Sprintf("%s/", *ref.Path)
		ref.Path = &suffixed
	}

	if ref.Path == nil {
		ref.Path = aws.String("/")
	}

	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}

	allSecretPaths, err := c.traverseAndFind(*ref.Path, func(name string) bool {
		return matcher.MatchName(name)
	})
	if err != nil {
		return nil, fmt.Errorf("error when listing secrets: %w", err)
	}

	results := make(map[string][]byte)
	for _, path := range allSecretPaths {
		data, err := c.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: path})

		if err != nil {
			return nil, err
		}
		results[strings.ReplaceAll(strings.TrimPrefix(path, *ref.Path), "/", "_")] = data
	}

	return results, nil
}

func (c *cerberus) Close(_ context.Context) error {
	return nil
}

func (c *cerberus) prependSDBPath(key string) string {
	return fmt.Sprintf("%s%s", c.sdb.Path, strings.TrimPrefix(key, "/"))
}

func (c *cerberus) traverseAndFind(startPath string, predicate func(string) bool) ([]string, error) {
	var collector []string

	list, err := c.client.ListSecret(c.prependSDBPath(startPath))
	if err != nil {
		return nil, err
	}

	if len(list.Data) == 0 {
		return collector, nil
	}

	secretsKeys, ok := list.Data["keys"]
	if !ok {
		return collector, nil
	}

	for _, secretKey := range secretsKeys.([]interface{}) {
		key := secretKey.(string)
		if strings.HasSuffix(key, "/") {
			subtreeCollector, err := c.traverseAndFind(fmt.Sprintf("%s%s", startPath, key), predicate)
			if err != nil {
				return nil, err
			}
			collector = append(collector, subtreeCollector...)
		} else if predicate(key) {
			collector = append(collector, fmt.Sprintf("%s%s", startPath, key))
		}
	}

	return collector, nil
}

func (c *cerberus) readCerberusSecretProperties(ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	properties, err := c.readAllPropsForPath(ref)
	if err != nil {
		return nil, fmt.Errorf("error when reading the secret: %w", err)
	}

	return properties, nil
}

func (c *cerberus) overwriteCerberusSecret(properties map[string][]byte, path string) error {
	fullPath := c.prependSDBPath(path)

	stringProperties := make(map[string]interface{})
	for k, v := range properties {
		stringProperties[k] = string(v)
	}

	shouldUpdate := false

	existing, err := c.client.ReadSecret(fullPath, nil)
	if err != nil {
		return err
	}

	if existing != nil && !reflect.DeepEqual(existing.Data, stringProperties) {
		shouldUpdate = true
	}

	if existing == nil || shouldUpdate {
		_, err = c.client.WriteSecret(fullPath, stringProperties)
	}

	return err
}

func (c *cerberus) deleteCerberusSecret(path string) error {
	fullPath := c.prependSDBPath(path)

	_, err := c.client.DeleteSecret(fullPath)

	return err
}

func (c *cerberus) readAllPropsForPath(ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data := make(map[string][]string)
	if ref.Version != "" {
		data[versionIDKey] = []string{ref.Version}
	}

	var secrets, err = c.client.ReadSecret(c.prependSDBPath(ref.Key), data)

	if err != nil {
		return nil, err
	}

	if secrets == nil {
		return map[string][]byte{}, nil
	}

	results := make(map[string][]byte)

	for k, v := range secrets.Data {
		results[k] = []byte(v.(string))
	}

	return results, nil
}
