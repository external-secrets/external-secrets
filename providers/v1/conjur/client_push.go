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
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const policyTemplate = `
- !policy
  id: {{ .Name }}
  body:
{{- range .Variables}}
  - !variable
    id: {{ . }}
{{- end -}}
`

func conjurPolicy(name string, vars []string) string {
	type policy struct {
		Name      string
		Variables []string
	}
	p := policy{
		Name:      name,
		Variables: vars,
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

	values := map[string]string{}
	vars := []string{}

	key := ref.GetSecretKey()
	if key == "" {
		for k, v := range secret.Data {
			values[k] = string(v)
			vars = append(vars, k)
		}
	} else {
		value, ok := secret.Data[key]
		if !ok {
			return errors.New("key not found")
		}
		values[key] = string(value)
		vars = append(vars, key)
	}

	fqSecretName := ref.GetRemoteKey()
	// if property is empty, we should create multiple variables for each key of the secret
	secretKey := ref.GetProperty()
	i := strings.LastIndex(fqSecretName, "/")
	if i == -1 {
		return errors.New("Expected RemoteKey to contain a '/'")
	}
	if secretKey != "" {
		vars = []string{secretKey}
	}
	parentPolicy := fqSecretName[0:i]
	policyName := fqSecretName[i+1:]
	policy := conjurPolicy(policyName, vars)

	_, err := conjurClient.LoadPolicy(conjurapi.PolicyModePost, parentPolicy, strings.NewReader(policy))
	if err != nil {
		return err
	}
	// if we're not given a property, store all the secrets under the k8s secret key
	if secretKey == "" {
		for k, v := range values {
			err = conjurClient.AddSecret(fmt.Sprintf("%s/%s", fqSecretName, k), v)
			if err != nil {
				return err
			}
		}
	}
	// if we have a property and a single k8s secret key, store it "as is"
	if secretKey != "" && key != "" {
		err = conjurClient.AddSecret(fmt.Sprintf("%s/%s", fqSecretName, secretKey), values[key])
		if err != nil {
			return err
		}
	} else if secretKey != "" && key == "" {
		// if we have a property, and all the k8s secret fields, store it as a json obj.
		value, err := esutils.JSONMarshal(values)
		if err != nil {
			return err
		}
		err = conjurClient.AddSecret(fmt.Sprintf("%s/%s", fqSecretName, secretKey), string(value))
		if err != nil {
			return err
		}
	}
	return nil
}
