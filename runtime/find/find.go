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

// Package find provides utilities for matching names against regular expressions.
package find

import (
	"fmt"
	"regexp"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// Matcher represents a pattern matcher that uses regular expressions to match names.
type Matcher struct {
	re *regexp.Regexp
}

// New creates a new Matcher using the provided FindName configuration.
func New(findName esv1.FindName) (*Matcher, error) {
	cmp, err := regexp.Compile(findName.RegExp)
	if err != nil {
		return nil, fmt.Errorf("could not compile find.name.regexp [%s]: %w", findName.RegExp, err)
	}
	return &Matcher{
		re: cmp,
	}, nil
}

// MatchName checks if the given name matches the configured regular expression pattern.
func (m *Matcher) MatchName(name string) bool {
	return m.re.MatchString(name)
}
