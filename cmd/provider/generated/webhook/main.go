package main

import (
	"context"
	"fmt"

	provider "github.com/external-secrets/external-secrets-provider-webhook"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	shell "github.com/external-secrets/external-secrets/pkg/provider-shell"
)

//go:generate ./generate.sh $GOFILE
func main() {
	p := provider.Provider{}
	fmt.Printf("provider cap: %#v\n", p.Capabilities())
	pc, err := p.NewClient(context.Background(), &v1beta1.SecretStore{
		Spec: v1beta1.SecretStoreSpec{
			Provider: &v1beta1.SecretStoreProvider{
				Webhook: &v1beta1.WebhookProvider{
					URL: "http://example.com",
				},
			},
		},
	}, nil, "")
	if err != nil {
		panic(err)
	}
	fmt.Printf("starting server")
	shell.RunServer(pc)
}
