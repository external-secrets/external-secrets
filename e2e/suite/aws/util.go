/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
)

// CreateAWSSecretsManagerSecret creates a sm secret with the given value.
func CreateAWSSecretsManagerSecret(endpoint, secretName, secretValue string) error {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials("foobar", "foobar", "secret-manager"),
			EndpointResolver: auth.ResolveEndpointWithServiceMap(map[string]string{
				"secretsmanager": endpoint,
			}),
			Region: aws.String("eu-east-1"),
		},
	})
	if err != nil {
		return err
	}
	sm := secretsmanager.New(sess)
	_, err = sm.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(secretValue),
	})
	return err
}
