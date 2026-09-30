/*
Copyright © The ESO Authors

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
	"maps"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/template"
)

// isGenericTarget checks if the ExternalSecret targets a generic resource.
func isGenericTarget(es *esv1.ExternalSecret) bool {
	return es.Spec.Target.Manifest != nil
}

// validateGenericTarget validates that generic targets are properly configured.
func (r *Reconciler) validateGenericTarget(log logr.Logger, es *esv1.ExternalSecret) error {
	if !r.AllowGenericTargets {
		return fmt.Errorf("generic targets are disabled. Enable with --unsafe-allow-generic-targets flag")
	}

	manifest := es.Spec.Target.Manifest
	if manifest.APIVersion == "" {
		return fmt.Errorf("target.manifest.apiVersion is required")
	}
	if manifest.Kind == "" {
		return fmt.Errorf("target.manifest.kind is required")
	}

	log.Info("Warning: Using generic target. Make sure access policies and encryption are properly configured.",
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

// getGenericResource retrieves a generic resource using the controller-runtime client.
func (r *Reconciler) getGenericResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret) (*unstructured.Unstructured, error) {
	gvk := getTargetGVK(es)

	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)

	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: es.Namespace,
		Name:      getTargetName(es),
	}, resource)

	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("target resource does not exist", "gvk", gvk.String(), "name", getTargetName(es))
			return nil, err
		}
		return nil, fmt.Errorf("failed to get target resource: %w", err)
	}

	return resource, nil
}

func (r *Reconciler) createGenericResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret, obj *unstructured.Unstructured) error {
	gvk := getTargetGVK(es)

	// Check if resource already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(gvk)
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: es.Namespace,
		Name:      getTargetName(es),
	}, existing)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to check if target resource exists: %w", err)
		}
	} else {
		return fmt.Errorf("target resource with name %s already exists", getTargetName(es))
	}

	log.Info("creating target resource", "gvk", gvk.String(), "name", getTargetName(es))
	err = r.Client.Create(ctx, obj)
	if err != nil {
		return fmt.Errorf("failed to create target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

func (r *Reconciler) patchGenericResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret, existing, patch *unstructured.Unstructured) error {
	gvk := getTargetGVK(es)

	patchBody, err := strategicMergePatchBody(patch)
	if err != nil {
		return fmt.Errorf("failed to build strategic merge patch: %w", err)
	}

	log.Info("patching target resource", "gvk", gvk.String(), "name", getTargetName(es))
	err = r.Client.Patch(ctx, existing, client.RawPatch(types.StrategicMergePatchType, patchBody))
	if err != nil {
		return fmt.Errorf("failed to patch target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

// deleteGenericResource deletes a generic resource.
func (r *Reconciler) deleteGenericResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret) error {
	if !r.AllowGenericTargets || !isGenericTarget(es) {
		return nil
	}

	gvk := getTargetGVK(es)

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetNamespace(es.Namespace)
	obj.SetName(getTargetName(es))

	log.Info("deleting target resource", "gvk", gvk.String(), "name", getTargetName(es))
	err := r.Client.Delete(ctx, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Deleted", fmt.Sprintf("Deleted %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

// applyTemplateToManifest renders an unstructured object from the generic resource template. It returns a partial
// unstructured object constructed only from the template data, suitable for strategic merge patching.
func (r *Reconciler) applyTemplateToManifest(ctx context.Context, es *esv1.ExternalSecret, dataMap map[string][]byte) (*unstructured.Unstructured, error) {
	gvk := getTargetGVK(es)
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(getTargetName(es))
	obj.SetNamespace(es.Namespace)
	switch gvk.Kind {
	case "ConfigMap", "Secret":
		obj.Object["data"] = map[string]any{}
	default:
		obj.Object["spec"] = map[string]any{}
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	srcLabels, srcAnnotations := es.ObjectMeta.Labels, es.ObjectMeta.Annotations
	if es.Spec.Target.Template != nil {
		srcLabels = es.Spec.Target.Template.Metadata.Labels
		srcAnnotations = es.Spec.Target.Template.Metadata.Annotations
	}
	maps.Copy(labels, srcLabels)
	maps.Copy(annotations, srcAnnotations)

	labels[esv1.LabelManaged] = esv1.LabelManagedValue

	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)

	var result *unstructured.Unstructured
	var err error
	if es.Spec.Target.Template == nil {
		result = r.createSimpleManifest(obj, dataMap)
	} else {
		result, err = r.renderTemplatedManifest(ctx, es, obj, dataMap)
	}
	if err != nil {
		return nil, err
	}

	ann := result.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}

	hash, err := genericTargetManagedContentHash(es, result)
	if err != nil {
		return nil, fmt.Errorf("failed to hash target %q content: %w", es.Spec.Target.Name, err)
	}

	ann[esv1.AnnotationDataHash] = hash
	result.SetAnnotations(ann)

	if err := r.applyOwnership(es, result); err != nil {
		return nil, err
	}

	return result, nil
}

// createSimpleManifest creates a simple resource without templates (e.g., ConfigMap with data field).
func (r *Reconciler) createSimpleManifest(obj *unstructured.Unstructured, dataMap map[string][]byte) *unstructured.Unstructured {
	// For ConfigMaps and similar resources, put data in .data field
	if obj.GetKind() == "ConfigMap" {
		data := make(map[string]string)
		for k, v := range dataMap {
			data[k] = string(v)
		}
		obj.Object["data"] = data

		return obj
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

	return obj
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
			if err := execute(out, dataMap, esv1.TemplateScopeKeysAndValues, targetPath, obj, tplFrom.ValuesDecodingStrategy); err != nil {
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
			if err := execute(tempSecret.Data, dataMap, esv1.TemplateScopeValues, targetPath, obj, esv1.ExternalSecretDecodeNone); err != nil {
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

		if err := execute(tplMap, dataMap, esv1.TemplateScopeValues, esv1.TemplateTargetData, obj, esv1.ExternalSecretDecodeNone); err != nil {
			return nil, fmt.Errorf("failed to execute template.data: %w", err)
		}
	}

	return obj, nil
}

// strategicMergePatchBody builds a partial strategic merge patch payload from the templated object. Constructing the
// patch payload with only metadata, spec, or data fields allows the API server to merge changes into the live object
// using its schema (e.g., x-kubernetes-list-type / x-kubernetes-list-map-keys), preserving any unrelated fields.
func strategicMergePatchBody(patch *unstructured.Unstructured) ([]byte, error) {
	body := map[string]any{}
	if metadata, ok := patch.Object["metadata"].(map[string]any); ok && len(metadata) > 0 {
		body["metadata"] = metadata
	}
	if spec, ok := patch.Object["spec"].(map[string]any); ok && len(spec) > 0 {
		body["spec"] = spec
	}
	if data, ok := patch.Object["data"].(map[string]any); ok && len(data) > 0 {
		body["data"] = data
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("patch payload is empty")
	}
	return json.Marshal(body)
}

// templateTargetPaths returns the target paths (TemplateFrom targets and Template.Data) that the ExternalSecret
// template writes to. These paths are used for calculating the managed-content hash.
func templateTargetPaths(es *esv1.ExternalSecret) []string {
	if es.Spec.Target.Template == nil {
		return nil
	}

	paths := make([]string, 0, len(es.Spec.Target.Template.TemplateFrom)+1)
	for _, tmplFrom := range es.Spec.Target.Template.TemplateFrom {
		targetPath := tmplFrom.Target
		if targetPath == "" {
			targetPath = esv1.TemplateTargetData
		}
		paths = append(paths, normalizeTemplateTargetPath(targetPath))
	}

	if len(es.Spec.Target.Template.Data) > 0 {
		paths = append(paths, esv1.TemplateTargetData)
	}

	return paths
}

// normalizeTemplateTargetPath maps target shorthands (like "annotations" and "labels") to their full metadata paths
// ("metadata.annotations", "metadata.labels") for accurate hash calculations. Any other path is returned unchanged.
func normalizeTemplateTargetPath(target string) string {
	switch strings.ToLower(target) {
	case "annotations":
		return "metadata.annotations"
	case "labels":
		return "metadata.labels"
	default:
		return target
	}
}

// extractTemplateTargetContent builds a nested structure with only the values at the configured target paths from the
// given object. Paths that don't exist are silently ignored so a clean hash can still be calculated.
func extractTemplateTargetContent(obj *unstructured.Unstructured, paths []string) (map[string]any, error) {
	content := make(map[string]any, len(paths))
	for _, path := range paths {
		parts := strings.Split(path, ".")
		val, found, err := unstructured.NestedFieldCopy(obj.Object, parts...)
		if err != nil {
			return nil, fmt.Errorf("failed to read path %q: %w", path, err)
		}
		if !found {
			continue
		}
		if err := unstructured.SetNestedField(content, val, parts...); err != nil {
			return nil, fmt.Errorf("failed to store path %q for hashing: %w", path, err)
		}
	}
	return content, nil
}

// genericTargetManagedContentHash calculates a hash of only the ExternalSecret template target paths, not the entire
// spec or data map. Hashing the full parent field would include unmanaged child fields from the fetched object, which
// would make AnnotationDataHash never match and force a refresh every reconcile.
func genericTargetManagedContentHash(es *esv1.ExternalSecret, obj *unstructured.Unstructured) (string, error) {
	paths := templateTargetPaths(es)
	if len(paths) == 0 {
		return genericTargetContentHash(obj)
	}
	content, err := extractTemplateTargetContent(obj, paths)
	if err != nil {
		return "", err
	}
	if len(content) == 0 {
		return genericTargetContentHash(obj)
	}
	return esutils.ObjectHash(content), nil
}
