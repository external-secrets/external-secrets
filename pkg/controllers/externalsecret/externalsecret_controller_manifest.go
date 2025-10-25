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

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

	log.Info("Warning: Using non-Secret target. Make sure access policies and encryption are properly configured.",
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
	// TODO: Do pluralization on the discovery API instead. Mapper on the controller runtime.
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

	// Handle templateFrom entries
	for _, tplFrom := range es.Spec.Target.Template.TemplateFrom {
		targetPath := tplFrom.Target
		if targetPath == "" {
			targetPath = esv1.TemplateTargetData
		}

		if tplFrom.Literal != nil {
			// Execute template directly against the unstructured object
			out := make(map[string][]byte)
			out[*tplFrom.Literal] = []byte(*tplFrom.Literal)
			if err := execute(out, dataMap, esv1.TemplateScopeKeysAndValues, targetPath, obj); err != nil {
				return nil, fmt.Errorf("failed to execute literal template: %w", err)
			}
		}

		if tplFrom.ConfigMap != nil || tplFrom.Secret != nil {
			// Parser still uses v1.Secret, so collect data and apply via template engine to the end result.
			tempSecret := &v1.Secret{Data: make(map[string][]byte)}
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

			// apply collected data to the target object
			if err := execute(tempSecret.Data, dataMap, esv1.TemplateScopeValues, targetPath, obj); err != nil {
				return nil, fmt.Errorf("failed to apply merged templates to path %s: %w", targetPath, err)
			}
		}
	}

	// Handle template.data entries
	if len(es.Spec.Target.Template.Data) > 0 {
		tplMap := make(map[string][]byte)
		for k, v := range es.Spec.Target.Template.Data {
			tplMap[k] = []byte(v)
		}

		if err := execute(tplMap, dataMap, esv1.TemplateScopeValues, esv1.TemplateTargetData, obj); err != nil {
			return nil, fmt.Errorf("failed to execute template.data: %w", err)
		}
	}

	return obj, nil
}
