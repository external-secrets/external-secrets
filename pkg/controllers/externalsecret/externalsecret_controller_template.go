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
	"maps"

	v1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/template"
	"github.com/external-secrets/external-secrets/pkg/utils"

	_ "github.com/external-secrets/external-secrets/pkg/provider/register" // Loading registered providers.
)

// merge template in the following order:
// * template.Data (highest precedence)
// * template.TemplateFrom
// * secret via es.data or es.dataFrom (if template.MergePolicy is Merge, or there is no template)
// * existing secret keys (if CreationPolicy is Merge).
func (r *Reconciler) applyTemplate(ctx context.Context, es *esv1beta1.ExternalSecret, secret *v1.Secret, dataMap map[string][]byte) error {
	// initialize maps within the secret so it's safe to set values
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	// remove managed fields from the secret
	// NOTE: this ensures templated fields are not left behind when they are removed from the template
	if err := removeManagedFields(secret, es.Name); err != nil {
		return err
	}

	// set labels and annotations (either from the template or the ExternalSecret itself)
	if err := setMetadata(secret, es); err != nil {
		return err
	}

	// clear data if the creation policy is not merge
	// NOTE: this is because for other policies, the template is "declarative" and should be the source of truth
	if es.Spec.Target.CreationPolicy != esv1beta1.CreatePolicyMerge {
		secret.Data = make(map[string][]byte)
	}

	// no template: copy data and return
	if es.Spec.Target.Template == nil {
		maps.Insert(secret.Data, maps.All(dataMap))
		return nil
	}

	// set the secret type if it is defined in the template, otherwise keep the existing type
	// NOTE: this prevents update loops because Kubernetes sets the type to "Opaque" if it is not defined,
	//       and so explicitly setting it to "" every time would cause a needless update
	if es.Spec.Target.Template.Type != "" {
		secret.Type = es.Spec.Target.Template.Type
	}

	// when TemplateMergePolicy is Merge, or there is no data template, we include the keys from `dataMap`
	noTemplate := len(es.Spec.Target.Template.Data) == 0 && len(es.Spec.Target.Template.TemplateFrom) == 0
	if es.Spec.Target.Template.MergePolicy == esv1beta1.MergePolicyMerge || noTemplate {
		maps.Insert(secret.Data, maps.All(dataMap))
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

	// apply data templates
	// NOTE: explicitly defined template.data templates take precedence over templateFrom
	err = p.MergeMap(es.Spec.Target.Template.Data, esv1beta1.TemplateTargetData)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// apply templates for labels
	// NOTE: this only works for v2 templates
	err = p.MergeMap(es.Spec.Target.Template.Metadata.Labels, esv1beta1.TemplateTargetLabels)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// apply template for annotations
	// NOTE: this only works for v2 templates
	err = p.MergeMap(es.Spec.Target.Template.Metadata.Annotations, esv1beta1.TemplateTargetAnnotations)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	return nil
}

// setMetadata sets the labels and annotations on the secret, either from the template or the ExternalSecret itself.
func setMetadata(secret *v1.Secret, es *esv1beta1.ExternalSecret) error {
	// if no template is defined, copy labels and annotations from the ExternalSecret
	if es.Spec.Target.Template == nil {
		utils.MergeStringMap(secret.Labels, es.Labels)
		utils.MergeStringMap(secret.Annotations, es.Annotations)
		return nil
	}

	// otherwise, copy labels and annotations from the template
	utils.MergeStringMap(secret.Labels, es.Spec.Target.Template.Metadata.Labels)
	utils.MergeStringMap(secret.Annotations, es.Spec.Target.Template.Metadata.Annotations)
	return nil
}

// removeManagedFields removes all fields managed by a given fieldOwner from the secret.
func removeManagedFields(secret *v1.Secret, fieldOwner string) error {
	// remove managed annotation keys
	annotationKeys, err := templating.GetManagedAnnotationKeys(secret, fieldOwner)
	if err != nil {
		return err
	}
	for i := range annotationKeys {
		delete(secret.Annotations, annotationKeys[i])
	}

	// remove managed label keys
	labelKeys, err := templating.GetManagedLabelKeys(secret, fieldOwner)
	if err != nil {
		return err
	}
	for i := range labelKeys {
		delete(secret.Labels, labelKeys[i])
	}

	// remove managed data keys
	dataKeys, err := templating.GetManagedDataKeys(secret, fieldOwner)
	if err != nil {
		return err
	}
	for i := range dataKeys {
		delete(secret.Data, dataKeys[i])
	}

	return nil
}
