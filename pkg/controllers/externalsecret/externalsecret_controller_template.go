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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/template"
	"github.com/external-secrets/external-secrets/pkg/utils"

	_ "github.com/external-secrets/external-secrets/pkg/provider/register" // Loading registered providers.
)

// merge template in the following order:
// * template.Data (highest precedence)
// * template.templateFrom
// * secret via es.data or es.dataFrom.
func (r *Reconciler) applyTemplate(ctx context.Context, es *esv1beta1.ExternalSecret, secret *v1.Secret, dataMap map[string][]byte) error {
	if err := setMetadata(secret, es); err != nil {
		return err
	}

	// no template: copy data and return
	if es.Spec.Target.Template == nil {
		secret.Data = dataMap
		return nil
	}
	// Merge Policy should merge secrets
	if es.Spec.Target.Template.MergePolicy == esv1beta1.MergePolicyMerge {
		for k, v := range dataMap {
			secret.Data[k] = v
		}
	}
	execute, err := template.EngineForVersion(es.Spec.Target.Template.EngineVersion)
	if err != nil {
		return err
	}

	p := templating.Parser{
		Client:       r.Client,
		TargetSecret: secret,
		DataMap:      dataMap,
		Exec:         execute,
	}
	// apply templates defined in template.templateFrom
	err = p.MergeTemplateFrom(ctx, es.Namespace, es.Spec.Target.Template)
	if err != nil {
		return fmt.Errorf(errFetchTplFrom, err)
	}
	// explicitly defined template.Data takes precedence over templateFrom
	err = p.MergeMap(es.Spec.Target.Template.Data, esv1beta1.TemplateTargetData)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// get template data for labels
	err = p.MergeMap(es.Spec.Target.Template.Metadata.Labels, esv1beta1.TemplateTargetLabels)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}
	// get template data for annotations
	err = p.MergeMap(es.Spec.Target.Template.Metadata.Annotations, esv1beta1.TemplateTargetAnnotations)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}
	// if no data was provided by template fallback
	// to value from the provider
	if len(es.Spec.Target.Template.Data) == 0 && len(es.Spec.Target.Template.TemplateFrom) == 0 {
		secret.Data = dataMap
	}
	return nil
}

// setMetadata sets Labels and Annotations to the given secret.
func setMetadata(secret *v1.Secret, es *esv1beta1.ExternalSecret) error {
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	// Clean up Labels and Annotations added by the operator
	// so that it won't leave outdated ones
	labelKeys, err := templating.GetManagedLabelKeys(secret, es.Name)
	if err != nil {
		return err
	}
	for _, key := range labelKeys {
		delete(secret.ObjectMeta.Labels, key)
	}

	annotationKeys, err := templating.GetManagedAnnotationKeys(secret, es.Name)
	if err != nil {
		return err
	}
	for _, key := range annotationKeys {
		delete(secret.ObjectMeta.Annotations, key)
	}

	if es.Spec.Target.Template == nil {
		utils.MergeStringMap(secret.ObjectMeta.Labels, es.ObjectMeta.Labels)
		utils.MergeStringMap(secret.ObjectMeta.Annotations, es.ObjectMeta.Annotations)
		return nil
	}

	secret.Type = es.Spec.Target.Template.Type
	utils.MergeStringMap(secret.ObjectMeta.Labels, es.Spec.Target.Template.Metadata.Labels)
	utils.MergeStringMap(secret.ObjectMeta.Annotations, es.Spec.Target.Template.Metadata.Annotations)
	return nil
}
