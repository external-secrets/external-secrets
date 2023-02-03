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
	},
})

func newTestClient() esv1beta1.SecretsClient {
	return &client{
		api: db,
	}
}

func TestGetSecret(t *testing.T) {

	ctx := context.Background()
	c := newTestClient()

	secret := db.secrets[0]

	// TODO: test that the error is NOT NoSecretErr when an error other than "not found" occurs

	testCases := map[string]struct {
		ref      esv1beta1.ExternalSecretDataRemoteRef
		response []byte
		err      error
	}{
		"empty version should mean latest": {
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
			response: secret.versions[1].data,
		},
		"asking for version by revision number": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "id:" + secret.id,
				Version: "1",
			},
			response: secret.versions[0].data,
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

	t.Run("to existing empty secret", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()
		data := []byte("some secret data 6a8ff33b-c69a-4e42-b162-b7b595ee7f5f")
		secret := db.secret("push-me")

		pushErr := c.PushSecret(ctx, data, pushRemoteRef("name:"+secret.name))

		assert.NoError(t, pushErr)
		assert.Equal(t, 1, len(secret.versions))
		assert.Equal(t, data, secret.versions[0].data)
	})

	t.Run("without change", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()
		secret := db.secret("not-changed")

		pushErr := c.PushSecret(ctx, secret.versions[0].data, pushRemoteRef("name:"+secret.name))

		assert.NoError(t, pushErr)
		assert.Equal(t, 1, len(secret.versions))
	})

	t.Run("non existing secret", func(t *testing.T) {

		ctx := context.Background()
		c := newTestClient()
		notASecret := "3fe3b79b-66c6-4b74-a03f-8a4be04afdd1"

		pushErr := c.PushSecret(ctx, []byte("some data"), pushRemoteRef(notASecret))

		assert.Error(t, pushErr)
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
				db.secret("secret-1").id: db.secret("secret-1").mustGetVersion("latest").data,
				db.secret("secret-2").id: db.secret("secret-2").mustGetVersion("latest").data,
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

	// TODO: test that the error is NOT NoSecretErr when an error other than "not found" occurs

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
