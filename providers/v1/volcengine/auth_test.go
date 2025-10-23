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

package volcengine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestNewSession_should_return_session_with_default_credentials_when_auth_is_nil(t *testing.T) {
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth:   nil,
	}
	kube := fake.NewClientBuilder().Build()

	sess, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, testRegion, *sess.Config.Region)
	_, err = sess.Config.Credentials.Get()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required environment variables: VOLCENGINE_OIDC_TOKEN_FILE or VOLCENGINE_OIDC_ROLE_TRN")
}

func TestNewSession_should_return_session_with_default_credentials_when_secretref_is_nil(t *testing.T) {
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth:   &esv1.VolcengineAuth{},
	}
	kube := fake.NewClientBuilder().Build()

	sess, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, testRegion, *sess.Config.Region)
	_, err = sess.Config.Credentials.Get()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required environment variables: VOLCENGINE_OIDC_TOKEN_FILE or VOLCENGINE_OIDC_ROLE_TRN")
}

func TestNewSession_should_return_session_with_static_credentials_when_secretref_is_provided(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			accessKeyIDKey:     []byte(testAccessKeyID),
			secretAccessKeyKey: []byte(testSecretAccessKey),
		},
	}
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth: &esv1.VolcengineAuth{
			SecretRef: &esv1.VolcengineAuthSecretRef{
				AccessKeyID: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  accessKeyIDKey,
				},
				SecretAccessKey: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  secretAccessKeyKey,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	sess, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, testRegion, *sess.Config.Region)
	creds, err := sess.Config.Credentials.Get()
	assert.NoError(t, err)
	assert.Equal(t, testAccessKeyID, creds.AccessKeyID)
	assert.Equal(t, testSecretAccessKey, creds.SecretAccessKey)
}

func TestNewSession_should_return_error_when_accesskeyid_secret_is_not_found(t *testing.T) {
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth: &esv1.VolcengineAuth{
			SecretRef: &esv1.VolcengineAuthSecretRef{
				AccessKeyID: esmeta.SecretKeySelector{
					Name: "non-existent-secret",
					Key:  accessKeyIDKey,
				},
			},
		},
	}
	kube := fake.NewClientBuilder().Build()

	_, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get accessKeyID")
}

func TestNewSession_should_return_error_when_secretaccesskey_secret_is_not_found(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			accessKeyIDKey: []byte(testAccessKeyID),
		},
	}
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth: &esv1.VolcengineAuth{
			SecretRef: &esv1.VolcengineAuthSecretRef{
				AccessKeyID: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  accessKeyIDKey,
				},
				SecretAccessKey: esmeta.SecretKeySelector{
					Name: "non-existent-secret",
					Key:  secretAccessKeyKey,
				},
			},
		},
	}
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	_, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secretAccessKey")
}

func TestNewSession_should_return_error_when_accesskeyid_key_is_not_found_in_secret(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{},
	}
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth: &esv1.VolcengineAuth{
			SecretRef: &esv1.VolcengineAuthSecretRef{
				AccessKeyID: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  "non-existent-key",
				},
			},
		},
	}
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	_, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key \"non-existent-key\" not found in secret")
}

func TestNewSession_should_return_session_with_token_credentials_when_secretref_is_provided(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			accessKeyIDKey:     []byte(testAccessKeyID),
			secretAccessKeyKey: []byte(testSecretAccessKey),
			tokenKey:           []byte(testSessionToken),
		},
	}
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth: &esv1.VolcengineAuth{
			SecretRef: &esv1.VolcengineAuthSecretRef{
				AccessKeyID: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  accessKeyIDKey,
				},
				SecretAccessKey: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  secretAccessKeyKey,
				},
				Token: &esmeta.SecretKeySelector{
					Name: secretName,
					Key:  tokenKey,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	sess, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, testRegion, *sess.Config.Region)
	creds, err := sess.Config.Credentials.Get()
	assert.NoError(t, err)
	assert.Equal(t, testAccessKeyID, creds.AccessKeyID)
	assert.Equal(t, testSecretAccessKey, creds.SecretAccessKey)
	assert.Equal(t, testSessionToken, creds.SessionToken)
}

func TestNewSession_should_return_error_when_secretaccesskey_key_is_not_found_in_secret(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			accessKeyIDKey: []byte(testAccessKeyID),
		},
	}
	store := &esv1.VolcengineProvider{
		Region: testRegion,
		Auth: &esv1.VolcengineAuth{
			SecretRef: &esv1.VolcengineAuthSecretRef{
				AccessKeyID: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  accessKeyIDKey,
				},
				SecretAccessKey: esmeta.SecretKeySelector{
					Name: secretName,
					Key:  "non-existent-key",
				},
			},
		},
	}
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	_, err := NewSession(context.Background(), store, kube, testNamespace)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key \"non-existent-key\" not found in secret")
}

func TestNewSession_should_return_error_when_store_is_nil(t *testing.T) {
	ctx := context.Background()
	var store *esv1.VolcengineProvider
	var kube client.Client
	namespace := "default"

	sess, err := NewSession(ctx, store, kube, namespace)

	assert.Error(t, err)
	assert.Nil(t, sess)
	assert.Equal(t, "volcengine provider can not be nil", err.Error())
}

func TestNewSession_should_return_error_when_region_is_empty(t *testing.T) {
	ctx := context.Background()
	store := &esv1.VolcengineProvider{}
	var kube client.Client
	namespace := "default"

	sess, err := NewSession(ctx, store, kube, namespace)

	assert.Error(t, err)
	assert.Nil(t, sess)
	assert.Equal(t, "region must be specified", err.Error())
}
