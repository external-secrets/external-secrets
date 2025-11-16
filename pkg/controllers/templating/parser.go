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

// Package templating provides functionality for templating secret data.
package templating

import (
	"context"
	"crypto/sha3"
	"encoding/json"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/template"
)

const fieldOwnerTemplate = "externalsecrets.external-secrets.io/%v"
const fieldOwnerTemplateSha = "externalsecrets.external-secrets.io/sha3/%x"

var (
	errTplCMMissingKey  = "error in configmap %s: missing key %s"
	errTplSecMissingKey = "error in secret %s: missing key %s"
	errExecTpl          = "could not execute template: %w"
)

// Parser is responsible for parsing and merging templates into a target secret.
type Parser struct {
	Exec         template.ExecFunc
	DataMap      map[string][]byte
	Client       client.Client
	TargetSecret *v1.Secret

	TemplateFromConfigMap *v1.ConfigMap
	TemplateFromSecret    *v1.Secret
}

// MergeConfigMap merges the configmap template specified in the ExternalSecretTemplate's TemplateFrom field.
func (p *Parser) MergeConfigMap(ctx context.Context, namespace string, tpl esv1.TemplateFrom) error {
	if tpl.ConfigMap == nil {
		return nil
	}

	var cm v1.ConfigMap
	if p.TemplateFromConfigMap != nil {
		cm = *p.TemplateFromConfigMap
	} else {
		err := p.Client.Get(ctx, types.NamespacedName{
			Name:      tpl.ConfigMap.Name,
			Namespace: namespace,
		}, &cm)
		if err != nil {
			return err
		}
	}

	for _, k := range tpl.ConfigMap.Items {
		val, ok := cm.Data[k.Key]
		out := make(map[string][]byte)
		if !ok {
			return fmt.Errorf(errTplCMMissingKey, tpl.ConfigMap.Name, k.Key)
		}
		switch k.TemplateAs {
		case esv1.TemplateScopeValues:
			out[k.Key] = []byte(val)
		case esv1.TemplateScopeKeysAndValues:
			out[val] = []byte(val)
		}
		err := p.Exec(out, p.DataMap, k.TemplateAs, tpl.Target, p.TargetSecret)
		if err != nil {
			return err
		}
	}
	return nil
}

// MergeSecret merges the secret template specified in the ExternalSecretTemplate's TemplateFrom field.
func (p *Parser) MergeSecret(ctx context.Context, namespace string, tpl esv1.TemplateFrom) error {
	if tpl.Secret == nil {
		return nil
	}

	var sec v1.Secret
	if p.TemplateFromSecret != nil {
		sec = *p.TemplateFromSecret
	} else {
		err := p.Client.Get(ctx, types.NamespacedName{
			Name:      tpl.Secret.Name,
			Namespace: namespace,
		}, &sec)
		if err != nil {
			return err
		}
	}

	for _, k := range tpl.Secret.Items {
		val, ok := sec.Data[k.Key]
		if !ok {
			return fmt.Errorf(errTplSecMissingKey, tpl.Secret.Name, k.Key)
		}
		out := make(map[string][]byte)
		switch k.TemplateAs {
		case esv1.TemplateScopeValues:
			out[k.Key] = val
		case esv1.TemplateScopeKeysAndValues:
			out[string(val)] = val
		}
		err := p.Exec(out, p.DataMap, k.TemplateAs, tpl.Target, p.TargetSecret)
		if err != nil {
			return err
		}
	}
	return nil
}

// MergeLiteral merges the literal template specified in the ExternalSecretTemplate's TemplateFrom field.
func (p *Parser) MergeLiteral(_ context.Context, tpl esv1.TemplateFrom) error {
	if tpl.Literal == nil {
		return nil
	}
	out := make(map[string][]byte)
	out[*tpl.Literal] = []byte(*tpl.Literal)
	return p.Exec(out, p.DataMap, esv1.TemplateScopeKeysAndValues, tpl.Target, p.TargetSecret)
}

// MergeTemplateFrom merges all templates specified in the ExternalSecretTemplate's TemplateFrom field.
func (p *Parser) MergeTemplateFrom(ctx context.Context, namespace string, template *esv1.ExternalSecretTemplate) error {
	if template == nil {
		return nil
	}

	for _, tpl := range template.TemplateFrom {
		err := p.MergeConfigMap(ctx, namespace, tpl)
		if err != nil {
			return err
		}
		err = p.MergeSecret(ctx, namespace, tpl)
		if err != nil {
			return err
		}
		err = p.MergeLiteral(ctx, tpl)
		if err != nil {
			return err
		}
	}
	return nil
}

// MergeMap merges the given map of templates into the target secret.
func (p *Parser) MergeMap(tplMap map[string]string, target string) error {
	byteMap := make(map[string][]byte)
	for k, v := range tplMap {
		byteMap[k] = []byte(v)
	}
	err := p.Exec(byteMap, p.DataMap, esv1.TemplateScopeValues, target, p.TargetSecret)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}
	return nil
}

// GetManagedAnnotationKeys returns the keys of the annotations managed by the given field owner.
func GetManagedAnnotationKeys(secret *v1.Secret, fieldOwner string) ([]string, error) {
	return getManagedFieldKeys(secret, fieldOwner, func(fields map[string]any) []string {
		metadataFields, exists := fields["f:metadata"]
		if !exists {
			return nil
		}
		mf, ok := metadataFields.(map[string]any)
		if !ok {
			return nil
		}
		annotationFields, exists := mf["f:annotations"]
		if !exists {
			return nil
		}
		af, ok := annotationFields.(map[string]any)
		if !ok {
			return nil
		}
		var keys []string
		for k := range af {
			keys = append(keys, k)
		}
		return keys
	})
}

// GetManagedLabelKeys returns the keys of labels that are managed by the given field owner.
// It checks the ManagedFields of the secret for entries with the specified field owner
// and extracts the keys of the labels from the fields managed by that owner.
func GetManagedLabelKeys(secret *v1.Secret, fieldOwner string) ([]string, error) {
	return getManagedFieldKeys(secret, fieldOwner, func(fields map[string]any) []string {
		metadataFields, exists := fields["f:metadata"]
		if !exists {
			return nil
		}
		mf, ok := metadataFields.(map[string]any)
		if !ok {
			return nil
		}
		labelFields, exists := mf["f:labels"]
		if !exists {
			return nil
		}
		lf, ok := labelFields.(map[string]any)
		if !ok {
			return nil
		}
		var keys []string
		for k := range lf {
			keys = append(keys, k)
		}
		return keys
	})
}

func getManagedFieldKeys(
	secret *v1.Secret,
	fieldOwner string,
	process func(fields map[string]any) []string,
) ([]string, error) {
	// If secret name is just too big, use the SHA3 hash of the secret name
	// Done this way for backwards compatibility thus avoiding breaking changes
	fqdn := fmt.Sprintf(fieldOwnerTemplate, fieldOwner)
	if len(fieldOwner) > 63 {
		fqdn = fmt.Sprintf(fieldOwnerTemplateSha, sha3.Sum224([]byte(fieldOwner)))
	}
	var keys []string
	for _, v := range secret.ObjectMeta.ManagedFields {
		if v.Manager != fqdn {
			continue
		}
		fields := make(map[string]any)
		err := json.Unmarshal(v.FieldsV1.Raw, &fields)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling managed fields: %w", err)
		}
		for _, key := range process(fields) {
			if key == "." {
				continue
			}
			keys = append(keys, strings.TrimPrefix(key, "f:"))
		}
	}
	return keys, nil
}
