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

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package scheduler provides a global scheduler for running periodic tasks.
package scheduler

import (
	"sync"
)

var (
	global *Scheduler
	once   sync.Once
)

// SetGlobal sets the global scheduler instance.
func SetGlobal(s Scheduler) {
	if s == nil {
		panic("scheduler: SetGlobal called with nil")
	}
	once.Do(func() {
		global = &s
	})
	if global != &s {
		panic("scheduler: SetGlobal called more than once")
	}
}

// Global returns the global scheduler instance.
func Global() Scheduler {
	if global == nil {
		panic("scheduler: Global called before SetGlobal")
	}
	return *global
}
