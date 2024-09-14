//Copyright External Secrets Inc. All Rights Reserved

package clock

import "time"

type RealClock struct {
}

func NewRealClock() *RealClock {
	return &RealClock{}
}

func (c *RealClock) CurrentTime() time.Time {
	return time.Now()
}
