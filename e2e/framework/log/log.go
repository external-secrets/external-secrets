//Copyright External Secrets Inc. All Rights Reserved

package log

import (
	"github.com/onsi/ginkgo/v2"
)

// Logf logs the format string to ginkgo stdout.
func Logf(format string, args ...any) {
	ginkgo.GinkgoWriter.Printf(format+"\n", args...)
}
