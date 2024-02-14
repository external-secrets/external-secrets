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

package scaleway

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

var db = buildDB(&fakeSecretAPI{
	secrets: []*fakeSecret{
		{
			name: "secret-1",
			versions: []*fakeSecretVersion{
				{revision: 1},
				{revision: 2},
				{revision: 3, status: "disabled"},
			},
		},
		{
			name: "secret-2",
			tags: []string{"secret-2-tag-1", "secret-2-tag-2"},
			versions: []*fakeSecretVersion{
				{revision: 1},
				{revision: 2},
			},
		},
		{
			name:     "push-me",
			versions: []*fakeSecretVersion{},
		},
		{
			name: "not-changed",
			versions: []*fakeSecretVersion{
				{revision: 1},
			},
		},
		{
			name: "disabling-old-versions",
			versions: []*fakeSecretVersion{
				{revision: 1},
			},
		},
		{
			name: "json-data",
			versions: []*fakeSecretVersion{
				{
					revision: 1,
					data:     []byte(`{"some_string": "abc def", "some_int": -100, "some_bool": false}`),
				},
			},
		},
		{
			name: "cant-push",
			versions: []*fakeSecretVersion{
				{revision: 1},
			},
		},
		{
			name: "json-nested",
			versions: []*fakeSecretVersion{
				{revision: 1, data: []byte(
					`{"root":{"intermediate":{"leaf":9}}}`,
				)},
			},
		},
		{
			name: "nested-secret",
			path: "/subpath",
			versions: []*fakeSecretVersion{
				{
					revision: 1,
					data:     []byte("secret data"),
				},
			},
		},
	},
})

func newTestClient() esv1beta1.SecretsClient {
	return &client{
		api:   db,
		cache: newCache(),
	}
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	secret := db.secrets[0]

	testCases := map[string]struct {
		ref      esv1beta1.ExternalSecretDataRemoteRef
		response []byte
		err      error
	}{
		"empty version should mean latest_enabled": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "",
			},
			response: secret.versions[1].data,
		},
		"asking for latest version": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "latest",
			},
			response: secret.versions[2].data,
		},
		"asking for latest version by name": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "name:" + secret.name,
				Version: "latest",
			},
			response: secret.versions[2].data,
		},
		"asking for version by revision number": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "1",
			},
			response: secret.versions[0].data,
		},
		"asking for version by revision number and name": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "name:" + secret.name,
				Version: "1",
			},
			response: secret.versions[0].data,
		},
		"asking for nested json property": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "id:" + db.secret("json-nested").id,
				Property: "root.intermediate.leaf",
				Version:  "latest",
			},
			response: []byte("9"),
		},
		"secret in path": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "path:/subpath/nested-secret",
				Version: "latest",
			},
			response: []byte("secret data"),
		},
		"non existing secret id should yield NoSecretErr": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "id:730aa98d-ec0c-4426-8202-b11aeec8ea1e",
			},
			err: esv1beta1.NoSecretErr,
		},
		"non existing secret name should yield NoSecretErr": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "name:not-a-secret",
			},
			err: esv1beta1.NoSecretErr,
		},
		"non existing revision should yield NoSecretErr": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "9999",
			},
			err: esv1beta1.NoSecretErr,
		},
		"non existing json property should yield not found": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "id:" + db.secret("json-nested").id,
				Property: "root.intermediate.missing",
				Version:  "latest",
			},
			err: esv1beta1.NoSecretErr,
		},
	}

	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			response, err := c.GetSecret(ctx, tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.response, response)
			} else {
				assert.Nil(t, response)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestPushSecret(t *testing.T) {
	secretKey := "secret-key"
	pushSecretData := func(remoteKey string) testingfake.PushSecretData {
		return testingfake.PushSecretData{
			SecretKey: secretKey,
			RemoteKey: remoteKey,
		}
	}
	secret := func(value []byte) *corev1.Secret {
		return &corev1.Secret{
			Data: map[string][]byte{secretKey: value},
		}
	}
	t.Run("to new secret", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data 6a8ff33b-c69a-4e42-b162-b7b595ee7f5f")
		secretName := "secret-creation-test"

		pushErr := c.PushSecret(ctx, secret(data), pushSecretData(fmt.Sprintf("name:%s", secretName)))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
		assert.Equal(t, data, db.secret(secretName).versions[0].data)
	})

	t.Run("to secret created by us", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data a11d416b-9169-4f4a-8c27-d2959b22e189")
		secretName := "secret-update-test"
		assert.NoError(t, c.PushSecret(ctx, secret([]byte("original data")), pushSecretData(fmt.Sprintf("name:%s", secretName))))

		pushErr := c.PushSecret(ctx, secret(data), pushSecretData(fmt.Sprintf("name:%s", secretName)))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 2)
		assert.Equal(t, data, db.secret(secretName).versions[1].data)
	})

	t.Run("to secret partially created by us with no version", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data a11d416b-9169-4f4a-8c27-d2959b22e189")
		secretName := "push-me"

		pushErr := c.PushSecret(ctx, secret(data), pushSecretData(fmt.Sprintf("name:%s", secretName)))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
		assert.Equal(t, data, db.secret(secretName).versions[0].data)
	})

	t.Run("secret created in path", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data in path")
		secretPath := "/folder"
		secretName := "secret-in-path"

		pushErr := c.PushSecret(ctx, secret(data), pushSecretData(fmt.Sprintf("path:%s/%s", secretPath, secretName)))
		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
		assert.Equal(t, data, db.secret(secretName).versions[0].data)
		assert.Equal(t, secretPath, db.secret(secretName).path)
	})

	t.Run("by invalid secret ref is an error", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()

		pushErr := c.PushSecret(ctx, secret([]byte("some data")), pushSecretData("invalid:abcd"))

		assert.Error(t, pushErr)
	})

	t.Run("by id is an error", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()

		pushErr := c.PushSecret(ctx, secret([]byte("some data")), pushSecretData(fmt.Sprintf("id:%s", db.secret("cant-push").id)))

		assert.Error(t, pushErr)
	})

	t.Run("without change does not create a version", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		fs := db.secret("not-changed")

		pushErr := c.PushSecret(ctx, secret(fs.versions[0].data), pushSecretData(fmt.Sprintf("name:%s", fs.name)))

		assert.NoError(t, pushErr)
		assert.Equal(t, 1, len(fs.versions))
	})

	t.Run("previous version is disabled", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		fs := db.secret("disabling-old-versions")

		pushErr := c.PushSecret(ctx, secret([]byte("some new data")), pushSecretData(fmt.Sprintf("name:%s", fs.name)))

		assert.NoError(t, pushErr)
		assert.Equal(t, 2, len(fs.versions))
		assert.Equal(t, "disabled", fs.versions[0].status)
	})
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	values, getErr := c.GetSecretMap(ctx, esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "id:" + db.secret("json-data").id,
		Version: "latest",
	})

	assert.NoError(t, getErr)
	assert.Equal(t, map[string][]byte{
		"some_string": []byte("abc def"),
		"some_int":    []byte("-100"),
		"some_bool":   []byte("false"),
	}, values)
}

