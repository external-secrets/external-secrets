//Copyright External Secrets Inc. All Rights Reserved

package clock

import "time"

type FakeClock struct {
	now time.Time
}

func NewFakeClock() *FakeClock {
	return &FakeClock{time.Time{}}
}

func (c *FakeClock) CurrentTime() time.Time {
	return c.now
}

func (c *FakeClock) AddDuration(duration time.Duration) {
	c.now = c.now.Add(duration)
}
