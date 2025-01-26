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

package externalsecret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"

	// Loading registered generators and providers.
	_ "github.com/external-secrets/external-secrets/pkg/generator/register"
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

// shouldSkipClusterSecretStore checks if we should skip an ExternalSecret because it uses a ClusterSecretStore
// and the controller does not manage ClusterSecretStores.
func (r *Reconciler) shouldSkipClusterSecretStore(es *esv1beta1.ExternalSecret) bool {
	if !r.ClusterSecretStoreEnabled {
		return es.Spec.SecretStoreRef.Kind == esv1beta1.ClusterSecretStoreKind
	}
	return false
}

// shouldSkipGenerator checks if we should skip a generator because its controller class does not match the controller.
func (r *Reconciler) shouldSkipGenerator(generatorDef *apiextensions.JSON) (bool, error) {
	// unmarshal the generator definition
	var genControllerClass genv1alpha1.ControllerClassResource
	err := json.Unmarshal(generatorDef.Raw, &genControllerClass)
	if err != nil {
		return false, err
	}

	// check if the generator is managed by this controller
	var controllerClass = genControllerClass.Spec.ControllerClass
	return controllerClass != "" && controllerClass != r.ControllerClass, nil
}

type dataSourcesResult struct {
	ssInfoMap           map[string]*StoreInfo
	cssInfoMap          map[string]*StoreInfo
	notExistSources     []string
	notReadySources     []string
	defaultStoreMissing bool
}

