package main

import (
	webhook "github.com/external-secrets/external-secrets-provider-webhook"
	pshell "github.com/external-secrets/external-secrets/pkg/remote/shell"
)

func main() {
	p := &webhook.Provider{}
	err := pshell.RunServer(p)
	if err != nil {
		panic(err)
	}
}
