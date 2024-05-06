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

	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	// Loading registered providers.
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/utils"

	// Loading registered generators.
	_ "github.com/external-secrets/external-secrets/pkg/generator/register"
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

// getProviderSecretData returns the provider's secret data with the provided ExternalSecret.
func (r *Reconciler) getProviderSecretData(ctx context.Context, externalSecret *esv1beta1.ExternalSecret) (map[string][]byte, error) {
	// We MUST NOT create multiple instances of a provider client (mostly due to limitations with GCP)
	// Clientmanager keeps track of the client instances
	// that are created during the fetching process and closes clients
	// if needed.
	mgr := secretstore.NewManager(r.Client, r.ControllerClass, r.EnableFloodGate)
	defer mgr.Close(ctx)

	providerData := make(map[string][]byte)
	for i, remoteRef := range externalSecret.Spec.DataFrom {
		var secretMap map[string][]byte
		var err error

		if remoteRef.Find != nil {
			secretMap, err = r.handleFindAllSecrets(ctx, externalSecret, remoteRef, mgr, i)
		} else if remoteRef.Extract != nil {
			secretMap, err = r.handleExtractSecrets(ctx, externalSecret, remoteRef, mgr, i)
		} else if remoteRef.SourceRef != nil && remoteRef.SourceRef.GeneratorRef != nil {
			secretMap, err = r.handleGenerateSecrets(ctx, externalSecret.Namespace, remoteRef, i)
		}
		if errors.Is(err, esv1beta1.NoSecretErr) && externalSecret.Spec.Target.DeletionPolicy != esv1beta1.DeletionPolicyRetain {
			r.recorder.Event(
				externalSecret,
				v1.EventTypeNormal,
				esv1beta1.ReasonDeleted,
				fmt.Sprintf("secret does not exist at provider using .dataFrom[%d]", i),
			)
			continue
		}
		if err != nil {
			return nil, err
		}
		providerData = utils.MergeByteMap(providerData, secretMap)
	}

	for i, secretRef := range externalSecret.Spec.Data {
		err := r.handleSecretData(ctx, i, *externalSecret, secretRef, providerData, mgr)
		if errors.Is(err, esv1beta1.NoSecretErr) && externalSecret.Spec.Target.DeletionPolicy != esv1beta1.DeletionPolicyRetain {
			r.recorder.Event(externalSecret, v1.EventTypeNormal, esv1beta1.ReasonDeleted, fmt.Sprintf("secret does not exist at provider using .data[%d] key=%s", i, secretRef.RemoteRef.Key))
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error retrieving secret at .data[%d], key: %s, err: %w", i, secretRef.RemoteRef.Key, err)
		}
	}

	return providerData, nil
}

func (r *Reconciler) handleSecretData(ctx context.Context, i int, externalSecret esv1beta1.ExternalSecret, secretRef esv1beta1.ExternalSecretData, providerData map[string][]byte, cmgr *secretstore.Manager) error {
	client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, toStoreGenSourceRef(secretRef.SourceRef))
	if err != nil {
		return err
	}
	secretData, err := client.GetSecret(ctx, secretRef.RemoteRef)
	if err != nil {
		return err
	}
	secretData, err = utils.Decode(secretRef.RemoteRef.DecodingStrategy, secretData)
	if err != nil {
		return fmt.Errorf(errDecode, "spec.data", i, err)
	}
	providerData[secretRef.SecretKey] = secretData
	return nil
}

func toStoreGenSourceRef(ref *esv1beta1.StoreSourceRef) *esv1beta1.StoreGeneratorSourceRef {
	if ref == nil {
		return nil
	}
	return &esv1beta1.StoreGeneratorSourceRef{
		SecretStoreRef: &ref.SecretStoreRef,
	}
}

func (r *Reconciler) handleGenerateSecrets(ctx context.Context, namespace string, remoteRef esv1beta1.ExternalSecretDataFromRemoteRef, i int) (map[string][]byte, error) {
	genDef, err := r.getGeneratorDefinition(ctx, namespace, remoteRef.SourceRef.GeneratorRef)
	if err != nil {
		return nil, err
	}
	gen, err := genv1alpha1.GetGenerator(genDef)
	if err != nil {
		return nil, err
	}
	secretMap, err := gen.Generate(ctx, genDef, r.Client, namespace)
	if err != nil {
		return nil, fmt.Errorf(errGenerate, i, err)
	}
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errRewrite, i, err)
	}
	if !utils.ValidateKeys(secretMap) {
		return nil, fmt.Errorf(errInvalidKeys, "generator", i)
	}
	return secretMap, err
}