// getDataSourcesOrSkip returns info about the data sources for an ExternalSecret, or returns true if the ExternalSecret should be skipped
// because this controller does not manage one of the sources used by the ExternalSecret (controller class mismatch).
func (r *Reconciler) getDataSourcesOrSkip(ctx context.Context, es *esv1beta1.ExternalSecret) (skip bool, result *dataSourcesResult, err error) {
	//
	// WARNING: do not return errors which can't be fixed by a retry from this method!
	//          we do not update the status of the ES if we return an error here, so the error would be hidden from the user.
	//          instead, create a new return value to capture the error and process it later in the reconciliation loop.
	//

	// these variables store error information that we can use to update the status of the ExternalSecret
	// we return them rather than creating an error so we can use them to update the status of the ExternalSecret
	notExistSources := make([]string, 0)
	notReadySources := make([]string, 0)
	defaultStoreMissing := false

	// these maps use "store name" as the key because `kind` may not be set SecretStoreRef
	// so there would be two ways to reference the same store
	ssInfoMap := make(map[string]*StoreInfo)
	cssInfoMap := make(map[string]*StoreInfo)

	// this map contains "generator name" -> "true" for all generators used by the ExternalSecret
	// NOTE: a struct can be a map key, as long as it uses primitive types: https://go.dev/blog/maps#key-types
	foundGenerators := make(map[esv1beta1.GeneratorRef]bool)

	// get the default store `spec.secretStoreRef`
	var defaultStore *esv1beta1.SecretStoreRef
	if es.Spec.SecretStoreRef.Name != "" {
		defaultStore = &es.Spec.SecretStoreRef
	}

	// addListedKey adds a secret store to its corresponding map and adds the listedKeys to the store info.
	addListedKeys := func(storeRef *esv1beta1.SecretStoreRef, listedKeys []string) {
		// if the store is not set, we should use the default store
		if storeRef == nil || storeRef.Name == "" {
			if defaultStore == nil {
				// if the default store is not set, we can not continue
				defaultStoreMissing = true
				return
			}
			storeRef = defaultStore
		}

		// determine which map to use based on the kind of store
		// NOTE: if kind is not set, we assume it is a SecretStore
		var storeMap map[string]*StoreInfo
		switch storeRef.Kind {
		case esv1beta1.SecretStoreKind, "":
			storeMap = ssInfoMap
		case esv1beta1.ClusterSecretStoreKind:
			storeMap = cssInfoMap
		}

		// add the listed keys to the store, initializing the store if needed
		if _, ok := storeMap[storeRef.Name]; !ok {
			storeMap[storeRef.Name] = &StoreInfo{
				ListedKeys: make(map[string]bool),
				FoundKeys:  make(map[string]bool),
			}
		}
		for _, key := range listedKeys {
			storeMap[storeRef.Name].ListedKeys[key] = true
		}
	}

	// unpack stores from `spec.data[].sourceRef.secretStoreRef`
	for _, ref := range es.Spec.Data {
		// get the store reference or nil if not set
		var storeRef *esv1beta1.SecretStoreRef
		if ref.SourceRef != nil {
			storeRef = &ref.SourceRef.SecretStoreRef
		}

		// every data entry will specify a single key
		listedKeys := []string{ref.RemoteRef.Key}
		addListedKeys(storeRef, listedKeys)
	}

	// unpack stores and generators from `spec.dataFrom[].sourceRef`
	for _, ref := range es.Spec.DataFrom {
		// if this is a generator, mark it as found and skip to the next ref
		if ref.SourceRef != nil && ref.SourceRef.GeneratorRef != nil {
			foundGenerators[*ref.SourceRef.GeneratorRef] = true
			continue
		}

		// this is a secret store, get the store reference or nil if not set
		var storeRef *esv1beta1.SecretStoreRef
		if ref.SourceRef != nil && ref.SourceRef.SecretStoreRef != nil {
			storeRef = ref.SourceRef.SecretStoreRef
		}

		// any explicitly listed keys will be under `dataFrom[].extract`.
		// for sources that use `dataFrom[].find`, we don't know the keys until we get the data,
		// so we populate them later under foundKeys.
		listedKeys := make([]string, 0, 1)
		if ref.Extract != nil {
			listedKeys = append(listedKeys, ref.Extract.Key)
		}
		addListedKeys(storeRef, listedKeys)
	}

	// processes ClusterSecretStores
	for storeName, storeInfo := range cssInfoMap {
		// if this controller does not manage ClusterSecretStores, stop immediately
		// and return true to skip this ExternalSecret
		if !r.ClusterSecretStoreEnabled {
			return true, nil, nil
		}

		// get the ClusterSecretStore
		store := &esv1beta1.ClusterSecretStore{}
		err = r.Get(ctx, types.NamespacedName{Name: storeName}, store)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// skip non-existent stores, but add them to the list of missing sources
				notExistSources = append(notExistSources, fmt.Sprintf("ClusterSecretStore/%s", storeName))
				storeInfo.NotExists = true
				continue
			}
			return false, nil, err
		}

		// if the store is not managed by this controller, stop immediately
		// and return true to skip this ExternalSecret
		class := store.Spec.Controller
		if class != "" && class != r.ControllerClass {
			return true, nil, nil
		}

		// check if the SecretStore is ready
		if !isStoreReady(store.Status) {
			notReadySources = append(notReadySources, fmt.Sprintf("ClusterSecretStore/%s", storeName))
			storeInfo.NotReady = true
		}

		// if the store is an in-cluster Kubernetes provider, capture the namespace
		storeInfo.InClusterKubernetesNamespace = getInClusterKubernetesNamespace(store.Spec.Provider)
	}

	// processes SecretStores
	for storeName, storeInfo := range ssInfoMap {
		// get the SecretStore
		store := &esv1beta1.SecretStore{}
		err = r.Get(ctx, types.NamespacedName{Name: storeName, Namespace: es.Namespace}, store)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// skip non-existent stores, but add them to the list of missing sources
				notExistSources = append(notExistSources, fmt.Sprintf("SecretStore/%s", storeName))
				storeInfo.NotExists = true
				continue
			}
			return false, nil, err
		}

		// if the store is not managed by this controller, stop immediately
		// and return true to skip this ExternalSecret
		class := store.Spec.Controller
		if class != "" && class != r.ControllerClass {
			return true, nil, nil
		}

		// check if the SecretStore is ready
		if !isStoreReady(store.Status) {
			notReadySources = append(notReadySources, fmt.Sprintf("SecretStore/%s", storeName))
			storeInfo.NotReady = true
		}

		// if the store is an in-cluster Kubernetes provider, capture the namespace
		storeInfo.InClusterKubernetesNamespace = getInClusterKubernetesNamespace(store.Spec.Provider)
	}

	// processes Generators
	for generatorRef := range foundGenerators {
		// get the Generator
		var obj *apiextensions.JSON
		_, obj, err = resolvers.GeneratorRef(ctx, r.Client, r.Scheme, es.Namespace, &generatorRef)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// skip non-existent generators, but add them to the list of missing sources
				notExistSources = append(notExistSources, fmt.Sprintf("%s/%s", generatorRef.Kind, generatorRef.Name))
				continue
			}
			if errors.Is(err, resolvers.ErrUnableToGetGenerator) {
				// skip generators that we can't get (e.g. due to being invalid)
				// NOTE: if we return an error here, it won't set the status of the ES
				//       so the user won't know why the ES is not working
				continue
			}
			return false, nil, err
		}

		// if the generator is not managed by this controller, stop immediately
		// and return true to skip this ExternalSecret
		var skipGenerator bool
		skipGenerator, err = r.shouldSkipGenerator(obj)
		if err != nil {
			return false, nil, err
		}
		if skipGenerator {
			return true, nil, nil
		}
	}

	result = &dataSourcesResult{
		ssInfoMap:           ssInfoMap,
		cssInfoMap:          cssInfoMap,
		notExistSources:     notExistSources,
		notReadySources:     notReadySources,
		defaultStoreMissing: defaultStoreMissing,
	}
	return false, result, nil
}

