/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package pulumi

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestValidateStore(t *testing.T) {
	tests := map[string]struct {
		cfg  esv1.PulumiProvider
		want error
	}{
		"invalid without organization": {
			cfg: esv1.PulumiProvider{
				Organization: "",
				Environment:  "foo",
			},
			want: errors.New("organization is required"),
		},
		"invalid without environment": {
			cfg: esv1.PulumiProvider{
				Organization: "foo",
				Environment:  "",
			},
			want: errors.New("environment is required"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
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
