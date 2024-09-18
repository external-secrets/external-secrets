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

package feature

import (
	"github.com/spf13/pflag"
)

// Feature contains the CLI flags that a provider exposes to a user.
// A optional Initialize func is called once the flags have been parsed.
// A provider can use this to do late-initialization using the defined cli args.
type Feature struct {
	Flags      *pflag.FlagSet
	Initialize func()
}

var features = make([]Feature, 0)

// Features returns all registered features.
func Features() []Feature {
	return features
}

// Register registers a new feature.
func Register(f Feature) {
	features = append(features, f)
}
