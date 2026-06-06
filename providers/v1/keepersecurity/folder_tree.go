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

package keepersecurity

import (
	"strings"

	ksm "github.com/keeper-security/secrets-manager-go/core"
)

// folderNode is a single folder in the hierarchy.
type folderNode struct {
	name   string
	parent *folderNode
}

// folderTree maps folder UID -> node, for resolving a UID to its full path.
type folderTree struct {
	nodes map[string]*folderNode
}

// buildFolderTree builds a UID->node index with parent links from a flat list.
func buildFolderTree(folders []*ksm.KeeperFolder) *folderTree {
	t := &folderTree{nodes: make(map[string]*folderNode, len(folders))}
	for _, f := range folders {
		if f == nil {
			continue
		}
		t.nodes[f.FolderUid] = &folderNode{name: f.Name}
	}
	for _, f := range folders {
		if f == nil {
			continue
		}
		if n, ok := t.nodes[f.FolderUid]; ok {
			if p, ok := t.nodes[f.ParentUid]; ok {
				n.parent = p
			}
		}
	}
	return t
}

// pathOf returns the slash-joined folder path for a folder UID, e.g. "A/B/C".
// Unknown UIDs (record at the share root) return "".
func (t *folderTree) pathOf(uid string) string {
	n, ok := t.nodes[uid]
	if !ok {
		return ""
	}
	var parts []string
	for cur := n; cur != nil; cur = cur.parent {
		if cur.name != "" {
			parts = append([]string{cur.name}, parts...)
		}
	}
	return strings.Join(parts, "/")
}

// pathMatchesPrefix reports whether recordPath is at or under the requested path.
// Both are normalized (leading/trailing slashes trimmed). An empty want matches all.
func pathMatchesPrefix(recordPath, want string) bool {
	want = strings.Trim(want, "/")
	recordPath = strings.Trim(recordPath, "/")
	if want == "" {
		return true
	}
	return recordPath == want || strings.HasPrefix(recordPath, want+"/")
}