func TestGetSecretMapNested(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	values, getErr := c.GetSecretMap(ctx, esv1beta1.ExternalSecretDataRemoteRef{
		Key:      "id:" + db.secret("json-nested").id,
		Property: "root.intermediate",
		Version:  "latest",
	})

	assert.NoError(t, getErr)
	assert.Equal(t, map[string][]byte{
		"leaf": []byte("9"),
	}, values)
}

func TestGetAllSecrets(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	testCases := map[string]struct {
		ref      esv1beta1.ExternalSecretFind
		response map[string][]byte
		err      error
	}{
		"find secrets by name": {
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{RegExp: "secret-.*"},
			},
			response: map[string][]byte{
				db.secret("secret-1").name: db.secret("secret-1").mustGetVersion("latest_enabled").data,
				db.secret("secret-2").name: db.secret("secret-2").mustGetVersion("latest_enabled").data,
			},
		},
		"find secrets by tags": {
			ref: esv1beta1.ExternalSecretFind{
				Tags: map[string]string{"secret-2-tag-1": "ignored-value"},
			},
			response: map[string][]byte{
				db.secrets[1].name: db.secrets[1].mustGetVersion("latest").data,
			},
		},
		"find secrets by path": {
			ref: esv1beta1.ExternalSecretFind{
				Path: utils.Ptr("/subpath"),
			},
			response: map[string][]byte{
				db.secret("nested-secret").name: db.secret("nested-secret").mustGetVersion("latest_enabled").data,
			},
		},
	}

	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			response, err := c.GetAllSecrets(ctx, tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.response, response)
			} else {
				assert.Nil(t, response)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	secret := db.secrets[0]
	byPath := db.secret("nested-secret")

	testCases := map[string]struct {
		ref testingfake.PushSecretData
		err error
	}{
		"Delete Successfully": {
			ref: testingfake.PushSecretData{RemoteKey: "name:" + secret.name},
			err: nil,
		},
		"Delete by path": {
			ref: testingfake.PushSecretData{RemoteKey: "path:" + byPath.path + "/" + byPath.name},
			err: nil,
		},
		"Secret Not Found": {
			ref: testingfake.PushSecretData{RemoteKey: "name:not-a-secret"},
			err: nil,
		},
	}

	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			err := c.DeleteSecret(ctx, tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestSplitNameAndPath(t *testing.T) {
	type test struct {
		in   string
		name string
		path string
		ok   bool
	}

	tests := []test{
		{
			in:   "/foo",
			name: "foo",
			path: "/",
			ok:   true,
		},
		{
			in:   "",
			name: "",
			path: "",
		},
		{
			in:   "/foo/bar",
			name: "bar",
			path: "/foo",
			ok:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			name, path, ok := splitNameAndPath(tc.in)
			assert.Equal(t, tc.ok, ok, "bad ref")
			if tc.ok {
				assert.Equal(t, tc.name, name, "wrong name")
				assert.Equal(t, tc.path, path, "wrong path")
			}
		})
	}
}