// getProviderSecretData returns the provider's secret data with the provided ExternalSecret.
func (r *Reconciler) getProviderSecretData(ctx context.Context, externalSecret *esv1beta1.ExternalSecret, ssInfoMap, cssInfoMap map[string]*StoreInfo) (map[string][]byte, error) {
	// We MUST NOT create multiple instances of a provider client (mostly due to limitations with GCP)
	// secretstore.Manager ensures only a single instance of a client is created for a given store and closes them when done.
	mgr := secretstore.NewManager(r.Client, r.ControllerClass, r.EnableFloodGate)
	defer mgr.Close(ctx)

	providerData := make(map[string][]byte)
	for i, remoteRef := range externalSecret.Spec.DataFrom {
		var secretMap map[string][]byte
		var err error

		if remoteRef.Find != nil {
			// NOTE: for `dataFrom[].find`, we don't know the list of keys used keys until we get the data,
			//       so we provide the store info maps (ssInfoMap, cssInfoMap) so it can populate the foundKeys[]
			//       of the corresponding store.
			secretMap, err = r.handleFindAllSecrets(ctx, externalSecret, remoteRef, mgr, ssInfoMap, cssInfoMap)
			if err != nil {
				err = fmt.Errorf(errSpecDataFromFind, i, err)
			}
		} else if remoteRef.Extract != nil {
			// NOTE: for `dataFrom[].extract`, while we know the keys at this point, we don't know if they exist until we get the data,
			//       so we provide the store info maps (ssInfoMap, cssInfoMap) so it can populate the listedKeys[].notExists
			//       of the corresponding store.
			secretMap, err = r.handleExtractSecrets(ctx, externalSecret, remoteRef, mgr, ssInfoMap, cssInfoMap)
			if err != nil {
				err = fmt.Errorf(errSpecDataFromExtract, i, remoteRef.Extract.Key, err)
			}
		} else if remoteRef.SourceRef != nil && remoteRef.SourceRef.GeneratorRef != nil {
			secretMap, err = r.handleGenerateSecrets(ctx, externalSecret.Namespace, remoteRef)
			if err != nil {
				err = fmt.Errorf(errSpecDataFromGenerator, i, err)
			}
		}

		if errors.Is(err, esv1beta1.NoSecretErr) && externalSecret.Spec.Target.DeletionPolicy != esv1beta1.DeletionPolicyRetain {
			// silently skip missing provider secrets, if the deletion policy is not retain
			continue
		}
		if err != nil {
			return nil, err
		}

		providerData = utils.MergeByteMap(providerData, secretMap)
	}

	for i, secretRef := range externalSecret.Spec.Data {
		err := r.handleSecretData(ctx, externalSecret, secretRef, providerData, mgr, ssInfoMap, cssInfoMap)
		if errors.Is(err, esv1beta1.NoSecretErr) && externalSecret.Spec.Target.DeletionPolicy != esv1beta1.DeletionPolicyRetain {
			// silently skip missing provider secrets, if the deletion policy is not retain
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(errSpecData, i, secretRef.RemoteRef.Key, err)
		}
	}

	return providerData, nil
}

