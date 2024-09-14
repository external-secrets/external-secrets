//Copyright External Secrets Inc. All Rights Reserved

package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/external-secrets/external-secrets/pkg/provider/util/fake"
)

func TestTokenFetcher(t *testing.T) {
	tf := &authTokenFetcher{
		ServiceAccount: "foobar",
		Namespace:      "example",
		k8sClient:      fake.NewCreateTokenMock().WithToken("FAKETOKEN"),
	}
	token, err := tf.FetchToken(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("FAKETOKEN"), token)
}
