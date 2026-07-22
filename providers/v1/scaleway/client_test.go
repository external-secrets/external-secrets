/*
Copyright © The ESO Authors

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

package scaleway

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
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
			name: "json-dotted-keys",
			versions: []*fakeSecretVersion{
				{revision: 1, data: []byte(
					`{"tls.crt":"CERT","username":"alice","a.b":"literal","a":{"b":"nested"}}`,
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

func newTestClient() esv1.SecretsClient {
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
		ref      esv1.ExternalSecretDataRemoteRef
		response []byte
		err      error
	}{
		"empty version should mean latest_enabled": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "",
			},
			response: secret.versions[1].data,
		},
		"asking for latest version": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "latest",
			},
			response: secret.versions[2].data,
		},
		"asking for latest version by name": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "name:" + secret.name,
				Version: "latest",
			},
			response: secret.versions[2].data,
		},
		"asking for version by revision number": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "1",
			},
			response: secret.versions[0].data,
		},
		"asking for version by revision number and name": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "name:" + secret.name,
				Version: "1",
			},
			response: secret.versions[0].data,
		},
		"asking for nested json property": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "id:" + db.secret("json-nested").id,
				Property: "root.intermediate.leaf",
				Version:  "latest",
			},
			response: []byte("9"),
		},
		"literal dotted key is found without escaping": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "id:" + db.secret("json-dotted-keys").id,
				Property: "tls.crt",
				Version:  "latest",
			},
			response: []byte("CERT"),
		},
		"nested path still wins over literal fallback": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "id:" + db.secret("json-dotted-keys").id,
				Property: "a.b",
				Version:  "latest",
			},
			response: []byte("nested"),
		},
		"secret in path": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "path:/subpath/nested-secret",
				Version: "latest",
			},
			response: []byte("secret data"),
		},
		"non existing secret id should yield NoSecretErr": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "id:730aa98d-ec0c-4426-8202-b11aeec8ea1e",
			},
			err: esv1.NoSecretErr,
		},
		"non existing secret name should yield NoSecretErr": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "name:not-a-secret",
			},
			err: esv1.NoSecretErr,
		},
		"non existing revision should yield NoSecretErr": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "9999",
			},
			err: esv1.NoSecretErr,
		},
		"non existing json property should yield not found": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "id:" + db.secret("json-nested").id,
				Property: "root.intermediate.missing",
				Version:  "latest",
			},
			err: esv1.NoSecretErr,
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
	pushSecretDataWithProperty := func(remoteKey, property string) testingfake.PushSecretData {
		return testingfake.PushSecretData{
			SecretKey: secretKey,
			RemoteKey: remoteKey,
			Property:  property,
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

	t.Run("whole secret is pushed as a JSON object", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "whole-secret-test"
		wholeSecret := &corev1.Secret{Data: map[string][]byte{
			"username": []byte("alice"),
			"password": []byte("s3cr3t"),
		}}

		pushErr := c.PushSecret(ctx, wholeSecret, testingfake.PushSecretData{RemoteKey: "name:" + secretName})

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
		assert.JSONEq(t, `{"username":"alice","password":"s3cr3t"}`, string(db.secret(secretName).versions[0].data))
	})

	t.Run("whole secret push without change does not create a version", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "whole-secret-idempotent"
		wholeSecret := &corev1.Secret{Data: map[string][]byte{"a": []byte("1"), "b": []byte("2")}}

		assert.NoError(t, c.PushSecret(ctx, wholeSecret, testingfake.PushSecretData{RemoteKey: "name:" + secretName}))
		assert.NoError(t, c.PushSecret(ctx, wholeSecret, testingfake.PushSecretData{RemoteKey: "name:" + secretName}))

		assert.Len(t, db.secret(secretName).versions, 1)
	})

	t.Run("property push to new secret creates a JSON object", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "property-new-secret"

		pushErr := c.PushSecret(ctx, secret([]byte("alice")), pushSecretDataWithProperty("name:"+secretName, "username"))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
		assert.JSONEq(t, `{"username":"alice"}`, string(db.secret(secretName).versions[0].data))
	})

	t.Run("property push merges with existing object and disables previous version", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "property-merge-secret"
		assert.NoError(t, c.PushSecret(ctx, secret([]byte("alice")), pushSecretDataWithProperty("name:"+secretName, "username")))

		pushErr := c.PushSecret(ctx, secret([]byte("s3cr3t")), pushSecretDataWithProperty("name:"+secretName, "password"))

		assert.NoError(t, pushErr)
		fs := db.secret(secretName)
		assert.Len(t, fs.versions, 2)
		assert.JSONEq(t, `{"username":"alice","password":"s3cr3t"}`, string(fs.versions[1].data))
		assert.Equal(t, "disabled", fs.versions[0].status)
	})

	t.Run("property push with dotted key stays a literal top-level key", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "property-dotted-secret"
		assert.NoError(t, c.PushSecret(ctx, secret([]byte("CERT")), pushSecretDataWithProperty("name:"+secretName, "tls.crt")))

		pushErr := c.PushSecret(ctx, secret([]byte("KEY")), pushSecretDataWithProperty("name:"+secretName, "tls.key"))

		assert.NoError(t, pushErr)
		fs := db.secret(secretName)
		assert.JSONEq(t, `{"tls.crt":"CERT","tls.key":"KEY"}`, string(fs.versions[len(fs.versions)-1].data))

		got, getErr := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
			Key: "name:" + secretName, Property: "tls.crt", Version: "latest",
		})
		assert.NoError(t, getErr)
		assert.Equal(t, []byte("CERT"), got)
	})

	t.Run("property push updates an existing nested path in place", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "property-nested-update"
		// seed with nested JSON via a raw push — do NOT reuse the shared
		// "json-nested" fixture, later tests read it and db is global
		assert.NoError(t, c.PushSecret(ctx, secret([]byte(`{"root":{"intermediate":{"leaf":"9"}}}`)), pushSecretData("name:"+secretName)))

		pushErr := c.PushSecret(ctx, secret([]byte("10")), pushSecretDataWithProperty("name:"+secretName, "root.intermediate.leaf"))

		assert.NoError(t, pushErr)
		fs := db.secret(secretName)
		assert.JSONEq(t, `{"root":{"intermediate":{"leaf":"10"}}}`, string(fs.versions[len(fs.versions)-1].data))
	})

	t.Run("whole secret push with property nests the object as a JSON string", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "whole-secret-with-property"
		wholeSecret := &corev1.Secret{Data: map[string][]byte{"user": []byte("alice")}}

		pushErr := c.PushSecret(ctx, wholeSecret, testingfake.PushSecretData{RemoteKey: "name:" + secretName, Property: "bundle"})

		assert.NoError(t, pushErr)
		assert.JSONEq(t, `{"bundle":"{\"user\":\"alice\"}"}`, string(db.secret(secretName).versions[0].data))
	})

	t.Run("property push replaces a raw non-object value", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "property-over-raw"
		assert.NoError(t, c.PushSecret(ctx, secret([]byte("raw bytes")), pushSecretData("name:"+secretName)))

		pushErr := c.PushSecret(ctx, secret([]byte("alice")), pushSecretDataWithProperty("name:"+secretName, "username"))

		assert.NoError(t, pushErr)
		fs := db.secret(secretName)
		assert.JSONEq(t, `{"username":"alice"}`, string(fs.versions[len(fs.versions)-1].data))
	})

	t.Run("property push without change does not create a version", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()
		secretName := "property-idempotent"
		assert.NoError(t, c.PushSecret(ctx, secret([]byte("alice")), pushSecretDataWithProperty("name:"+secretName, "username")))

		pushErr := c.PushSecret(ctx, secret([]byte("alice")), pushSecretDataWithProperty("name:"+secretName, "username"))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
	})

	t.Run("missing secret key is an error", func(t *testing.T) {
		ctx := context.Background()
		c := newTestClient()

		pushErr := c.PushSecret(ctx, secret([]byte("x")), testingfake.PushSecretData{SecretKey: "other-key", RemoteKey: "name:missing-key-test"})

		assert.Error(t, pushErr)
	})
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	values, getErr := c.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{
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

	values, getErr := c.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{
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
		ref      esv1.ExternalSecretFind
		response map[string][]byte
		err      error
	}{
		"find secrets by name": {
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "^secret-\\d$"},
			},
			response: map[string][]byte{
				db.secret("secret-1").name: db.secret("secret-1").mustGetVersion("latest_enabled").data,
				db.secret("secret-2").name: db.secret("secret-2").mustGetVersion("latest_enabled").data,
			},
		},
		"find secrets by tags": {
			ref: esv1.ExternalSecretFind{
				Tags: map[string]string{"secret-2-tag-1": "ignored-value"},
			},
			response: map[string][]byte{
				db.secrets[1].name: db.secrets[1].mustGetVersion("latest").data,
			},
		},
		"find secrets by path": {
			ref: esv1.ExternalSecretFind{
				Path: new("/subpath"),
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
