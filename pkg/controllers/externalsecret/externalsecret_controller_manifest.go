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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/template"
)

// isNonSecretTarget checks if the ExternalSecret targets a non-Secret resource.
func isNonSecretTarget(es *esv1.ExternalSecret) bool {
	return es.Spec.Target.Manifest != nil
}

// validateNonSecretTarget validates that non-Secret targets are properly configured.
func (r *Reconciler) validateNonSecretTarget(log logr.Logger, es *esv1.ExternalSecret) error {
	if !r.AllowNonSecretTargets {
		return fmt.Errorf("non-Secret targets are disabled. Enable with --unsafe-allow-non-secret-targets flag")
	}

	manifest := es.Spec.Target.Manifest
	if manifest.APIVersion == "" {
		return fmt.Errorf("target.manifest.apiVersion is required")
	}
	if manifest.Kind == "" {
		return fmt.Errorf("target.manifest.kind is required")
	}

	log.Info("WARNING: Using non-Secret target. Data will not be encrypted at rest.",
		"apiVersion", manifest.APIVersion,
		"kind", manifest.Kind,
		"name", getTargetName(es))

	return nil
}

// getTargetGVK returns the GroupVersionKind for the target resource.
func getTargetGVK(es *esv1.ExternalSecret) schema.GroupVersionKind {
	manifest := es.Spec.Target.Manifest
	gv, _ := schema.ParseGroupVersion(manifest.APIVersion)

	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    manifest.Kind,
	}
}

// getTargetName returns the name of the target resource.
func getTargetName(es *esv1.ExternalSecret) string {
	if es.Spec.Target.Name != "" {
		return es.Spec.Target.Name
	}
	return es.Name
}

// getNonSecretResource retrieves a non-Secret resource using the dynamic client.
func (r *Reconciler) getNonSecretResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret) (*unstructured.Unstructured, error) {
	gvk := getTargetGVK(es)
	gvr, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{Kind: gvk.Kind, Group: gvk.Group, Version: gvk.Version})

	resource, err := r.DynamicClient.Resource(gvr).Namespace(es.Namespace).Get(ctx, getTargetName(es), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("target resource does not exist", "gvk", gvk.String(), "name", getTargetName(es))
			return nil, err
		}
		return nil, fmt.Errorf("failed to get target resource: %w", err)
	}

	return resource, nil
}

func (r *Reconciler) createNonSecretResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret, obj *unstructured.Unstructured) error {
	gvk := getTargetGVK(es)
	gvr, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{Kind: gvk.Kind, Group: gvk.Group, Version: gvk.Version})
	existing, err := r.DynamicClient.Resource(gvr).Namespace(es.Namespace).Get(ctx, getTargetName(es), metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to check if target resource exists: %w", err)
		}
	}

	if existing != nil {
		return fmt.Errorf("target resource with name %s already exists", getTargetName(es))
	}

	log.Info("creating target resource", "gvk", gvk.String(), "name", getTargetName(es))
	_, err = r.DynamicClient.Resource(gvr).Namespace(es.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

func (r *Reconciler) updateNonSecretResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret, existing *unstructured.Unstructured) error {
	gvk := getTargetGVK(es)
	gvr, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{Kind: gvk.Kind, Group: gvk.Group, Version: gvk.Version})

	// Update existing resource
	log.Info("updating target resource", "gvk", gvk.String(), "name", getTargetName(es))
	_, err := r.DynamicClient.Resource(gvr).Namespace(es.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

// deleteNonSecretResource deletes a non-Secret resource.
func (r *Reconciler) deleteNonSecretResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret) error {
	if !isNonSecretTarget(es) {
		return nil
	}

	gvk := getTargetGVK(es)
	gvr, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{Kind: gvk.Kind, Group: gvk.Group, Version: gvk.Version})

	log.Info("deleting target resource", "gvk", gvk.String(), "name", getTargetName(es))
	err := r.DynamicClient.Resource(gvr).Namespace(es.Namespace).Delete(ctx, getTargetName(es), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Deleted", fmt.Sprintf("Deleted %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

// ApplyTemplateToManifest renders templates for non-Secret resources and returns an unstructured object.
func (r *Reconciler) ApplyTemplateToManifest(ctx context.Context, es *esv1.ExternalSecret, dataMap map[string][]byte) (*unstructured.Unstructured, error) {
	gvk := getTargetGVK(es)

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(getTargetName(es))
	obj.SetNamespace(es.Namespace)

	labels := make(map[string]string)
	annotations := make(map[string]string)

	if es.Spec.Target.Template != nil {
		for k, v := range es.Spec.Target.Template.Metadata.Labels {
			labels[k] = v
		}
		for k, v := range es.Spec.Target.Template.Metadata.Annotations {
			annotations[k] = v
		}
	}

	labels[esv1.LabelManaged] = esv1.LabelManagedValue

	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)

	if es.Spec.Target.Template == nil {
		return r.createSimpleManifest(obj, dataMap)
	}

	return r.renderTemplatedManifest(ctx, es, obj, dataMap)
}