// getGeneratorDefinition returns the generator JSON for a given sourceRef
// when it uses a generatorRef it fetches the resource and returns the JSON.
func (r *Reconciler) getGeneratorDefinition(ctx context.Context, namespace string, generatorRef *esv1beta1.GeneratorRef) (*apiextensions.JSON, error) {
	// client-go dynamic client needs a GVR to fetch the resource
	// But we only have the GVK in our generatorRef.
	//
	// TODO: there is no need to discover the GroupVersionResource
	//       this should be cached.
	c := discovery.NewDiscoveryClientForConfigOrDie(r.RestConfig)
	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return nil, err
	}

	gv, err := schema.ParseGroupVersion(generatorRef.APIVersion)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := mapper.RESTMapping(schema.GroupKind{
		Group: gv.Group,
		Kind:  generatorRef.Kind,
	})
	if err != nil {
		return nil, err
	}
	d, err := dynamic.NewForConfig(r.RestConfig)
	if err != nil {
		return nil, err
	}
	res, err := d.Resource(mapping.Resource).
		Namespace(namespace).
		Get(ctx, generatorRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	jsonRes, err := res.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return &apiextensions.JSON{Raw: jsonRes}, nil
}

func (r *Reconciler) handleExtractSecrets(ctx context.Context, externalSecret *esv1beta1.ExternalSecret, remoteRef esv1beta1.ExternalSecretDataFromRemoteRef, cmgr *secretstore.Manager, i int) (map[string][]byte, error) {
	client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}
	secretMap, err := client.GetSecretMap(ctx, *remoteRef.Extract)
	if err != nil {
		return nil, err
	}
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errRewrite, i, err)
	}
	if len(remoteRef.Rewrite) == 0 {
		secretMap, err = utils.ConvertKeys(remoteRef.Extract.ConversionStrategy, secretMap)
		if err != nil {
			return nil, fmt.Errorf(errConvert, err)
		}
	}
	if !utils.ValidateKeys(secretMap) {
		return nil, fmt.Errorf(errInvalidKeys, "extract", i)
	}
	secretMap, err = utils.DecodeMap(remoteRef.Extract.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errDecode, "spec.dataFrom", i, err)
	}
	return secretMap, err
}

func (r *Reconciler) handleFindAllSecrets(ctx context.Context, externalSecret *esv1beta1.ExternalSecret, remoteRef esv1beta1.ExternalSecretDataFromRemoteRef, cmgr *secretstore.Manager, i int) (map[string][]byte, error) {
	client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}
	secretMap, err := client.GetAllSecrets(ctx, *remoteRef.Find)
	if err != nil {
		return nil, err
	}
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errRewrite, i, err)
	}
	if len(remoteRef.Rewrite) == 0 {
		// ConversionStrategy is deprecated. Use RewriteMap instead.
		r.recorder.Event(externalSecret, v1.EventTypeWarning, esv1beta1.ReasonDeprecated, fmt.Sprintf("dataFrom[%d].find.conversionStrategy=%v is deprecated and will be removed in further releases. Use dataFrom.rewrite instead", i, remoteRef.Find.ConversionStrategy))
		secretMap, err = utils.ConvertKeys(remoteRef.Find.ConversionStrategy, secretMap)
		if err != nil {
			return nil, fmt.Errorf(errConvert, err)
		}
	}
	if !utils.ValidateKeys(secretMap) {
		return nil, fmt.Errorf(errInvalidKeys, "find", i)
	}
	secretMap, err = utils.DecodeMap(remoteRef.Find.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errDecode, "spec.dataFrom", i, err)
	}
	return secretMap, err
}

func shouldSkipGenerator(r *Reconciler, generatorDef *apiextensions.JSON) (bool, error) {
	var genControllerClass genv1alpha1.ControllerClassResource
	err := json.Unmarshal(generatorDef.Raw, &genControllerClass)
	if err != nil {
		return false, err
	}
	var controllerClass = genControllerClass.Spec.ControllerClass
	return controllerClass != "" && controllerClass != r.ControllerClass, nil
}
