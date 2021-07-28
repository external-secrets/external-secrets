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
package gitlab

import (
	"context"
	"fmt"
	"testing"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestCreateGitlabClient(t *testing.T) {
	credentials := GitlabCredentials{Token: GITLAB_TOKEN}
	gitlab := NewGitlabProvider()
	gitlab.NewGitlabClient(credentials, GITLAB_PROJECT_ID)

	// user, _, _ := gitlab.client.Users.CurrentUser()
	// fmt.Printf("Created client for username: %v", user)
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()

	ref := v1alpha1.ExternalSecretDataRemoteRef{Key: "mySecretBanana"}

	credentials := GitlabCredentials{Token: GITLAB_TOKEN}
	gitlab := NewGitlabProvider()
	gitlab.NewGitlabClient(credentials, GITLAB_PROJECT_ID)

	secretData, err := gitlab.GetSecret(ctx, ref)

	if err != nil {
		fmt.Printf("that's an error, Charlie. %v", err)
	}

	fmt.Printf("Got secret data %v", string(secretData))
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()

	ref := v1alpha1.ExternalSecretDataRemoteRef{Key: "myJsonSecret"}

	credentials := GitlabCredentials{Token: GITLAB_TOKEN}
	gitlab := NewGitlabProvider()
	gitlab.NewGitlabClient(credentials, GITLAB_PROJECT_ID)

	secretData, err := gitlab.GetSecretMap(ctx, ref)

	if err != nil {
		fmt.Errorf("that's an error, Charlie. %w", err)
	}

	fmt.Printf("Got secret map: %v", secretData)
}