func (r *Reconciler) handleSecretData(
	ctx context.Context,
	externalSecret *esv1beta1.ExternalSecret,
	secretRef esv1beta1.ExternalSecretData,
	providerData map[string][]byte,
	cmgr *secretstore.Manager,
	ssInfoMap,
	cssInfoMap map[string]*StoreInfo,
) error {
	storeRef, client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, toStoreGenSourceRef(secretRef.SourceRef))
	if err != nil {
		return err
	}

	// choose which store info map to use based on the kind of store
	var storeInfo *StoreInfo
	switch storeRef.Kind {
	case esv1beta1.SecretStoreKind, "":
		storeInfo = ssInfoMap[storeRef.Name]
	case esv1beta1.ClusterSecretStoreKind:
		storeInfo = cssInfoMap[storeRef.Name]
	}

	// get a single secret from the store
	secretData, err := client.GetSecret(ctx, secretRef.RemoteRef)
	if err != nil {
		if errors.Is(err, esv1beta1.NoSecretErr) {
			// if the secret does not exist, mark it as false in the ListedKeys of the store info map
			storeInfo.ListedKeys[secretRef.RemoteRef.Key] = false
		}
		return err
	}

	// decode the secret if needed
	secretData, err = utils.Decode(secretRef.RemoteRef.DecodingStrategy, secretData)
	if err != nil {
		return fmt.Errorf(errDecode, secretRef.RemoteRef.DecodingStrategy, err)
	}

	// store the secret data
	providerData[secretRef.SecretKey] = secretData

	return nil
}

func (r *Reconciler) handleGenerateSecrets(ctx context.Context, namespace string, remoteRef esv1beta1.ExternalSecretDataFromRemoteRef) (map[string][]byte, error) {
	gen, obj, err := resolvers.GeneratorRef(ctx, r.Client, r.Scheme, namespace, remoteRef.SourceRef.GeneratorRef)
	if err != nil {
		return nil, err
	}

	// use the generator
	secretMap, err := gen.Generate(ctx, obj, r.Client, namespace)
	if err != nil {
		return nil, fmt.Errorf(errGenerate, err)
	}

	// rewrite the keys if needed
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errRewrite, err)
	}

	// validate the keys
	err = utils.ValidateKeys(secretMap)
	if err != nil {
		return nil, fmt.Errorf(errInvalidKeys, err)
	}

	return secretMap, err
}

func (r *Reconciler) handleExtractSecrets(
	ctx context.Context,
	externalSecret *esv1beta1.ExternalSecret,
	remoteRef esv1beta1.ExternalSecretDataFromRemoteRef,
	cmgr *secretstore.Manager,
	ssInfoMap,
	cssInfoMap map[string]*StoreInfo,
) (map[string][]byte, error) {
	storeRef, client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}

	// choose which store info map to use based on the kind of store
	var storeInfo *StoreInfo
	switch storeRef.Kind {
	case esv1beta1.SecretStoreKind, "":
		storeInfo = ssInfoMap[storeRef.Name]
	case esv1beta1.ClusterSecretStoreKind:
		storeInfo = cssInfoMap[storeRef.Name]
	}

	// get (potentially) multiple secrets from the store for a single key
	secretMap, err := client.GetSecretMap(ctx, *remoteRef.Extract)
	if err != nil {
		if errors.Is(err, esv1beta1.NoSecretErr) {
			// if the secret does not exist, mark it as false in the ListedKeys of the store info map
			storeInfo.ListedKeys[remoteRef.Extract.Key] = false
		}
		return nil, err
	}

	// rewrite the keys if needed
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errRewrite, err)
	}
	if len(remoteRef.Rewrite) == 0 {
		secretMap, err = utils.ConvertKeys(remoteRef.Extract.ConversionStrategy, secretMap)
		if err != nil {
			return nil, fmt.Errorf(errConvert, remoteRef.Extract.ConversionStrategy, err)
		}
	}

	// validate the keys
	err = utils.ValidateKeys(secretMap)
	if err != nil {
		return nil, fmt.Errorf(errInvalidKeys, err)
	}

	// decode the secrets if needed
	secretMap, err = utils.DecodeMap(remoteRef.Extract.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errDecode, remoteRef.Extract.DecodingStrategy, err)
	}

	return secretMap, err
}

