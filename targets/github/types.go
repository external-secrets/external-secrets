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

// Copyright External Secrets Inc. 2025
// All Rights Reserved

package github

import "strings"

type pathFilter struct {
	exact    map[string]struct{} // exact file matches: "a/b/c.txt"
	prefixes []string            // directory prefixes: "a/b/" (must end with '/')
}

func newPathFilter(paths []string) *pathFilter {
	filter := &pathFilter{
		exact:    make(map[string]struct{}),
		prefixes: make([]string, 0, len(paths)),
	}

	seenPrefix := make(map[string]struct{})

	for _, path := range paths {
		path = strings.TrimSpace(path)
		path = strings.TrimPrefix(path, "/")
		if path == "" {
			continue
		}

		filter.exact[path] = struct{}{}

		pref := path
		if !strings.HasSuffix(pref, "/") {
			pref += "/"
		}
		if _, duplicate := seenPrefix[pref]; !duplicate {
			filter.prefixes = append(filter.prefixes, pref)
			seenPrefix[pref] = struct{}{}
		}
	}

	return filter
}

func (f *pathFilter) allow(path string) bool {
	// No filters -> allow all
	if len(f.exact) == 0 && len(f.prefixes) == 0 {
		return true
	}
	if _, ok := f.exact[path]; ok {
		return true
	}
	for _, prefix := range f.prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
