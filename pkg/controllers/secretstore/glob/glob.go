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

Inspired from https://github.com/argoproj/argo-cd/tree/master/util/glob
*/

package glob

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/gobwas/glob"
)

func Match(log logr.Logger, pattern, text string) bool {
	compiledGlob, err := glob.Compile(pattern)
	if err != nil {
		log.V(1).Error(err, fmt.Sprintf("failed to compile pattern %s due to error", pattern))
		return false
	}
	return compiledGlob.Match(text)
}

func MatchStringInList(log logr.Logger, list []string, item string) bool {
	for _, ll := range list {
		if item == ll || Match(log, ll, item) {
			return true
		}
	}
	return false
}
