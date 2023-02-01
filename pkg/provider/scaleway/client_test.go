package scaleway

import (
	"context"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/stretchr/testify/assert"
	"testing"
)

var db = buildDb(&fakeSecretApi{
	projects: []*fakeProject{
		{
			secrets: []*fakeSecret{
				{
					name: "secret-1",
					versions: []*fakeSecretVersion{
						{
							revision: 1,
						},
						{
							revision: 2,
						},
					},
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

	secret := db.projects[0].secrets[0]

	// TODO: test that the error is NOT NoSecretErr when an error other than "not found" occurs

	testCases := map[string]struct {
		ref      esv1beta1.ExternalSecretDataRemoteRef
		response []byte
		err      error
	}{
		"empty version should mean latest": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     secret.id,
				Version: "",
			},
			response: secret.versions[1].data,
		},
		"asking for latest version": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     secret.id,
				Version: "latest",
			},
			response: secret.versions[1].data,
		},
		"asking for version by revision number": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     secret.id,
				Version: "1",
			},
			response: secret.versions[0].data,
		},
		"non existing revision should yield NoSecretErr": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     secret.id,
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
