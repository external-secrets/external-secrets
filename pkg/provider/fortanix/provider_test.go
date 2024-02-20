/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package fortanix

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestValidateStore(t *testing.T) {
	tests := map[string]struct {
		cfg  esv1beta1.FortanixProvider
		want error
	}{
		"missing api key": {
			cfg:  esv1beta1.FortanixProvider{},
			want: errors.New("apiKey is required"),
		},
		"missing api key secret ref": {
			cfg: esv1beta1.FortanixProvider{
				ApiKey: &esv1beta1.FortanixProviderSecretRef{},
			},
			want: errors.New("apiKey.secretRef is required"),
		},
		"missing api key secret ref name": {
			cfg: esv1beta1.FortanixProvider{
				ApiKey: &esv1beta1.FortanixProviderSecretRef{
					SecretRef: &v1.SecretKeySelector{
						Key: "key",
					},
				},
			},
			want: errors.New("apiKey.secretRef.name is required"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Fortanix: &tc.cfg,
					},
				},
			}
			p := &Provider{}
			_, got := p.ValidateStore(&s)
			assert.Equal(t, tc.want, got)
		})
	}
}
