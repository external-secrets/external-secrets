// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package github

import (
	"context"
	"errors"
	"testing"

	github "github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type getSecretFn func(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) (*github.Secret, *github.Response, error)

func withGetSecretFn(secret *github.Secret, response *github.Response, err error) getSecretFn {
	return func(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (*github.Secret, *github.Response, error) {
		return secret, response, err
	}
}

type getPublicKeyFn func(ctx context.Context) (*github.PublicKey, *github.Response, error)

func withGetPublicKeyFn(key *github.PublicKey, response *github.Response, err error) getPublicKeyFn {
	return func(_ context.Context) (*github.PublicKey, *github.Response, error) {
		return key, response, err
	}
}

type createOrUpdateSecretFn func(ctx context.Context, encryptedSecret *github.EncryptedSecret) (*github.Response, error)

func withCreateOrUpdateSecretFn(response *github.Response, err error) createOrUpdateSecretFn {
	return func(_ context.Context, _ *github.EncryptedSecret) (*github.Response, error) {
		return response, err
	}
}

func TestSecretExists(t *testing.T) {
	type testCase struct {
		name        string
		prov        *esv1beta1.GithubProvider
		remoteRef   esv1beta1.PushSecretData
		getSecretFn getSecretFn
		wantErr     error
		exists      bool
	}
	tests := []testCase{
		{
			name:        "getSecret fail",
			getSecretFn: withGetSecretFn(nil, nil, errors.New("boom")),
			exists:      false,
			wantErr:     errors.New("error fetching secret"),
		},
		{
			name:        "no secret",
			getSecretFn: withGetSecretFn(nil, nil, nil),
			exists:      false,
		},
		{
			name:        "with secret",
			getSecretFn: withGetSecretFn(&github.Secret{}, nil, nil),
			exists:      true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := Client{
				provider: test.prov,
			}
			g.getSecretFn = test.getSecretFn
			ok, err := g.SecretExists(context.TODO(), test.remoteRef)
			assert.Equal(t, test.exists, ok)
			if test.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.wantErr.Error())
			}
		})
	}
}

func TestPushSecret(t *testing.T) {
	type testCase struct {
		name             string
		prov             *esv1beta1.GithubProvider
		secret           *corev1.Secret
		remoteRef        esv1beta1.PushSecretData
		getSecretFn      getSecretFn
		getPublicKeyFn   getPublicKeyFn
		createOrUpdateFn createOrUpdateSecretFn
		wantErr          error
	}
	tests := []testCase{
		{
			name:        "failGetSecretFn",
			getSecretFn: withGetSecretFn(nil, nil, errors.New("boom")),
			wantErr:     errors.New("error fetching secret"),
		},
		{
			name: "failGetPublicKey",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(nil, nil, errors.New("boom")),
			wantErr:        errors.New("error fetching public key"),
		},
		{
			name: "failDecodeKey",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("broken"),
				KeyID: ptr.To("123"),
			}, nil, nil),
			wantErr: errors.New("unable to decode public key"),
		},
		{
			name: "failSecretData",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("Cg=="),
				KeyID: ptr.To("123"),
			}, nil, nil),
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bar"),
				},
			},
			remoteRef: esv1alpha1.PushSecretData{
				Match: esv1alpha1.PushSecretMatch{
					SecretKey: "bar",
				},
			},
			wantErr: errors.New("not found in secret"),
		},
		{
			name: "failSecretData",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("Zm9vYmFyCg=="),
				KeyID: ptr.To("123"),
			}, nil, nil),
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bingg"),
				},
			},
			remoteRef: esv1alpha1.PushSecretData{
				Match: esv1alpha1.PushSecretMatch{
					SecretKey: "foo",
				},
			},
			createOrUpdateFn: withCreateOrUpdateSecretFn(nil, errors.New("boom")),
			wantErr:          errors.New("failed to create secret"),
		},
		{
			name: "Success",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("Zm9vYmFyCg=="),
				KeyID: ptr.To("123"),
			}, nil, nil),
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bingg"),
				},
			},
			remoteRef: esv1alpha1.PushSecretData{
				Match: esv1alpha1.PushSecretMatch{
					SecretKey: "foo",
				},
			},
			createOrUpdateFn: withCreateOrUpdateSecretFn(nil, nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := Client{
				provider: test.prov,
			}
			g.getSecretFn = test.getSecretFn
			g.getPublicKeyFn = test.getPublicKeyFn
			g.createOrUpdateFn = test.createOrUpdateFn
			err := g.PushSecret(context.TODO(), test.secret, test.remoteRef)
			if test.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.wantErr.Error())
			}
		})
	}
}
