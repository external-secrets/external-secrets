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

package find

import (
	"fmt"
	"regexp"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type Matcher struct {
	re *regexp.Regexp
}

func New(findName esv1beta1.FindName) (*Matcher, error) {
	cmp, err := regexp.Compile(findName.RegExp)
	if err != nil {
		return nil, fmt.Errorf("could not compile find.name.regexp [%s]: %w", findName.RegExp, err)
	}
	return &Matcher{
		re: cmp,
	}, nil
}

func (m *Matcher) MatchName(name string) bool {
	return m.re.MatchString(name)
}
