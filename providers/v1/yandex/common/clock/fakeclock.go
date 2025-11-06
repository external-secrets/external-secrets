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

package clock

import "time"

// FakeClock provides a clock implementation with manually controlled time for testing.
type FakeClock struct {
	now time.Time
}

// NewFakeClock creates a new FakeClock instance initialized to current time.
func NewFakeClock() *FakeClock {
	return &FakeClock{
		now: time.Now(),
	}
}

// CurrentTime returns the current fake time.
func (c *FakeClock) CurrentTime() time.Time {
	return c.now
}

// AddDuration advances the fake clock by the specified duration.
func (c *FakeClock) AddDuration(duration time.Duration) {
	c.now = c.now.Add(duration)
}
