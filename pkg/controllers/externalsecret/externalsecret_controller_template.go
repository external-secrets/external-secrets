/*
Copyright © 2025 ESO Maintainer Team

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

package externalsecret

import (
	"context"
	"fmt"
	"maps"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/external-secrets/external-secrets/pkg/template"

	_ "github.com/external-secrets/external-secrets/pkg/provider/register" // Loading registered providers.
)

// ApplyTemplate merges templates in the following order:
// * template.Data (highest precedence)
// * template.TemplateFrom
// * secret via es.data or es.dataFrom (if template.MergePolicy is Merge, or there is no template)
// * existing secret keys (if CreationPolicy is Merge).
func (r *Reconciler) ApplyTemplate(ctx context.Context, es *esv1.ExternalSecret, secret *v1.Secret, dataMap map[string][]byte) error {
	// update metadata (labels, annotations, finalizers) of the secret
	if err := setMetadata(secret, es); err != nil {
		return err
	}

	// we only keep existing keys if creation policy is Merge, otherwise we clear the secret
	if es.Spec.Target.CreationPolicy != esv1.CreatePolicyMerge {
		secret.Data = make(map[string][]byte)
	}

	// no template: copy data and return
	if es.Spec.Target.Template == nil {
		maps.Insert(secret.Data, maps.All(dataMap))
		return nil
	}

	// set the secret type if it is defined in the template, otherwise keep the existing type
	if es.Spec.Target.Template.Type != "" {
		secret.Type = es.Spec.Target.Template.Type
	}

	// when TemplateMergePolicy is Merge, or there is no data template, we include the keys from `dataMap`
	noTemplate := len(es.Spec.Target.Template.Data) == 0 && len(es.Spec.Target.Template.TemplateFrom) == 0
	if es.Spec.Target.Template.MergePolicy == esv1.MergePolicyMerge || noTemplate {
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
	err = p.MergeMap(es.Spec.Target.Template.Data, esv1.TemplateTargetData)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// apply templates for labels
	// NOTE: this only works for v2 templates
	err = p.MergeMap(es.Spec.Target.Template.Metadata.Labels, esv1.TemplateTargetLabels)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// apply template for annotations
	// NOTE: this only works for v2 templates
	err = p.MergeMap(es.Spec.Target.Template.Metadata.Annotations, esv1.TemplateTargetAnnotations)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	return nil
}

// setMetadata sets Labels and Annotations to the given secret.
func setMetadata(secret *v1.Secret, es *esv1.ExternalSecret) error {
	// ensure that Labels and Annotations are not nil
	// so it is safe to merge them
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	// remove any existing labels managed by this external secret
	// this is to ensure that we don't have any stale labels
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

	// if no template is defined, copy labels and annotations from the ExternalSecret
	if es.Spec.Target.Template == nil {
		esutils.MergeStringMap(secret.ObjectMeta.Labels, es.ObjectMeta.Labels)
		esutils.MergeStringMap(secret.ObjectMeta.Annotations, es.ObjectMeta.Annotations)
		return nil
	}

	// copy labels and annotations from the template
	esutils.MergeStringMap(secret.ObjectMeta.Labels, es.Spec.Target.Template.Metadata.Labels)
	esutils.MergeStringMap(secret.ObjectMeta.Annotations, es.Spec.Target.Template.Metadata.Annotations)

	// add finalizers from the template
	if secret.ObjectMeta.DeletionTimestamp.IsZero() {
		for _, finalizer := range es.Spec.Target.Template.Metadata.Finalizers {
			if !controllerutil.ContainsFinalizer(secret, finalizer) {
				controllerutil.AddFinalizer(secret, finalizer)
			}
		}
	}

	return nil
}
