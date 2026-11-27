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
	"fmt"
	"maps"
	"slices"
	"strings"

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
	if err := esv1.ValidateManifestTarget(&es.Spec); err != nil {
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

// templatableMetadata are the only target metadata fields a template may contribute. Every
// other field belongs either to the live object or to the ExternalSecret's own declaration,
// and addRenderedContent reports any the template tried to set instead of applying them.
var templatableMetadata = []string{"labels", "annotations"}

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
	// Claim the fields we write. Without a field owner every ExternalSecret's writes land
	// under one shared manager, and dropStaleOwnedMetadata could not tell the keys this
	// ExternalSecret set from another ExternalSecret's or another controller's.
	err = r.Client.Create(ctx, obj, client.FieldOwner(fqdnFor(es.Name)))
	if err != nil {
		return fmt.Errorf("failed to create target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

func (r *Reconciler) updateGenericResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret, existing *unstructured.Unstructured) error {
	gvk := getTargetGVK(es)

	log.Info("updating target resource", "gvk", gvk.String(), "name", getTargetName(es))
	err := r.Client.Update(ctx, existing, client.FieldOwner(fqdnFor(es.Name)))
	if err != nil {
		return fmt.Errorf("failed to update target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

// deleteGenericResource deletes the generic target, but only if this ExternalSecret controls
// it. It returns ErrTargetNotControlled when the live target belongs to someone else, so a
// caller can tell a refusal apart from an API failure.
func (r *Reconciler) deleteGenericResource(ctx context.Context, log logr.Logger, es *esv1.ExternalSecret) error {
	if !r.AllowGenericTargets || !isGenericTarget(es) {
		return nil
	}

	gvk := getTargetGVK(es)

	// Fetch the target instead of deleting an object built from the declared name alone: the
	// controller owner reference that authorizes this deletion only exists on the live object.
	existing, err := r.getGenericResource(ctx, log, es)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if !isControlledByExternalSecret(es, existing) {
		return fmt.Errorf("%w: %s %s", ErrTargetNotControlled, gvk.Kind, getTargetName(es))
	}

	log.Info("deleting target resource", "gvk", gvk.String(), "name", getTargetName(es))
	err = r.deleteCheckedTarget(ctx, existing)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete target resource: %w", err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, "Deleted", fmt.Sprintf("Deleted %s %s", gvk.Kind, getTargetName(es)))
	return nil
}

// applyTemplateToManifest builds the target from three sources, each the only authority for
// its own part: the ExternalSecret declares the identity and the labels and annotations it
// owns, the live object keeps the rest of its metadata, and rendering supplies the content.
func (r *Reconciler) applyTemplateToManifest(ctx context.Context, es *esv1.ExternalSecret, dataMap map[string][]byte, existingObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Rendering runs on a sandbox that carries the declared kind and the content, and no
	// metadata at all, so every metadata field that comes back was written by the template.
	// That is what lets addRenderedContent take the fields policy allows and name the rest
	// without comparing anything against the live object.
	rendered, err := r.render(ctx, es, dataMap, existingObj)
	if err != nil {
		return nil, err
	}

	target := liveOrNewTarget(es, existingObj)
	if err := dropStaleOwnedMetadata(es, target); err != nil {
		return nil, err
	}
	addDeclaredMetadata(es, target)
	ignored := addRenderedContent(rendered, target)

	if err := r.applyOwnership(es, target); err != nil {
		return nil, err
	}
	if err := addManagedMarkers(target); err != nil {
		return nil, fmt.Errorf("failed to hash target %q content: %w", es.Spec.Target.Name, err)
	}

	if len(ignored) > 0 {
		r.recorder.Eventf(es, v1.EventTypeWarning, "TemplatedMetadataIgnored",
			"template wrote target metadata outside labels and annotations, ignored: %s",
			strings.Join(ignored, ", "))
	}
	return target, nil
}

// render produces the target's content in isolation. The object handed to the template carries
// the declared kind and, for the Merge policies, every live field to merge into, but no
// metadata: the caller reads back only the content and the templatable metadata fields, so
// anything else a template writes has nowhere to land.
func (r *Reconciler) render(ctx context.Context, es *esv1.ExternalSecret, dataMap map[string][]byte, existingObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	sandbox := &unstructured.Unstructured{Object: map[string]any{}}

	if existingObj != nil && (es.Spec.Target.CreationPolicy == esv1.CreatePolicyMerge ||
		es.Spec.Target.CreationPolicy == esv1.CreatePolicyCreateOrMerge) {
		// A Merge only changes what it renders, and the target is sent as a full update, so
		// every live field has to come along (a ConfigMap's binaryData, say), not only the
		// content field. The metadata is the one part left out: the sandbox must not carry it.
		for name, value := range existingObj.Object {
			if name != "metadata" {
				sandbox.Object[name] = runtime.DeepCopyJSONValue(value)
			}
		}
	}

	// Rendering writes into the content field, so it has to be a map whatever the live object held.
	field := contentField(es)
	if _, ok := sandbox.Object[field].(map[string]any); !ok {
		sandbox.Object[field] = map[string]any{}
	}
	sandbox.SetGroupVersionKind(getTargetGVK(es))

	if es.Spec.Target.Template == nil {
		return r.createSimpleManifest(sandbox, dataMap), nil
	}
	return r.renderTemplatedManifest(ctx, es, sandbox, dataMap)
}

// contentField is the field rendering writes the secret data into. A "Secret" here is a Secret
// kind in a custom API group: validateGenericTarget rejects the core one before rendering.
func contentField(es *esv1.ExternalSecret) string {
	switch getTargetGVK(es).Kind {
	case "ConfigMap", "Secret":
		return "data"
	default:
		return "spec"
	}
}

// liveOrNewTarget returns the target as it stands before rendering contributes anything: the
// live object's metadata under the declared identity, or just the identity when the target
// does not exist yet.
func liveOrNewTarget(es *esv1.ExternalSecret, existingObj *unstructured.Unstructured) *unstructured.Unstructured {
	target := &unstructured.Unstructured{Object: map[string]any{}}
	if existingObj != nil {
		// Copies the metadata subtree only. DeepCopy would also copy the content, which
		// rendering is the sole authority for and which for a custom resource is most of the
		// object.
		if metadata, ok := existingObj.Object["metadata"].(map[string]any); ok {
			target.Object["metadata"] = runtime.DeepCopyJSON(metadata)
		}
	}

	// Written last so the declared identity wins over anything the live object carried.
	target.SetGroupVersionKind(getTargetGVK(es))
	target.SetName(getTargetName(es))
	target.SetNamespace(es.Namespace)
	return target
}

// dropStaleOwnedMetadata removes the labels and annotations this ExternalSecret set on the
// target before and no longer declares. Without it, carrying the live object's metadata over
// would make its own keys un-removable: drop one from the template and the live copy puts it
// straight back. This mirrors setMetadata on the Secret lane.
//
// Targets written before ESO declared a field owner have their keys recorded under the
// client's default manager, so on the first sync after upgrade a key this ExternalSecret has
// since stopped rendering survives once. Everything written after is attributed.
func dropStaleOwnedMetadata(es *esv1.ExternalSecret, target *unstructured.Unstructured) error {
	labels, annotations := target.GetLabels(), target.GetAnnotations()
	if err := templating.ClearOwnedMetadata(target, es.Name, labels, annotations); err != nil {
		return err
	}

	target.SetLabels(labels)
	target.SetAnnotations(annotations)
	return nil
}

// addDeclaredMetadata merges in the labels and annotations the ExternalSecret declares, key by
// key, so that another controller's keys on the live object survive. A target template's
// metadata block replaces the ExternalSecret's own labels and annotations as the source rather
// than adding to them.
func addDeclaredMetadata(es *esv1.ExternalSecret, target *unstructured.Unstructured) {
	labels, annotations := es.ObjectMeta.Labels, es.ObjectMeta.Annotations
	if es.Spec.Target.Template != nil {
		labels = es.Spec.Target.Template.Metadata.Labels
		annotations = es.Spec.Target.Template.Metadata.Annotations
	}

	target.SetLabels(mergedInto(target.GetLabels(), labels))
	target.SetAnnotations(mergedInto(target.GetAnnotations(), annotations))
}

// addRenderedContent copies the content and the templatable metadata fields of a rendered
// object onto the target, and returns the names of the metadata fields it refused, sorted so
// the report does not depend on Go's map iteration order. The content is every top-level field
// the template rendered other than apiVersion, kind and metadata, not only spec or data.
// Rendering is the sole authority for the content; for labels and annotations it wins key by
// key over what was declared.
func addRenderedContent(rendered, target *unstructured.Unstructured) []string {
	for field, value := range rendered.Object {
		// The identity is the ExternalSecret's to declare, and metadata is handled below.
		if field != "metadata" && field != "apiVersion" && field != "kind" {
			target.Object[field] = value
		}
	}

	target.SetLabels(mergedInto(target.GetLabels(), rendered.GetLabels()))
	target.SetAnnotations(mergedInto(target.GetAnnotations(), rendered.GetAnnotations()))

	renderedMetadata, _ := rendered.Object["metadata"].(map[string]any)
	ignored := make([]string, 0, len(renderedMetadata))
	for field := range renderedMetadata {
		if !slices.Contains(templatableMetadata, field) {
			ignored = append(ignored, field)
		}
	}
	slices.Sort(ignored)
	return ignored
}

// addManagedMarkers stamps the label marking the target as ESO-managed and the annotation
// recording its content hash. It runs last: applyOwnership may add labels of its own, and the
// hash has to cover the final content.
func addManagedMarkers(target *unstructured.Unstructured) error {
	target.SetLabels(mergedInto(target.GetLabels(), map[string]string{esv1.LabelManaged: esv1.LabelManagedValue}))

	hash, err := genericTargetContentHash(target)
	if err != nil {
		return err
	}

	target.SetAnnotations(mergedInto(target.GetAnnotations(), map[string]string{esv1.AnnotationDataHash: hash}))
	return nil
}

// mergedInto returns base with over's entries written over it, key by key. Either may be nil,
// and the result never is, so the unstructured setters always have a map to write.
func mergedInto(base, over map[string]string) map[string]string {
	merged := map[string]string{}
	maps.Copy(merged, base)
	maps.Copy(merged, over)
	return merged
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
