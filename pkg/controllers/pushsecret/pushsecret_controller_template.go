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
	"github.com/external-secrets/external-secrets/pkg/template"
	"github.com/external-secrets/external-secrets/pkg/utils"

	_ "github.com/external-secrets/external-secrets/pkg/provider/register" // Loading registered providers.
)

const (
	errFetchTplFrom = "error fetching templateFrom data: %w"
	errExecTpl      = "could not execute template: %w"
)

// applyTemplate merges template in the following order:
// * template.Data (highest precedence)
// * template.templateFrom
// * secret via ps.data or ps.dataFrom.
// Apply template modifications for the source secret. These modifications will only live in memory as we will
// never modify it.
func (r *Reconciler) applyTemplate(ctx context.Context, ps *v1alpha1.PushSecret, secret *v1.Secret) error {
	// no template: nothing to do
	if ps.Spec.Template == nil {
		return nil
	}

	if err := setMetadata(secret, ps); err != nil {
		return err
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

	return nil
}

// setMetadata sets Labels and Annotations in the source secret, but we will never write them back.
// It is only set to satisfy templated changes.
func setMetadata(secret *v1.Secret, ps *v1alpha1.PushSecret) error {
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	secret.Type = ps.Spec.Template.Type
	utils.MergeStringMap(secret.ObjectMeta.Labels, ps.Spec.Template.Metadata.Labels)
	utils.MergeStringMap(secret.ObjectMeta.Annotations, ps.Spec.Template.Metadata.Annotations)

	return nil
}
