package scaleway

import (
	"context"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/stretchr/testify/assert"
	"testing"
)

var db = buildDb(&fakeSecretApi{
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

type pushRemoteRef string

func (ref pushRemoteRef) GetRemoteKey() string {
	return string(ref)
}

func TestPushSecret(t *testing.T) {

	t.Run("to new secret", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data 6a8ff33b-c69a-4e42-b162-b7b595ee7f5f")
		secretName := "secret-creation-test"

		pushErr := c.PushSecret(ctx, data, pushRemoteRef("name:"+secretName))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
		assert.Equal(t, data, db.secret(secretName).versions[0].data)
	})

	t.Run("to secret created by us", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data a11d416b-9169-4f4a-8c27-d2959b22e189")
		secretName := "secret-update-test"
		assert.NoError(t, c.PushSecret(ctx, []byte("original data"), pushRemoteRef("name:"+secretName)))

		pushErr := c.PushSecret(ctx, data, pushRemoteRef("name:"+secretName))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 2)
		assert.Equal(t, data, db.secret(secretName).versions[1].data)
	})

	t.Run("to secret partially created by us with no version", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data a11d416b-9169-4f4a-8c27-d2959b22e189")
		secretName := "push-me"

		pushErr := c.PushSecret(ctx, data, pushRemoteRef("name:"+secretName))

		assert.NoError(t, pushErr)
		assert.Len(t, db.secret(secretName).versions, 1)
		assert.Equal(t, data, db.secret(secretName).versions[0].data)
	})

	t.Run("by invalid secret ref is an error", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()

		pushErr := c.PushSecret(ctx, []byte("some data"), pushRemoteRef("invalid:abcd"))

		assert.Error(t, pushErr)
	})

	t.Run("by id is an error", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()

		pushErr := c.PushSecret(ctx, []byte("some data"), pushRemoteRef("id:"+db.secret("cant-push").id))

		assert.Error(t, pushErr)
	})

	t.Run("without change does not create a version", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()
		secret := db.secret("not-changed")

		pushErr := c.PushSecret(ctx, secret.versions[0].data, pushRemoteRef("name:"+secret.name))

		assert.NoError(t, pushErr)
		assert.Equal(t, 1, len(secret.versions))
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
				db.secret("secret-1").id: db.secret("secret-1").mustGetVersion("latest_enabled").data,
				db.secret("secret-2").id: db.secret("secret-2").mustGetVersion("latest_enabled").data,
			},
		},
		"find secrets by tags": {
			ref: esv1beta1.ExternalSecretFind{
				Tags: map[string]string{"secret-2-tag-1": "ignored-value"},
			},
			response: map[string][]byte{
				db.secrets[1].id: db.secrets[1].mustGetVersion("latest").data,
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

	testCases := map[string]struct {
		ref esv1beta1.PushRemoteRef
		err error
	}{
		"Delete Successfully": {
			ref: pushRemoteRef("name:" + secret.name),
			err: nil,
		},
		"Secret Not Found": {
			ref: pushRemoteRef("name:not-a-secret"),
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
