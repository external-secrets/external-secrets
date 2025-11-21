// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "workflows.external-secrets.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	// WorkflowKind is the kind name for Workflow resources.
	WorkflowKind = reflect.TypeOf(Workflow{}).Name()
	// WorkflowGroupKind is the group kind for Workflow resources.
	WorkflowGroupKind = schema.GroupKind{Group: Group, Kind: WorkflowKind}.String()
	// WorkflowKindAPIVersion is the API version for Workflow resources.
	WorkflowKindAPIVersion = WorkflowKind + "." + SchemeGroupVersion.String()
	// WorkflowGroupVersionKind is the group version kind for Workflow resources.
	WorkflowGroupVersionKind = SchemeGroupVersion.WithKind(WorkflowKind)

	// WorkflowTemplateKind is the kind name for WorkflowTemplate resources.
	WorkflowTemplateKind = reflect.TypeOf(WorkflowTemplate{}).Name()
	// WorkflowTemplateGroupKind is the group kind for WorkflowTemplate resources.
	WorkflowTemplateGroupKind = schema.GroupKind{Group: Group, Kind: WorkflowTemplateKind}.String()
	// WorkflowTemplateKindAPIVersion is the API version for WorkflowTemplate resources.
	WorkflowTemplateKindAPIVersion = WorkflowTemplateKind + "." + SchemeGroupVersion.String()
	// WorkflowTemplateGroupVersionKind is the group version kind for WorkflowTemplate resources.
	WorkflowTemplateGroupVersionKind = SchemeGroupVersion.WithKind(WorkflowTemplateKind)

	// WorkflowRunKind is the kind name for WorkflowRun resources.
	WorkflowRunKind = reflect.TypeOf(WorkflowRun{}).Name()
	// WorkflowRunGroupKind is the group kind for WorkflowRun resources.
	WorkflowRunGroupKind = schema.GroupKind{Group: Group, Kind: WorkflowRunKind}.String()
	// WorkflowRunKindAPIVersion is the API version for WorkflowRun resources.
	WorkflowRunKindAPIVersion = WorkflowRunKind + "." + SchemeGroupVersion.String()
	// WorkflowRunGroupVersionKind is the group version kind for WorkflowRun resources.
	WorkflowRunGroupVersionKind = SchemeGroupVersion.WithKind(WorkflowRunKind)
)

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
	SchemeBuilder.Register(&WorkflowTemplate{}, &WorkflowTemplateList{})
	SchemeBuilder.Register(&WorkflowRun{}, &WorkflowRunList{})
}
