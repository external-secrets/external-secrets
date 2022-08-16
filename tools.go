//go:build tools
// +build tools

package tools

import (
	_ "github.com/ahmetb/gen-crd-api-reference-docs"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