// createSimpleManifest creates a simple resource without templates (e.g., ConfigMap with data field).
func (r *Reconciler) createSimpleManifest(obj *unstructured.Unstructured, dataMap map[string][]byte) (*unstructured.Unstructured, error) {
	// For ConfigMaps and similar resources, put data in .data field
	if obj.GetKind() == "ConfigMap" {
		data := make(map[string]string)
		for k, v := range dataMap {
			data[k] = string(v)
		}
		obj.Object["data"] = data

		return obj, nil
	}

	// For other resources, put in spec.data or just data
	data := make(map[string]string)
	for k, v := range dataMap {
		data[k] = string(v)
	}
	if obj.Object["spec"] == nil {
		obj.Object["spec"] = make(map[string]any)
	}
	spec := obj.Object["spec"].(map[string]any)
	spec["data"] = data

	return obj, nil
}

// renderTemplatedManifest renders templates for a custom resource.
func (r *Reconciler) renderTemplatedManifest(ctx context.Context, es *esv1.ExternalSecret, obj *unstructured.Unstructured, dataMap map[string][]byte) (*unstructured.Unstructured, error) {
	execute, err := template.EngineForVersion(es.Spec.Target.Template.EngineVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get template engine: %w", err)
	}

	for _, tplFrom := range es.Spec.Target.Template.TemplateFrom {
		if tplFrom.Literal != nil {
			// Determine target path: ManifestTarget takes precedence over Target
			var targetPath string
			if tplFrom.ManifestTarget != nil {
				targetPath = *tplFrom.ManifestTarget
			} else {
				targetPath = string(tplFrom.Target)
			}

			rendered, err := r.renderTemplate(execute, *tplFrom.Literal, dataMap)
			if err != nil {
				return nil, fmt.Errorf("failed to render template: %w", err)
			}

			if err := r.applyToPath(obj, targetPath, rendered); err != nil {
				return nil, fmt.Errorf("failed to apply template to path %s: %w", targetPath, err)
			}
		}

		if tplFrom.ConfigMap != nil || tplFrom.Secret != nil {
			tempSecret := &v1.Secret{
				Data: make(map[string][]byte),
			}

			p := templating.Parser{
				Client:       r.Client,
				TargetSecret: tempSecret,
				DataMap:      dataMap,
				Exec:         execute,
			}

			if tplFrom.ConfigMap != nil {
				if err := p.MergeConfigMap(ctx, es.Namespace, tplFrom); err != nil {
					return nil, fmt.Errorf("failed to merge configmap template: %w", err)
				}
			}

			if tplFrom.Secret != nil {
				if err := p.MergeSecret(ctx, es.Namespace, tplFrom); err != nil {
					return nil, fmt.Errorf("failed to merge secret template: %w", err)
				}
			}

			// Determine target path: ManifestTarget takes precedence over Target
			var targetPath string
			if tplFrom.ManifestTarget != nil {
				targetPath = *tplFrom.ManifestTarget
			} else {
				targetPath = string(tplFrom.Target)
			}

			for k, v := range tempSecret.Data {
				if err := r.applyToPath(obj, targetPath+"."+k, string(v)); err != nil {
					return nil, fmt.Errorf("failed to apply data to path: %w", err)
				}
			}
		}
	}

	for key, tmpl := range es.Spec.Target.Template.Data {
		rendered, err := r.renderTemplate(execute, tmpl, dataMap)
		if err != nil {
			return nil, fmt.Errorf("failed to render template for key %s: %w", key, err)
		}

		if obj.GetKind() == "ConfigMap" {
			if obj.Object["data"] == nil {
				obj.Object["data"] = make(map[string]any)
			}
			data := obj.Object["data"].(map[string]any)
			data[key] = rendered
		} else {
			// Default to spec.data for custom resources
			if err := r.applyToPath(obj, "spec.data."+key, rendered); err != nil {
				return nil, fmt.Errorf("failed to apply template data: %w", err)
			}
		}
	}

	return obj, nil
}

