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
	"errors"
	"fmt"
	"strings"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

var refBuilder map[gvk]client.Object
var refBuildlock sync.RWMutex

func init() {
	refBuilder = make(map[gvk]client.Object)
}

type gvk struct {
	apiGroup   string
	apiVersion string
	kind       string
}

func newGroupVersionKind(ref esmeta.ProviderRef) gvk {
	group := strings.Split(ref.APIVersion, "/")[0]
	version := strings.Split(ref.APIVersion, "/")[1]
	return gvk{
		apiGroup:   group,
		apiVersion: version,
		kind:       ref.Kind,
	}
}
func RefRegister(s client.Object, provider esmeta.ProviderRef) {
	refBuildlock.Lock()
	gvk := newGroupVersionKind(provider)
	defer refBuildlock.Unlock()
	_, exists := refBuilder[gvk]
	if exists {
		panic(fmt.Sprintf("provider %q already registered", gvk))
	}

	refBuilder[gvk] = s
}

// ForceRegister adds to store schema, overwriting a store if
// already registered. Should only be used for testing.
func RefForceRegister(s client.Object, ref esmeta.ProviderRef) {
	gvk := newGroupVersionKind(ref)
	refBuildlock.Lock()
	refBuilder[gvk] = s
	refBuildlock.Unlock()
}

// GetProvider returns the provider from the generic store.
func GetManifestByKind(ref *esmeta.ProviderRef) (client.Object, error) {
	if ref == nil {
		return nil, errors.New("provider ref cannot be nil")
	}
	gvk := newGroupVersionKind(*ref)
	refBuildlock.RLock()
	f, ok := refBuilder[gvk]
	refBuildlock.RUnlock()

	if !ok {
		return nil, fmt.Errorf("failed to find registered object backend for kind: %s", gvk)
	}
	return f, nil
}
