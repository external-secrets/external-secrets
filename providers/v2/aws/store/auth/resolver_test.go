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

package auth

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
)

// do we need this file now that resolving logic is isolated to each service?

func TestResolver(t *testing.T) {
	endpointEnvKey := STSEndpointEnv
	endpointURL := "http://sts.foo"

	t.Setenv(endpointEnvKey, endpointURL)

	f, err := customEndpointResolver{}.ResolveEndpoint(context.Background(), sts.EndpointParameters{})

	assert.Nil(t, err)
	assert.Equal(t, endpointURL, f.URI.String())
}
