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

package pushsecret

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	_ "github.com/external-secrets/external-secrets/pkg/provider/register" // Loading registered providers.
	"github.com/external-secrets/external-secrets/pkg/template"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errFetchTplFrom = "error fetching templateFrom data: %w"
	errExecTpl      = "could not execute template: %w"
)

// merge template in the following order:
// * template.Data (highest precedence)
// * template.templateFrom
// * secret via es.data or es.dataFrom.
// Whatever is in the Secret THAT'S the Data.
func (r *Reconciler) applyTemplate(ctx context.Context, ps *v1alpha1.PushSecret, secret *v1.Secret) error {
	if err := setMetadata(secret, ps); err != nil {
		return err
	}

	// no template: copy data and return
	if ps.Spec.Template == nil {
		return nil
	}

	execute, err := template.EngineForVersion(esv1beta1.TemplateEngineV2)
	if err != nil {
		return err
	}

	p := templating.Parser{
		Client:       r.Client,
		TargetSecret: secret,
		DataMap:      secret.Data,
		Exec:         execute,
	}

	// apply templates defined in template.templateFrom
	err = p.MergeTemplateFrom(ctx, ps.Namespace, ps.Spec.Template)
	if err != nil {
		return fmt.Errorf(errFetchTplFrom, err)
	}
	// explicitly defined template.Data takes precedence over templateFrom
	err = p.MergeMap(ps.Spec.Template.Data, esv1beta1.TemplateTargetData)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}

	// get template data for labels
	err = p.MergeMap(ps.Spec.Template.Metadata.Labels, esv1beta1.TemplateTargetLabels)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}
	// get template data for annotations
	err = p.MergeMap(ps.Spec.Template.Metadata.Annotations, esv1beta1.TemplateTargetAnnotations)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}
	// if no data was provided by template fallback
	// to value from the provider
	// no provider, we already have the secret...
	// if len(ps.Spec.Template.Data) == 0 && len(ps.Spec.Template.TemplateFrom) == 0 {
	//	 secret.Data = dataMap
	// }

	return nil
}

// setMetadata sets Labels and Annotations to the given secret.
func setMetadata(secret *v1.Secret, es *v1alpha1.PushSecret) error {
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

	if es.Spec.Template == nil {
		utils.MergeStringMap(secret.ObjectMeta.Labels, es.ObjectMeta.Labels)
		utils.MergeStringMap(secret.ObjectMeta.Annotations, es.ObjectMeta.Annotations)

		return nil
	}

	secret.Type = es.Spec.Template.Type
	utils.MergeStringMap(secret.ObjectMeta.Labels, es.Spec.Template.Metadata.Labels)
	utils.MergeStringMap(secret.ObjectMeta.Annotations, es.Spec.Template.Metadata.Annotations)

	return nil
}
