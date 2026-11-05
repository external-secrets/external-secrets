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
	"errors"
	"fmt"
	"maps"
	"reflect"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
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

	// Same check the admission webhook runs, applied before any provider retrieval,
	// rendering or write, so objects stored before the ban and ExternalSecrets admitted
	// without the webhook are refused here instead of reaching the generic write path.
	if err := esv1.ValidateManifestTarget(es); err != nil {
		return err
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

// errProtectedMetadata covers every way rendering can change metadata outside labels and
// annotations, including name and namespace. Naming the offending key would make the
// message depend on Go's randomized map iteration order when a template changes several.
var errProtectedMetadata = errors.New("rendered target changed protected metadata")

// normalizeAndValidateManifestCandidate ensures rendering changed only the declared target's
// content, labels and annotations. It also restores protected metadata that the template
// omitted, so it mutates target. The restore has to stay between the two loops: a changed
// value must be rejected before a missing one is filled back in.
func normalizeAndValidateManifestCandidate(es *esv1.ExternalSecret, target *unstructured.Unstructured, originalMetadata map[string]any) error {
	declaredGVK := getTargetGVK(es)
	renderedGVK := target.GroupVersionKind()

	if renderedGVK != declaredGVK {
		return fmt.Errorf(
			"rendered target GVK %q differs from declared target GVK %q",
			renderedGVK.String(),
			declaredGVK.String(),
		)
	}

	renderedMetadata, ok := target.Object["metadata"].(map[string]any)
	if !ok {
		return errProtectedMetadata
	}
	for key, rendered := range renderedMetadata {
		if key == "labels" || key == "annotations" {
			continue
		}
		original, ok := originalMetadata[key]
		if !ok || !reflect.DeepEqual(original, rendered) {
			return errProtectedMetadata
		}
	}
	// A whole-map template can omit server-managed metadata; restore only missing keys.
	for key, value := range originalMetadata {
		if key == "labels" || key == "annotations" {
			continue
		}
		if _, ok := renderedMetadata[key]; !ok {
			renderedMetadata[key] = value
		}
	}
	// Unreachable today: the loops above leave every protected key equal to the original,
	// and the original identity is the declared one. Kept as a guard on that invariant,
	// which is what stops a template from retargeting another namespace's object.
	if target.GetNamespace() != es.Namespace || target.GetName() != getTargetName(es) {
		return errors.New("rendered target namespace/name differs from declared target")
	}
	return nil
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

func (r *Reconciler) updateGenericResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret, existing *unstructured.Unstructured) error {
	gvk := getTargetGVK(es)

	log.Info("updating target resource", "gvk", gvk.String(), "name", getTargetName(es))
	err := r.Client.Update(ctx, existing)
	if err != nil {
		return fmt.Errorf("failed to update target resource: %w", err)
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

// applyTemplateToManifest renders templates for generic resources and returns an unstructured object.
// Existing metadata is retained for updates; Merge policies also retain existing content.
func (r *Reconciler) applyTemplateToManifest(ctx context.Context, es *esv1.ExternalSecret, dataMap map[string][]byte, existingObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	var obj *unstructured.Unstructured
	if existingObj != nil && (es.Spec.Target.CreationPolicy == esv1.CreatePolicyMerge ||
		es.Spec.Target.CreationPolicy == esv1.CreatePolicyCreateOrMerge) {
		obj = existingObj.DeepCopy()
	} else {
		gvk := getTargetGVK(es)
		obj = &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(getTargetName(es))
		obj.SetNamespace(es.Namespace)
		// Copy the metadata subtree only. DeepCopy would also copy the content this
		// branch is about to discard, which for a custom resource is most of the object.
		if existingObj != nil {
			if existingMetadata, ok := existingObj.Object["metadata"].(map[string]any); ok {
				obj.Object["metadata"] = runtime.DeepCopyJSON(existingMetadata)
			}
		}
		switch gvk.Kind {
		case "ConfigMap", "Secret":
			obj.Object["data"] = map[string]any{}
		default:
			obj.Object["spec"] = map[string]any{}
		}
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Keep the original metadata for validation; only labels and annotations are templatable.
	existingMetadata := obj.DeepCopy().Object["metadata"].(map[string]any)

	srcLabels, srcAnnotations := es.ObjectMeta.Labels, es.ObjectMeta.Annotations
	if es.Spec.Target.Template != nil {
		srcLabels = es.Spec.Target.Template.Metadata.Labels
		srcAnnotations = es.Spec.Target.Template.Metadata.Annotations
	}
	maps.Copy(labels, srcLabels)
	maps.Copy(annotations, srcAnnotations)

	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	labels = maps.Clone(labels)
	annotations = maps.Clone(annotations)

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

	// A template may replace metadata maps; preserve existing keys unless the template overrides them.
	maps.Copy(labels, result.GetLabels())
	result.SetLabels(labels)
	maps.Copy(annotations, result.GetAnnotations())
	result.SetAnnotations(annotations)

	if err := normalizeAndValidateManifestCandidate(es, result, existingMetadata); err != nil {
		return nil, err
	}
	if err := r.applyOwnership(es, result); err != nil {
		return nil, err
	}

	labels = result.GetLabels()
	labels[esv1.LabelManaged] = esv1.LabelManagedValue
	result.SetLabels(labels)
	hash, err := genericTargetContentHash(result)
	if err != nil {
		return nil, fmt.Errorf("failed to hash target %q content: %w", es.Spec.Target.Name, err)
	}
	annotations[esv1.AnnotationDataHash] = hash
	result.SetAnnotations(annotations)
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
	log := r.Log.WithValues("ExternalSecret", client.ObjectKeyFromObject(es))
	log.V(1).Info("template render started")
	defer log.V(1).Info("template render finished")

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
