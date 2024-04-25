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

package templating

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/template"
)

const fieldOwnerTemplate = "externalsecrets.external-secrets.io/%v"

var (
	errTplCMMissingKey  = "error in configmap %s: missing key %s"
	errTplSecMissingKey = "error in secret %s: missing key %s"
	errExecTpl          = "could not execute template: %w"
)

type Parser struct {
	Exec         template.ExecFunc
	DataMap      map[string][]byte
	Client       client.Client
	TargetSecret *v1.Secret
}

func (p *Parser) MergeConfigMap(ctx context.Context, namespace string, tpl esv1beta1.TemplateFrom) error {
	if tpl.ConfigMap == nil {
		return nil
	}
	var cm v1.ConfigMap
	err := p.Client.Get(ctx, types.NamespacedName{
		Name:      tpl.ConfigMap.Name,
		Namespace: namespace,
	}, &cm)
	if err != nil {
		return err
	}
	for _, k := range tpl.ConfigMap.Items {
		val, ok := cm.Data[k.Key]
		out := make(map[string][]byte)
		if !ok {
			return fmt.Errorf(errTplCMMissingKey, tpl.ConfigMap.Name, k.Key)
		}
		switch k.TemplateAs {
		case esv1beta1.TemplateScopeValues:
			out[k.Key] = []byte(val)
		case esv1beta1.TemplateScopeKeysAndValues:
			out[val] = []byte(val)
		}
		err = p.Exec(out, p.DataMap, k.TemplateAs, tpl.Target, p.TargetSecret)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) MergeSecret(ctx context.Context, namespace string, tpl esv1beta1.TemplateFrom) error {
	if tpl.Secret == nil {
		return nil
	}
	var sec v1.Secret
	err := p.Client.Get(ctx, types.NamespacedName{
		Name:      tpl.Secret.Name,
		Namespace: namespace,
	}, &sec)
	if err != nil {
		return err
	}
	for _, k := range tpl.Secret.Items {
		val, ok := sec.Data[k.Key]
		if !ok {
			return fmt.Errorf(errTplSecMissingKey, tpl.Secret.Name, k.Key)
		}
		out := make(map[string][]byte)
		switch k.TemplateAs {
		case esv1beta1.TemplateScopeValues:
			out[k.Key] = val
		case esv1beta1.TemplateScopeKeysAndValues:
			out[string(val)] = val
		}
		err = p.Exec(out, p.DataMap, k.TemplateAs, tpl.Target, p.TargetSecret)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) MergeLiteral(_ context.Context, tpl esv1beta1.TemplateFrom) error {
	if tpl.Literal == nil {
		return nil
	}
	out := make(map[string][]byte)
	out[*tpl.Literal] = []byte(*tpl.Literal)
	return p.Exec(out, p.DataMap, esv1beta1.TemplateScopeKeysAndValues, tpl.Target, p.TargetSecret)
}

func (p *Parser) MergeTemplateFrom(ctx context.Context, namespace string, template *esv1beta1.ExternalSecretTemplate) error {
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

func (p *Parser) MergeMap(tplMap map[string]string, target esv1beta1.TemplateTarget) error {
	byteMap := make(map[string][]byte)
	for k, v := range tplMap {
		byteMap[k] = []byte(v)
	}
	err := p.Exec(byteMap, p.DataMap, esv1beta1.TemplateScopeValues, target, p.TargetSecret)
	if err != nil {
		return fmt.Errorf(errExecTpl, err)
	}
	return nil
}

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
	fqdn := fmt.Sprintf(fieldOwnerTemplate, fieldOwner)
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
