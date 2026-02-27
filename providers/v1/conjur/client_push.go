/*
Copyright © The ESO Authors

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

package conjur

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/cyberark/conjur-api-go/conjurapi"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const policyTemplate = `
- !policy
  id: {{ .Name }}
  body:
  - !variable
    id: {{ .Variable }}
`

func conjurPolicy(name, variable string) string {
	type policy struct {
		Name     string
		Variable string
	}
	p := policy{
		Name:     name,
		Variable: variable,
	}
	t := template.Must(template.New("policy").Parse(policyTemplate))
	buf := &bytes.Buffer{}
	_ = t.Execute(buf, p)
	return buf.String()
}

// PushSecret writes a single secret into the provider.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, ref esv1.PushSecretData) error {
	conjurClient, getConjurClientError := c.GetConjurClient(ctx)
	if getConjurClientError != nil {
		return getConjurClientError
	}

	val, ok := secret.Data[ref.GetSecretKey()]
	if !ok {
		return errors.New("key not found")
	}

	fqSecretName := ref.GetRemoteKey()
	// if property is empty, we should create multiple variables for each key of the secret
	secretKey := ref.GetProperty()
	i := strings.LastIndex(fqSecretName, "/")
	if i == -1 {
		return errors.New("Expected RemoteKey to contain a '/'")
	}
	if secretKey == "" {
		return errors.New("property required")
	}
	parentPolicy := fqSecretName[0:i]
	policyName := fqSecretName[i+1:]
	policy := conjurPolicy(policyName, secretKey)

	_, err := conjurClient.LoadPolicy(conjurapi.PolicyModePost, parentPolicy, strings.NewReader(policy))
	if err != nil {
		return err
	}
	err = conjurClient.AddSecret(fmt.Sprintf("%s/%s", fqSecretName, secretKey), string(val))
	if err != nil {
		return err
	}
	return nil
}
