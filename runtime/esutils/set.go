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

package esutils

// Set is a generic collection of unique values.
type Set[T comparable] struct {
	data map[T]struct{}
}

// NewSet creates and returns an empty Set based on the comparable type T.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		data: make(map[T]struct{}),
	}
}

// Add inserts v into the set.
func (s *Set[T]) Add(v T) {
	s.data[v] = struct{}{}
}

// Remove deletes v from the set.
func (s *Set[T]) Remove(v T) {
	delete(s.data, v)
}

// Contains reports whether v is present in the set.
func (s *Set[T]) Contains(v T) bool {
	_, found := s.data[v]
	return found
}

// Len returns the number of elements in the set.
func (s *Set[T]) Len() int {
	return len(s.data)
}

// ToSlice returns the set elements as a slice. The order of elements is not guaranteed.
func (s *Set[T]) ToSlice() []T {
	keys := make([]T, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

// Union returns a new Set containing all elements from s and other.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for key := range s.data {
		result.Add(key)
	}
	for key := range other.data {
		result.Add(key)
	}
	return result
}

// Intersection returns a new Set containing elements present in both
// s and other.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for key := range s.data {
		if other.Contains(key) {
			result.Add(key)
		}
	}
	return result
}

// Difference returns a new Set containing elements present in s but
// not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for key := range s.data {
		if !other.Contains(key) {
			result.Add(key)
		}
	}
	return result
}
