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
package iamauth

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

func TestResolver(t *testing.T) {
	tbl := []struct {
		env     string
		service string
		url     string
	}{
		{
			env:     STSEndpointEnv,
			service: "sts",
			url:     "http://sts.foo",
		},
	}

	for _, item := range tbl {
		t.Setenv(item.env, item.url)
	}

	f := ResolveEndpoint()

	for _, item := range tbl {
		ep, err := f.EndpointFor(item.service, "")
		assert.Nil(t, err)
		assert.Equal(t, item.url, ep.URL)
	}
}
