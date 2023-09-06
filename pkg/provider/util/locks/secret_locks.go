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
package locks

import "sync"

// SecretLocks is a collection of locks for secrets to prevent lost update.
type SecretLocks struct {
	locks sync.Map
}

// TryLock tries to hold lock for a given secret and returns true if succeeded.
func (s *SecretLocks) TryLock(name string) (func(), bool) {
	lock, _ := s.locks.LoadOrStore(name, &sync.Mutex{})
	mu, _ := lock.(*sync.Mutex)
	return mu.Unlock, mu.TryLock()
}
