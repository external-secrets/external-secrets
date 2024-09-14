//Copyright External Secrets Inc. All Rights Reserved

package clock

import (
	"time"
)

type Clock interface {
	CurrentTime() time.Time
}
