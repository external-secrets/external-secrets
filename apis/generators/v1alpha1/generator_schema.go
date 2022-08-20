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
	"fmt"
	"sync"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

var builder map[string]Generator
var buildlock sync.RWMutex

func init() {
	builder = make(map[string]Generator)
}

// Register a generator type. Register panics if a
// backend with the same store is already registered.
func Register(kind string, g Generator) {
	buildlock.Lock()
	defer buildlock.Unlock()
	_, exists := builder[kind]
	if exists {
		panic(fmt.Sprintf("kind %q already registered", kind))
	}

	builder[kind] = g
}

// ForceRegister adds to store schema, overwriting a store if
// already registered. Should only be used for testing.
func ForceRegister(kind string, g Generator) {
	buildlock.Lock()
	builder[kind] = g
	buildlock.Unlock()
}

// GetGeneratorByName returns the provider implementation by name.
func GetGeneratorByName(kind string) (Generator, bool) {
	buildlock.RLock()
	f, ok := builder[kind]
	buildlock.RUnlock()
	return f, ok
}

// GetGenerator returns the provider from the generic store.
func GetGenerator(obj *apiextensions.JSON) (Generator, error) {
	type unknownGenerator struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`
	}
	var res unknownGenerator
	err := json.Unmarshal(obj.Raw, &res)
	if err != nil {
		return nil, err
	}
	buildlock.RLock()
	defer buildlock.RUnlock()
	gen, ok := builder[res.Kind]
	if !ok {
		return nil, fmt.Errorf("failed to find registered generator for: %s", string(obj.Raw))
	}
	return gen, nil
}
