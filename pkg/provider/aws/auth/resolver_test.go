//Copyright External Secrets Inc. All Rights Reserved

package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolver(t *testing.T) {
	tbl := []struct {
		env     string
		service string
		url     string
	}{
		{
			env:     SecretsManagerEndpointEnv,
			service: "secretsmanager",
			url:     "http://sm.foo",
		},
		{
			env:     SSMEndpointEnv,
			service: "ssm",
			url:     "http://ssm.foo",
		},
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
