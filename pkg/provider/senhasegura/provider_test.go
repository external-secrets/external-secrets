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

package senhasegura

import (
	"testing"

	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func TestValidateStore(t *testing.T) {
	tbl := []struct {
		test   string
		store  esv1beta1.GenericStore
		expErr bool
	}{
		{
			test:   "should not create provider due to nil store",
			store:  nil,
			expErr: true,
		},
		{
			test:   "should not create provider due to missing provider",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{},
			},
		},
		{
			test:   "should not create provider due to missing provider field",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{},
				},
			},
		},
		{
			test:   "should not create provider due to missing provider module",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Senhasegura: &esv1beta1.SenhaseguraProvider{},
					},
				},
			},
		},
		{
			test:   "should not create provider due to missing provider auth client ID",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Senhasegura: &esv1beta1.SenhaseguraProvider{
							Module: esv1beta1.SenhaseguraModuleDSM,
						},
					},
				},
			},
		},
		{
			test:   "invalid module should return an error",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Senhasegura: &esv1beta1.SenhaseguraProvider{
							Module: "HIHIHIHHEHEHEHEHEHE",
						},
					},
				},
			},
		},
		{
			test:   "should not create provider due senhasegura URL without https scheme",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Senhasegura: &esv1beta1.SenhaseguraProvider{
							Module: esv1beta1.SenhaseguraModuleDSM,
							URL:    "http://dev.null",
						},
					},
				},
			},
		},
		{
			test:   "should not create provider due senhasegura URL without valid name",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Senhasegura: &esv1beta1.SenhaseguraProvider{
							Module: esv1beta1.SenhaseguraModuleDSM,
							URL:    "https://",
						},
					},
				},
			},
		},
		{
			test:   "should create provider",
			expErr: false,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Senhasegura: &esv1beta1.SenhaseguraProvider{
							Module: esv1beta1.SenhaseguraModuleDSM,
							URL:    "https://senhasegura.local",
							Auth: esv1beta1.SenhaseguraAuth{
								ClientID: "example",
							},
						},
					},
				},
			},
		},
	}
	for i := range tbl {
		row := tbl[i]
		t.Run(row.test, func(t *testing.T) {
			err := validateStore(row.store)
			if row.expErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
