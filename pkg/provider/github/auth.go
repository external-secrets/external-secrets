// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	github "github.com/google/go-github/v56/github"

	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

func (g *Client) AuthWithPrivateKey(ctx context.Context) (*github.Client, error) {
	privateKey, err := resolvers.SecretKeyRef(ctx, g.crClient, g.storeKind, g.namespace, &g.provider.Auth.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't get private key from secret: resolvers.SecretKeyRef failed with error %w", err)
	}

	itr, err := ghinstallation.New(http.DefaultTransport, g.provider.AppID, g.provider.InstallationID, []byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("could not instantiate new installation transport: %w", err)
	}
	client := github.NewClient(&http.Client{Transport: itr})
	if (g.provider.URL != "") && (g.provider.URL != "https://github.com/") {
		uploadURL := g.provider.UploadURL
		if uploadURL == "" {
			uploadURL = g.provider.URL
		}
		return client.WithEnterpriseURLs(g.provider.URL, uploadURL)
	}
	return client, nil
}
