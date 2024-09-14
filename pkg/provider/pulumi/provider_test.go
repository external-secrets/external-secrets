// Copyright External Secrets Inc. All Rights Reserved
package pulumi

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func TestValidateStore(t *testing.T) {
	tests := map[string]struct {
		cfg  esv1beta1.PulumiProvider
		want error
	}{
		"invalid without organization": {
			cfg: esv1beta1.PulumiProvider{
				Organization: "",
				Environment:  "foo",
			},
			want: errors.New("organization is required"),
		},
		"invalid without environment": {
			cfg: esv1beta1.PulumiProvider{
				Organization: "foo",
				Environment:  "",
			},
			want: errors.New("environment is required"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Pulumi: &tc.cfg,
					},
				},
			}
			p := &Provider{}
			_, got := p.ValidateStore(&s)
			assert.Equal(t, tc.want, got)
		})
	}
}
