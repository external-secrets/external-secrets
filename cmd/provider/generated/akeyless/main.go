package main

import (
	provider "github.com/external-secrets/external-secrets-provider-akeyless"
	"github.com/external-secrets/external-secrets/pkg/remote/shell"
)

//go:generate ./generate.sh $GOFILE
func main() {
	p := &provider.Provider{}
	err := shell.RunServer(p)
	if err != nil {
		panic(err)
	}
}