func (r *Reconciler) handleFindAllSecrets(
	ctx context.Context,
	externalSecret *esv1beta1.ExternalSecret,
	remoteRef esv1beta1.ExternalSecretDataFromRemoteRef,
	cmgr *secretstore.Manager,
	ssInfoMap,
	cssInfoMap map[string]*StoreInfo,
) (map[string][]byte, error) {
	storeRef, client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}

	// choose which store info map to use based on the kind of store
	var storeInfo *StoreInfo
	switch storeRef.Kind {
	case esv1beta1.SecretStoreKind, "":
		storeInfo = ssInfoMap[storeRef.Name]
	case esv1beta1.ClusterSecretStoreKind:
		storeInfo = cssInfoMap[storeRef.Name]
	}

	// get all secrets from the store that match the selector
	secretMap, err := client.GetAllSecrets(ctx, *remoteRef.Find)
	if err != nil {
		return nil, err
	}

	// store the found keys in the store info map
	for key := range secretMap {
		storeInfo.FoundKeys[key] = true
	}

	// rewrite the keys if needed
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errRewrite, err)
	}
	if len(remoteRef.Rewrite) == 0 {
		secretMap, err = utils.ConvertKeys(remoteRef.Find.ConversionStrategy, secretMap)
		if err != nil {
			return nil, fmt.Errorf(errConvert, remoteRef.Find.ConversionStrategy, err)
		}
	}

	// validate the keys
	err = utils.ValidateKeys(secretMap)
	if err != nil {
		return nil, fmt.Errorf(errInvalidKeys, err)
	}

	// decode the secrets if needed
	secretMap, err = utils.DecodeMap(remoteRef.Find.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errDecode, remoteRef.Find.DecodingStrategy, err)
	}
	return secretMap, err
}

func toStoreGenSourceRef(ref *esv1beta1.StoreSourceRef) *esv1beta1.StoreGeneratorSourceRef {
	if ref == nil {
		return nil
	}
	return &esv1beta1.StoreGeneratorSourceRef{
		SecretStoreRef: &ref.SecretStoreRef,
	}
}

// isStoreReady checks if the SecretStore or ClusterSecretStore has a Ready=true condition.
func isStoreReady(status esv1beta1.SecretStoreStatus) bool {
	condition := secretstore.GetSecretStoreCondition(status, esv1beta1.SecretStoreReady)
	if condition == nil || condition.Status != v1.ConditionTrue {
		return false
	}
	return true
}

// getInClusterKubernetesNamespace returns the namespace of the in-cluster Kubernetes-type SecretStore, or an empty string if it is any other type.
func getInClusterKubernetesNamespace(provider *esv1beta1.SecretStoreProvider) string {
	// check if the provider is a Kubernetes provider
	if provider.Kubernetes == nil {
		return ""
	}

	// get the kubernetes server hostname from the provider
	// NOTE: users may specify any valid `rest.Config.Host` from `k8s.io/client-go`, so we need to handle both
	//       URLs (e.g. "https://kubernetes.default.svc") and naked hostnames (e.g. "kubernetes.default.svc")
	//       so we try to parse the URL and get the hostname, and if that fails, we assume it is a hostname
	serverHostname := provider.Kubernetes.Server.URL
	if u, err := url.Parse(serverHostname); err == nil {
		if h := u.Hostname(); h != "" {
			serverHostname = h
		}
	}

	// check if the server hostname points to the in-cluster API server
	// NOTE: 127.0.0.1 is used for tests, but is unlikely to be used in production
	switch serverHostname {
	case "", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster.local", "127.0.0.1":
		return provider.Kubernetes.RemoteNamespace
	}

	return ""
}

// StoreInfo contains information about a SecretStore or ClusterSecretStore.
// This directly translates to esv1beta1.ProviderSourceInfo, but is different because it's easier to build lists of keys using a map.
type StoreInfo struct {
	// NotReady is true if the SecretStore or ClusterSecretStore is not ready.
	NotReady bool

	// NotExists is true if the SecretStore or ClusterSecretStore does not exist.
	NotExists bool

	// InClusterKubernetesNamespace if this source is an in-cluster kubernetes provider, this is the namespace which the secrets are fetched from.
	InClusterKubernetesNamespace string

	// ListedKeys are the remote keys explicitly referenced in `data[].remoteRef.key` and `dataFrom[].extract.key`.
	// Note, if a key is not found in the provider, it will be marked as false.
	ListedKeys map[string]bool

	// FoundKeys are the remote keys found in the provider from `dataFrom[].extract.find`.
	FoundKeys map[string]bool
}
