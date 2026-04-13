/*
Copyright © The ESO Authors

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

// Package npws implements a Netwrix Password Secure provider for External Secrets.
package npws

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/npws/npwssdk"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var guidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	errNPWSStore                 = "received invalid NPWS SecretStore resource: %w"
	errNPWSStoreNilSpec          = "nil spec"
	errNPWSStoreNilSpecProvider  = "nil spec.provider"
	errNPWSStoreNilSpecNPWS      = "nil spec.provider.npws"
	errNPWSStoreMissingHost      = "missing: spec.provider.npws.host"
	errNPWSStoreMissingSecretRef = "missing: spec.provider.npws.auth.secretRef"
	errNPWSStoreMissingAPIKeyRef = "missing: spec.provider.npws.auth.secretRef.apiKey.name"
)

// npwsAPI abstracts all NPWS SDK methods used by the Provider.
type npwsAPI interface {
	GetContainer(ctx context.Context, id string) (*npwssdk.PsrContainer, error)
	GetContainerByName(ctx context.Context, name string) (*npwssdk.PsrContainer, error)
	GetContainerList(ctx context.Context, containerType npwssdk.ContainerType, filter *npwssdk.PsrContainerListFilter) ([]npwssdk.PsrContainer, error)
	UpdateContainer(ctx context.Context, container *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error)
	AddContainer(ctx context.Context, container *npwssdk.PsrContainer, parentOrgUnitID string) (*npwssdk.PsrContainer, error)
	DeleteContainer(ctx context.Context, container *npwssdk.PsrContainer) error
	DecryptContainerItem(ctx context.Context, item *npwssdk.PsrContainerItem, reason string) (string, error)
	GetCurrentUserID() string
	Logout(ctx context.Context) error
}

// psrAPIAdapter wraps *npwssdk.PsrAPI to satisfy the npwsAPI interface.
type psrAPIAdapter struct {
	inner *npwssdk.PsrAPI
}

func (a *psrAPIAdapter) GetContainer(ctx context.Context, id string) (*npwssdk.PsrContainer, error) {
	return a.inner.Containers.GetContainer(ctx, id)
}

func (a *psrAPIAdapter) GetContainerByName(ctx context.Context, name string) (*npwssdk.PsrContainer, error) {
	return a.inner.Containers.GetContainerByName(ctx, name)
}

func (a *psrAPIAdapter) GetContainerList(ctx context.Context, containerType npwssdk.ContainerType, filter *npwssdk.PsrContainerListFilter) ([]npwssdk.PsrContainer, error) {
	return a.inner.Containers.GetContainerList(ctx, containerType, filter)
}

func (a *psrAPIAdapter) UpdateContainer(ctx context.Context, container *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error) {
	return a.inner.Containers.UpdateContainer(ctx, container)
}

func (a *psrAPIAdapter) AddContainer(ctx context.Context, container *npwssdk.PsrContainer, parentOrgUnitID string) (*npwssdk.PsrContainer, error) {
	return a.inner.Containers.AddContainer(ctx, container, parentOrgUnitID)
}

func (a *psrAPIAdapter) DeleteContainer(ctx context.Context, container *npwssdk.PsrContainer) error {
	return a.inner.Containers.DeleteContainer(ctx, container)
}

func (a *psrAPIAdapter) DecryptContainerItem(ctx context.Context, item *npwssdk.PsrContainerItem, reason string) (string, error) {
	return a.inner.Containers.DecryptContainerItem(ctx, item, reason)
}

func (a *psrAPIAdapter) GetCurrentUserID() string {
	return a.inner.UserKeys.GetCurrentUserID()
}

func (a *psrAPIAdapter) Logout(ctx context.Context) error {
	return a.inner.Auth.Logout(ctx)
}

// Provider implements the Netwrix Password Secure provider.
// It satisfies both esv1.Provider and esv1.SecretsClient.
type Provider struct {
	api                      npwsAPI
	deletionPolicyWholeEntry bool
}

// Compile-time interface checks.
var (
	_ esv1.Provider      = &Provider{}
	_ esv1.SecretsClient = &Provider{}
)

// NewProvider returns a new NPWS Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the SecretStoreProvider spec that identifies this provider.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		NPWS: &esv1.NPWSProvider{},
	}
}

// MaintenanceStatus returns the maintenance status for this provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}

// Capabilities declares that this provider supports both read and write operations.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient constructs a ready-to-use SecretsClient from the given SecretStore.
// It reads the API key from the referenced Kubernetes Secret and authenticates with NPWS.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	cfg := store.GetSpec().Provider.NPWS

	apiKey, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &cfg.Auth.SecretRef.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve NPWS API key: %w", err)
	}

	api := npwssdk.NewPsrAPI(cfg.Host)
	api.SetClientType("ExternalSecretsOperator")
	if err := api.Auth.LoginWithAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to authenticate with NPWS: %w", err)
	}

	return &Provider{api: &psrAPIAdapter{inner: api}, deletionPolicyWholeEntry: cfg.DeletionPolicyWholeEntry}, nil
}

// ValidateStore checks that the SecretStore configuration is structurally valid.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if err := validateStore(store); err != nil {
		return nil, fmt.Errorf(errNPWSStore, err)
	}
	return nil, nil
}

func validateStore(store esv1.GenericStore) error {
	spec := store.GetSpec()
	if spec == nil {
		return fmt.Errorf(errNPWSStoreNilSpec)
	}
	if spec.Provider == nil {
		return fmt.Errorf(errNPWSStoreNilSpecProvider)
	}
	cfg := spec.Provider.NPWS
	if cfg == nil {
		return fmt.Errorf(errNPWSStoreNilSpecNPWS)
	}
	if cfg.Host == "" {
		return fmt.Errorf(errNPWSStoreMissingHost)
	}
	if cfg.Auth.SecretRef == nil {
		return fmt.Errorf(errNPWSStoreMissingSecretRef)
	}
	if cfg.Auth.SecretRef.APIKey.Name == "" {
		return fmt.Errorf(errNPWSStoreMissingAPIKeyRef)
	}
	return nil
}

// isGUID returns true if the string is a valid GUID format.
func isGUID(s string) bool {
	return guidRegex.MatchString(s)
}

// resolveContainer resolves a container by ID (if GUID) or by name.
func (p *Provider) resolveContainer(ctx context.Context, key string) (*npwssdk.PsrContainer, error) {
	if isGUID(key) {
		return p.api.GetContainer(ctx, key)
	}
	return p.api.GetContainerByName(ctx, key)
}

// GetSecret returns the value of a single secret.
// ref.Key = Container ID or name, ref.Property = Container item name (optional).
// If ref.Property is empty, returns the first password item's value.
func (p *Provider) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	container, err := p.resolveContainer(ctx, ref.Key)
	if err != nil {
		return nil, fmt.Errorf("GetSecret: %w", err)
	}
	if container == nil {
		return nil, fmt.Errorf("GetSecret: container %q not found", ref.Key)
	}

	item := findItem(container, ref.Property)
	if item == nil {
		return nil, esv1.NoSecretError{}
	}

	if item.IsEncrypted() {
		plaintext, err := p.api.DecryptContainerItem(ctx, item, "ESO GetSecret")
		if err != nil {
			return nil, fmt.Errorf("GetSecret: decrypt: %w", err)
		}
		item.PlainTextValue = plaintext
	}

	return []byte(item.GetValue()), nil
}

// GetSecretMap returns all fields of a container as key/value pairs.
// ref.Key = Container ID or name.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	container, err := p.resolveContainer(ctx, ref.Key)
	if err != nil {
		return nil, fmt.Errorf("GetSecretMap: %w", err)
	}
	if container == nil {
		return nil, fmt.Errorf("GetSecretMap: container %q not found", ref.Key)
	}

	result := make(map[string][]byte)
	for i := range container.Items {
		item := &container.Items[i]
		if item.ContainerItemType == npwssdk.ContainerItemHeader {
			continue
		}

		if item.IsEncrypted() {
			plaintext, err := p.api.DecryptContainerItem(ctx, item, "ESO GetSecretMap")
			if err != nil {
				return nil, fmt.Errorf("GetSecretMap: decrypt %q: %w", item.Name, err)
			}
			item.PlainTextValue = plaintext
		}

		result[item.Name] = []byte(item.GetValue())
	}

	return result, nil
}

// GetAllSecrets returns secrets matching the find criteria.
// ref.Path is used as container type filter, ref.Name.RegExp filters by container name.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	filter := &npwssdk.PsrContainerListFilter{
		ContainerType: npwssdk.ContainerTypePassword,
	}

	containers, err := p.api.GetContainerList(ctx, npwssdk.ContainerTypePassword, filter)
	if err != nil {
		return nil, fmt.Errorf("GetAllSecrets: %w", err)
	}

	var nameRegex *regexp.Regexp
	if ref.Name != nil && ref.Name.RegExp != "" {
		nameRegex, err = regexp.Compile(ref.Name.RegExp)
		if err != nil {
			return nil, fmt.Errorf("GetAllSecrets: invalid regex: %w", err)
		}
	}

	result := make(map[string][]byte)
	for _, c := range containers {
		if nameRegex != nil && !nameRegex.MatchString(c.GetDisplayName()) {
			continue
		}

		// Get the first password item from each container
		pwItem := findFirstPasswordItem(&c)
		if pwItem == nil {
			continue
		}

		if pwItem.IsEncrypted() {
			plaintext, err := p.api.DecryptContainerItem(ctx, pwItem, "ESO GetAllSecrets")
			if err != nil {
				continue // skip containers we can't decrypt
			}
			pwItem.PlainTextValue = plaintext
		}

		result[c.GetDisplayName()] = []byte(pwItem.GetValue())
	}

	return result, nil
}

// PushSecret writes a secret to Netwrix Password Secure.
// If the container does not exist and remoteKey is not a GUID, a new container is created.
func (p *Provider) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	remoteKey := data.GetRemoteKey()
	property := data.GetProperty()
	value := string(secret.Data[data.GetSecretKey()])

	container, err := p.resolveContainer(ctx, remoteKey)
	if err != nil {
		return fmt.Errorf("PushSecret: %w", err)
	}
	if container == nil {
		// Container not found — create new one if remoteKey is a name (not GUID)
		if isGUID(remoteKey) {
			return fmt.Errorf("PushSecret: container with ID %q not found", remoteKey)
		}
		if isGUID(property) {
			return fmt.Errorf("PushSecret: cannot create container with GUID property %q", property)
		}
		return p.createContainer(ctx, remoteKey, property, value)
	}

	// Container exists — find or create the item
	item := findItem(container, property)
	if item == nil {
		if isGUID(property) {
			return fmt.Errorf("PushSecret: item with ID %q not found in container %q", property, remoteKey)
		}
		name := property
		if name == "" {
			name = "Password"
		}
		container.Items = append(container.Items, npwssdk.PsrContainerItem{
			Name:              name,
			ContainerItemType: npwssdk.ContainerItemPassword,
		})
		item = &container.Items[len(container.Items)-1]
	} else {
		// Item exists — check if the value actually changed
		var oldValue string
		if item.IsEncrypted() {
			decrypted, err := p.api.DecryptContainerItem(ctx, item, "ESO PushSecret compare")
			if err != nil {
				return fmt.Errorf("PushSecret: decrypt for compare: %w", err)
			}
			oldValue = decrypted
		} else {
			oldValue = item.GetValue()
		}
		if oldValue == value {
			return nil // no change — skip update
		}
	}

	oldDataName := container.DataName()
	if err := item.SetValue(value); err != nil {
		return fmt.Errorf("PushSecret: %w", err)
	}
	if err := checkDataNameUnchanged(oldDataName, container); err != nil {
		return fmt.Errorf("PushSecret: %w", err)
	}

	if _, err := p.api.UpdateContainer(ctx, container); err != nil {
		return fmt.Errorf("PushSecret: update: %w", err)
	}

	return nil
}

// createContainer creates a new NPWS container with a Name field and a Password field.
func (p *Provider) createContainer(ctx context.Context, containerName, propertyName, value string) error {
	if propertyName == "" {
		propertyName = "Password"
	}

	nameItem := npwssdk.PsrContainerItem{
		Name:              "Name",
		ContainerItemType: npwssdk.ContainerItemText,
	}
	if err := nameItem.SetValue(containerName); err != nil {
		return fmt.Errorf("PushSecret: create: %w", err)
	}

	pwItem := npwssdk.PsrContainerItem{
		Name:              propertyName,
		ContainerItemType: npwssdk.ContainerItemPassword,
	}
	if err := pwItem.SetValue(value); err != nil {
		return fmt.Errorf("PushSecret: create: %w", err)
	}

	container := &npwssdk.PsrContainer{
		Type:          "MtoContainer",
		ContainerType: npwssdk.ContainerTypePassword,
		Items:         []npwssdk.PsrContainerItem{nameItem, pwItem},
	}

	orgUnitID := p.api.GetCurrentUserID()
	if _, err := p.api.AddContainer(ctx, container, orgUnitID); err != nil {
		return fmt.Errorf("PushSecret: create: %w", err)
	}

	return nil
}

// DeleteSecret removes a secret from Netwrix Password Secure.
// If deleteWholeEntry is false (default) and a property is specified, only that field is removed from the entry.
// If deleteWholeEntry is true or no property is specified, the entire entry is deleted.
func (p *Provider) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	container, err := p.resolveContainer(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return fmt.Errorf("DeleteSecret: %w", err)
	}
	if container == nil {
		return nil // already gone — nothing to do
	}

	property := remoteRef.GetProperty()
	if !p.deletionPolicyWholeEntry && property != "" {
		item := findItem(container, property)
		if item == nil {
			return nil // already gone — nothing to do
		}
		oldDataName := container.DataName()
		container.Items = removeItem(container.Items, item)
		if len(container.Items) == 0 || onlyOneDataNameCandidateLeft(container) {
			return p.api.DeleteContainer(ctx, container)
		}
		if err := checkDataNameUnchanged(oldDataName, container); err != nil {
			return fmt.Errorf("DeleteSecret: %w", err)
		}
		if _, err := p.api.UpdateContainer(ctx, container); err != nil {
			return fmt.Errorf("DeleteSecret: update after field removal: %w", err)
		}
		return nil
	}

	return p.api.DeleteContainer(ctx, container)
}

// checkDataNameUnchanged verifies that a container modification does not change the DataName.
// This prevents accidental loss of the container's display name in NPWS.
func checkDataNameUnchanged(oldName string, container *npwssdk.PsrContainer) error {
	newName := container.DataName()
	if newName != oldName {
		return fmt.Errorf("operation would change the entry name from %q to %q — aborting to prevent name mismatch", oldName, newName)
	}
	return nil
}

// onlyOneDataNameCandidateLeft returns true if the container has exactly one item
// and that item is a DataName candidate.
func onlyOneDataNameCandidateLeft(container *npwssdk.PsrContainer) bool {
	return len(container.Items) == 1 && npwssdk.IsDataNameCandidate(&container.Items[0])
}

// removeItem removes the given item pointer from the slice.
func removeItem(items []npwssdk.PsrContainerItem, target *npwssdk.PsrContainerItem) []npwssdk.PsrContainerItem {
	result := make([]npwssdk.PsrContainerItem, 0, len(items)-1)
	for i := range items {
		if &items[i] != target {
			result = append(result, items[i])
		}
	}
	return result
}

// SecretExists checks whether a container with the given key exists in NPWS.
func (p *Provider) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	container, err := p.resolveContainer(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return false, nil
	}
	return container != nil, nil
}

// Validate checks whether the client is properly configured and can reach NPWS.
func (p *Provider) Validate() (esv1.ValidationResult, error) {
	if p.api == nil {
		return esv1.ValidationResultError, fmt.Errorf("NPWS provider: not connected")
	}
	return esv1.ValidationResultReady, nil
}

// Close cleans up resources and logs out from NPWS.
func (p *Provider) Close(ctx context.Context) error {
	if p.api != nil {
		return p.api.Logout(ctx)
	}
	return nil
}

// typePrefix is the prefix used to select items by type instead of name.
const typePrefix = "type:"

// typeMap maps type names to ContainerItemType values.
var typeMap = map[string]npwssdk.ContainerItemType{
	"text":         npwssdk.ContainerItemText,
	"password":     npwssdk.ContainerItemPassword,
	"date":         npwssdk.ContainerItemDate,
	"check":        npwssdk.ContainerItemCheck,
	"url":          npwssdk.ContainerItemURL,
	"email":        npwssdk.ContainerItemEmail,
	"phone":        npwssdk.ContainerItemPhone,
	"list":         npwssdk.ContainerItemList,
	"memo":         npwssdk.ContainerItemMemo,
	"passwordmemo": npwssdk.ContainerItemPasswordMemo,
	"int":          npwssdk.ContainerItemInt,
	"decimal":      npwssdk.ContainerItemDecimal,
	"username":     npwssdk.ContainerItemUserName,
	"ip":           npwssdk.ContainerItemIP,
	"hostname":     npwssdk.ContainerItemHostName,
	"otp":          npwssdk.ContainerItemOtp,
}

// findItem finds a container item by ID, name, type prefix, or the first password item if property is empty.
// Resolution order: empty → first password, GUID → by item ID, "type:..." → by type, otherwise → by name.
func findItem(container *npwssdk.PsrContainer, property string) *npwssdk.PsrContainerItem {
	if property == "" {
		return findFirstPasswordItem(container)
	}
	if isGUID(property) {
		return findItemByID(container, property)
	}
	if strings.HasPrefix(strings.ToLower(property), typePrefix) {
		typeName := strings.ToLower(strings.TrimPrefix(property, typePrefix))
		return findFirstItemByType(container, typeName)
	}
	return findItemByName(container, property)
}

// findItemByID returns the item with the given ID, or nil.
func findItemByID(container *npwssdk.PsrContainer, id string) *npwssdk.PsrContainerItem {
	for i := range container.Items {
		if container.Items[i].ID == id {
			return &container.Items[i]
		}
	}
	return nil
}

// findItemByName returns the item with the given name, or nil.
func findItemByName(container *npwssdk.PsrContainer, name string) *npwssdk.PsrContainerItem {
	for i := range container.Items {
		if container.Items[i].Name == name {
			return &container.Items[i]
		}
	}
	return nil
}

// findFirstItemByType returns the first item matching the given type name, or nil.
func findFirstItemByType(container *npwssdk.PsrContainer, typeName string) *npwssdk.PsrContainerItem {
	itemType, ok := typeMap[typeName]
	if !ok {
		return nil
	}
	for i := range container.Items {
		if container.Items[i].ContainerItemType == itemType {
			return &container.Items[i]
		}
	}
	return nil
}

// findFirstPasswordItem returns the first encrypted item in the container, or nil.
// Priority: Password > PasswordMemo > OTP.
func findFirstPasswordItem(container *npwssdk.PsrContainer) *npwssdk.PsrContainerItem {
	priority := []npwssdk.ContainerItemType{
		npwssdk.ContainerItemPassword,
		npwssdk.ContainerItemPasswordMemo,
		npwssdk.ContainerItemOtp,
	}
	for _, t := range priority {
		for i := range container.Items {
			if container.Items[i].ContainerItemType == t {
				return &container.Items[i]
			}
		}
	}
	return nil
}