// renderTemplate executes a template string with the provided data.
func (r *Reconciler) renderTemplate(execute template.ExecFunc, tmpl string, dataMap map[string][]byte) (string, error) {
	out := make(map[string][]byte)
	out["template"] = []byte(tmpl)
	tempSecret := &v1.Secret{
		Data: make(map[string][]byte),
	}

	err := execute(out, dataMap, esv1.TemplateScopeKeysAndValues, esv1.TemplateTargetData, tempSecret)
	if err != nil {
		return "", err
	}

	if rendered, ok := tempSecret.Data["template"]; ok {
		return string(rendered), nil
	}

	return "", fmt.Errorf("template execution did not produce output")
}

// applyToPath applies a value to a specific path in the unstructured object.
// Supports paths like "data", "spec", "spec.config", etc.
func (r *Reconciler) applyToPath(obj *unstructured.Unstructured, path string, value any) error {
	path = strings.ToLower(path)
	switch path {
	case "data":
		if obj.Object["data"] == nil {
			obj.Object["data"] = make(map[string]any)
		}
		if str, ok := value.(string); ok {
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(str), &parsed); err == nil {
				for k, v := range parsed {
					obj.Object["data"].(map[string]any)[k] = v
				}
				return nil
			}
		}
		obj.Object["data"] = value
		return nil

	case "spec":
		if str, ok := value.(string); ok {
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(str), &parsed); err == nil {
				obj.Object["spec"] = parsed
				return nil
			}
		}
		obj.Object["spec"] = value
		return nil

	case "annotations":
		if str, ok := value.(string); ok {
			var parsed map[string]string
			if err := json.Unmarshal([]byte(str), &parsed); err == nil {
				obj.SetAnnotations(parsed)
				return nil
			}
		}
		return fmt.Errorf("annotations must be a valid JSON object")

	case "labels":
		if str, ok := value.(string); ok {
			var parsed map[string]string
			if err := json.Unmarshal([]byte(str), &parsed); err == nil {
				obj.SetLabels(parsed)
				return nil
			}
		}
		return fmt.Errorf("labels must be a valid JSON object")
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid path: %s", path)
	}

	current := obj.Object
	for i := range len(parts) {
		part := parts[i]
		if current[part] == nil {
			current[part] = make(map[string]any)
		}
		var ok bool
		current, ok = current[part].(map[string]any)
		if !ok {
			return fmt.Errorf("path %s is not a map at segment %s", path, part)
		}
	}

	lastPart := parts[len(parts)-1]

	if str, ok := value.(string); ok {
		var parsed any
		if err := yaml.Unmarshal([]byte(str), &parsed); err == nil {
			current[lastPart] = parsed
			return nil
		}
	}

	current[lastPart] = value
	return nil
}
