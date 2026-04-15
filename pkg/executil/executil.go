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

package executil

import (
	"fmt"
	"os/exec"

	"golang.org/x/sys/execabs"
)

// Command resolves an executable to an absolute path before constructing the command.
func Command(name string, args ...string) (*exec.Cmd, error) {
	path, err := execabs.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("find executable %q: %w", name, err)
	}

	return execabs.Command(path, args...), nil
}
