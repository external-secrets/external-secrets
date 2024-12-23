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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"

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
			secretMap, err = r.handleFindAllSecrets(ctx, externalSecret, remoteRef, mgr)
			if err != nil {
				err = fmt.Errorf("error processing spec.dataFrom[%d].find, err: %w", i, err)
			}
		} else if remoteRef.Extract != nil {
			secretMap, err = r.handleExtractSecrets(ctx, externalSecret, remoteRef, mgr)
			if err != nil {
				err = fmt.Errorf("error processing spec.dataFrom[%d].extract, err: %w", i, err)
			}
		} else if remoteRef.SourceRef != nil && remoteRef.SourceRef.GeneratorRef != nil {
			secretMap, err = r.handleGenerateSecrets(ctx, externalSecret.Namespace, remoteRef)
			if err != nil {
				err = fmt.Errorf("error processing spec.dataFrom[%d].sourceRef.generatorRef, err: %w", i, err)
			}
		}

		if errors.Is(err, esv1beta1.NoSecretErr) && externalSecret.Spec.Target.DeletionPolicy != esv1beta1.DeletionPolicyRetain {
			r.recorder.Eventf(externalSecret, v1.EventTypeNormal, esv1beta1.ReasonMissingProviderSecret, eventMissingProviderSecret, i)
			continue
		}
		if err != nil {
			return nil, err
		}

		providerData = utils.MergeByteMap(providerData, secretMap)
	}

	for i, secretRef := range externalSecret.Spec.Data {
		err := r.handleSecretData(ctx, *externalSecret, secretRef, providerData, mgr)
		if errors.Is(err, esv1beta1.NoSecretErr) && externalSecret.Spec.Target.DeletionPolicy != esv1beta1.DeletionPolicyRetain {
			r.recorder.Eventf(externalSecret, v1.EventTypeNormal, esv1beta1.ReasonMissingProviderSecret, eventMissingProviderSecretKey, i, secretRef.RemoteRef.Key)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error processing spec.data[%d] (key: %s), err: %w", i, secretRef.RemoteRef.Key, err)
		}
	}

	return providerData, nil
}

func (r *Reconciler) handleSecretData(ctx context.Context, externalSecret esv1beta1.ExternalSecret, secretRef esv1beta1.ExternalSecretData, providerData map[string][]byte, cmgr *secretstore.Manager) error {
	client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, toStoreGenSourceRef(secretRef.SourceRef))
	if err != nil {
		return err
	}

	// get a single secret from the store
	secretData, err := client.GetSecret(ctx, secretRef.RemoteRef)
	if err != nil {
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

func toStoreGenSourceRef(ref *esv1beta1.StoreSourceRef) *esv1beta1.StoreGeneratorSourceRef {
	if ref == nil {
		return nil
	}
	return &esv1beta1.StoreGeneratorSourceRef{
		SecretStoreRef: &ref.SecretStoreRef,
	}
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

func (r *Reconciler) handleExtractSecrets(ctx context.Context, externalSecret *esv1beta1.ExternalSecret, remoteRef esv1beta1.ExternalSecretDataFromRemoteRef, cmgr *secretstore.Manager) (map[string][]byte, error) {
	client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}

	// get multiple secrets from the store
	secretMap, err := client.GetSecretMap(ctx, *remoteRef.Extract)
	if err != nil {
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

func (r *Reconciler) handleFindAllSecrets(ctx context.Context, externalSecret *esv1beta1.ExternalSecret, remoteRef esv1beta1.ExternalSecretDataFromRemoteRef, cmgr *secretstore.Manager) (map[string][]byte, error) {
	client, err := cmgr.Get(ctx, externalSecret.Spec.SecretStoreRef, externalSecret.Namespace, remoteRef.SourceRef)
	if err != nil {
		return nil, err
	}

	// get all secrets from the store that match the selector
	secretMap, err := client.GetAllSecrets(ctx, *remoteRef.Find)
	if err != nil {
		return nil, fmt.Errorf("error getting all secrets: %w", err)
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

func shouldSkipGenerator(r *Reconciler, generatorDef *apiextensions.JSON) (bool, error) {
	var genControllerClass genv1alpha1.ControllerClassResource
	err := json.Unmarshal(generatorDef.Raw, &genControllerClass)
	if err != nil {
		return false, err
	}
	var controllerClass = genControllerClass.Spec.ControllerClass
	return controllerClass != "" && controllerClass != r.ControllerClass, nil
}
