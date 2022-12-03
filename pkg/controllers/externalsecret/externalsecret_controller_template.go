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
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
	"github.com/external-secrets/external-secrets/pkg/template"
	utils "github.com/external-secrets/external-secrets/pkg/utils"
)

// merge template in the following order:
// * template.Data (highest precedence)
// * template.templateFrom
// * secret via es.data or es.dataFrom.
func (r *Reconciler) applyTemplate(ctx context.Context, es *esv1beta1.ExternalSecret, secret *v1.Secret, dataMap map[string][]byte) error {
	mergeMetadata(secret, es)

	// no template: copy data and return
	if es.Spec.Target.Template == nil {
		secret.Data = dataMap
		secret.Annotations[esv1beta1.AnnotationDataHash] = utils.ObjectHash(secret.Data)
		return nil
	}
	execute, err := template.EngineForVersion(es.Spec.Target.Template.EngineVersion)
	if err != nil {
		return err
	}

	// apply templates defined in template.templateFrom
	err = r.mergeTemplateFrom(ctx, es, execute, secret, dataMap)
	if err != nil {
		return fmt.Errorf(errFetchTplFrom, err)
	}
	// explicitly defined template.Data takes precedence over templateFrom
	tplMap := make(map[string][]byte)
	for k, v := range es.Spec.Target.Template.Data {
		tplMap[k] = []byte(v)
	}
	err = execute(tplMap, dataMap, esv1beta1.TemplateScopeValues, esv1beta1.TemplateTargetData, secret)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	r.Log.V(1).Info("found template data", "tpl_data", tplMap)

	tplMapLabels := make(map[string][]byte)
	tplMapAnnotations := make(map[string][]byte)

	// get template data for labels
	if es.Spec.Target.Template.Metadata.Labels != nil {
		for k, v := range es.Spec.Target.Template.Metadata.Labels {
			tplMapLabels[k] = []byte(v)
		}
		r.Log.V(1).Info("found template metadata (labels)", "tpl_labels", tplMapLabels)
	}
	err = execute(tplMapLabels, dataMap, esv1beta1.TemplateScopeValues, esv1beta1.TemplateTargetLabels, secret)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// get template data for annotations
	if es.Spec.Target.Template.Metadata.Annotations != nil {
		for k, v := range es.Spec.Target.Template.Metadata.Annotations {
			tplMapAnnotations[k] = []byte(v)
		}
		r.Log.V(1).Info("found template metadata (annotations)", "tpl_annotations", tplMapAnnotations)
	}
	err = execute(tplMapAnnotations, dataMap, esv1beta1.TemplateScopeValues, esv1beta1.TemplateTargetAnnotations, secret)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// if no data was provided by template fallback
	// to value from the provider
	if len(es.Spec.Target.Template.Data) == 0 && len(es.Spec.Target.Template.TemplateFrom) == 0 {
		secret.Data = dataMap
	}
	secret.Annotations[esv1beta1.AnnotationDataHash] = utils.ObjectHash(secret.Data)

	return nil
}

// we do not want to force-override the label/annotations
// and only copy the necessary key/value pairs.
func mergeMetadata(secret *v1.Secret, externalSecret *esv1beta1.ExternalSecret) {
	if secret.ObjectMeta.Labels == nil {
		secret.ObjectMeta.Labels = make(map[string]string)
	}
	if secret.ObjectMeta.Annotations == nil {
		secret.ObjectMeta.Annotations = make(map[string]string)
	}
	if externalSecret.Spec.Target.Template == nil {
		utils.MergeStringMap(secret.ObjectMeta.Labels, externalSecret.ObjectMeta.Labels)
		utils.MergeStringMap(secret.ObjectMeta.Annotations, externalSecret.ObjectMeta.Annotations)
		return
	}
	// if template is defined: use those labels/annotations
	secret.Type = externalSecret.Spec.Target.Template.Type
	utils.MergeStringMap(secret.ObjectMeta.Labels, externalSecret.Spec.Target.Template.Metadata.Labels)
	utils.MergeStringMap(secret.ObjectMeta.Annotations, externalSecret.Spec.Target.Template.Metadata.Annotations)
}

func (r *Reconciler) mergeTemplateFrom(ctx context.Context, externalSecret *esv1beta1.ExternalSecret, execute template.ExecFunc, secret *v1.Secret, dataMap map[string][]byte) error {
	if externalSecret.Spec.Target.Template == nil {
		return nil
	}
	for _, tpl := range externalSecret.Spec.Target.Template.TemplateFrom {
		out := make(map[string][]byte)
		// Literal is only compatible with KeysAndValues scope
		if tpl.Literal != nil && tpl.Scope == esv1beta1.TemplateScopeKeysAndValues {
			out[*tpl.Literal] = []byte(*tpl.Literal)
		}
		err := mergeConfigMap(ctx, r.Client, externalSecret, tpl, out)
		if err != nil {
			return err
		}
		err = mergeSecret(ctx, r.Client, externalSecret, tpl, out)
		if err != nil {
			return err
		}
		err = execute(out, dataMap, tpl.Scope, tpl.Target, secret)
		if err != nil {
			return err
		}
	}
	return nil
}

func mergeConfigMap(ctx context.Context, k8sClient client.Client, es *esv1beta1.ExternalSecret, tpl esv1beta1.TemplateFrom, out map[string][]byte) error {
	if tpl.ConfigMap == nil {
		return nil
	}

	var cm v1.ConfigMap
	err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      tpl.ConfigMap.Name,
		Namespace: es.Namespace,
	}, &cm)
	if err != nil {
		return err
	}
	for _, k := range tpl.ConfigMap.Items {
		val, ok := cm.Data[k.Key]
		if !ok {
			return fmt.Errorf(errTplCMMissingKey, tpl.ConfigMap.Name, k.Key)
		}
		out[k.Key] = []byte(val)
	}
	return nil
}

func mergeSecret(ctx context.Context, k8sClient client.Client, es *esv1beta1.ExternalSecret, tpl esv1beta1.TemplateFrom, out map[string][]byte) error {
	if tpl.Secret == nil {
		return nil
	}
	var sec v1.Secret
	err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      tpl.Secret.Name,
		Namespace: es.Namespace,
	}, &sec)
	if err != nil {
		return err
	}
	for _, k := range tpl.Secret.Items {
		val, ok := sec.Data[k.Key]
		if !ok {
			return fmt.Errorf(errTplSecMissingKey, tpl.Secret.Name, k.Key)
		}
		out[k.Key] = val
	}
	return nil
}
