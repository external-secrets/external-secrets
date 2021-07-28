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
	"log"
	"os"

	gitlab "github.com/xanzy/go-gitlab"
)

// Requires a token to be set in environment variable
var GITLABTOKEN = os.Getenv("GITLABTOKEN")

type GitlabCredentials struct {
	Token string `json:"token"`
}

// Gitlab struct with values for *gitlab.Client and projectID
type Gitlab struct {
	client    *gitlab.Client
	projectID interface{}
}

// Function newGitlabProvider returns a reference to a new Gitlab struct 'instance'
func NewGitlabProvider() *Gitlab {
	return &Gitlab{}
}

// Method on Gitlab to set up client with credentials and populate projectID
func (g *Gitlab) NewGitlabClient(cred GitlabCredentials, projectID int) {
	var err error
	// Create a new Gitlab client with credentials
	g.client, err = gitlab.NewClient(cred.Token, nil)
	g.projectID = projectID
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
}
