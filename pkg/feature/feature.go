//Copyright External Secrets Inc. All Rights Reserved

package feature

import (
	"github.com/spf13/pflag"
)

// Feature contains the CLI flags that a provider exposes to a user.
// A optional Initialize func is called once the flags have been parsed.
// A provider can use this to do late-initialization using the defined cli args.
type Feature struct {
	Flags      *pflag.FlagSet
	Initialize func()
}

var features = make([]Feature, 0)

// Features returns all registered features.
func Features() []Feature {
	return features
}

// Register registers a new feature.
func Register(f Feature) {
	features = append(features, f)
}
