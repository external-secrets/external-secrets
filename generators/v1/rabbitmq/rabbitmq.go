// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// 2024 External Secrets Inc.
// All rights Reserved

// Package rabbitmq implements RabbitMQ user generator.
package rabbitmq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	rabbit "github.com/michaelklishin/rabbit-hole/v3"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"

	"github.com/external-secrets/external-secrets/generators/v1/password"
)

// Generator implements RabbitMQ user generation.
type Generator struct{}

const (
	errNoSpec    = "no spec provided"
	errParseSpec = "failed to parse spec: %w"
	errGetClient = "failed to get client: %w"
	errGetSecret = "failed to get secret: %w"
	errGetKey    = "no keys found with name: %v"

	errGenPassword = "failed to generate password: %w"
	errGetUser     = "failed to get user: %w"

	errPutUser = "failed to put user: %w"

	errGetPassword           = "failed to get password: %w"
	errInvalidPasswordPolicy = "invalid password policy"
	errInvalidAuthMethod     = "invalid auth method"
	errGetGenerator          = "failed to get generator: %w"
)

type pw struct {
	Password     string                  `json:"password"`
	PasswordHash string                  `json:"passwordHash"`
	Algorithm    rabbit.HashingAlgorithm `json:"algorithm"`
}

// Generate generates RabbitMQ user credentials.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kclient client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	client, err := getClient(ctx, res, kclient, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetClient, err)
	}
	newPassword, err := getOrGeneratePassword(ctx, res, kclient, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf(errGenPassword, err)
	}
	user, err := client.GetUser(res.Spec.Config.Username)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetUser, err)
	}
	if user.PasswordHash == newPassword.PasswordHash {
		return map[string][]byte{
			"password": []byte(newPassword.Password),
		}, nil, nil
	}
	resp, err := client.PutUser(res.Spec.Config.Username, rabbit.UserSettings{
		PasswordHash:     newPassword.PasswordHash,
		HashingAlgorithm: newPassword.Algorithm,
		Tags:             user.Tags,
	})
	if err != nil {
		return nil, nil, fmt.Errorf(errPutUser, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return map[string][]byte{
		"password": []byte(newPassword.Password),
	}, nil, nil
}

func getOrGeneratePassword(ctx context.Context, res *genv1alpha1.RabbitMQ, kclient client.Client, namespace string) (*pw, error) {
	if res.Spec.Config.PasswordPolicy.PasswordGeneratorRef != nil {
		return genFromGenerator(ctx, res, kclient, namespace)
	}
	if res.Spec.Config.PasswordPolicy.SecretRef != nil {
		return genFromSecret(ctx, res, kclient, namespace)
	}
	return nil, errors.New(errInvalidPasswordPolicy)
}

func genFromGenerator(ctx context.Context, res *genv1alpha1.RabbitMQ, kclient client.Client, namespace string) (*pw, error) {
	var generator genv1alpha1.Password
	err := kclient.Get(ctx, client.ObjectKey{Name: res.Spec.Config.PasswordPolicy.PasswordGeneratorRef.Name, Namespace: namespace}, &generator)
	if err != nil {
		return nil, fmt.Errorf(errGetGenerator, err)
	}
	// convert the generator to unstructured object
	u := &unstructured.Unstructured{}
	u.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(generator)
	if err != nil {
		return nil, err
	}
	jsonObj, err := u.MarshalJSON()
	if err != nil {
		return nil, err
	}
	spec := &apiextensions.JSON{Raw: jsonObj}
	p := password.Generator{}
	genPassword, _, err := p.Generate(ctx, spec, kclient, namespace)
	if err != nil {
		return nil, fmt.Errorf(errGenPassword, err)
	}
	hashPassword := rabbit.Base64EncodedSaltedPasswordHashSHA256(string(genPassword["password"]))
	return &pw{
		Password:     string(genPassword["password"]),
		PasswordHash: hashPassword,
		Algorithm:    rabbit.HashingAlgorithmSHA256,
	}, nil
}

func genFromSecret(ctx context.Context, res *genv1alpha1.RabbitMQ, kclient client.Client, namespace string) (*pw, error) {
	secret := v1.Secret{}
	err := kclient.Get(ctx, client.ObjectKey{Name: res.Spec.Config.PasswordPolicy.SecretRef.Name, Namespace: namespace}, &secret)
	if err != nil {
		return nil, fmt.Errorf(errGetSecret, err)
	}
	password, ok := secret.Data[res.Spec.Config.PasswordPolicy.SecretRef.Key]
	if !ok {
		return nil, fmt.Errorf(errGetKey, res.Spec.Config.PasswordPolicy.SecretRef.Key)
	}
	hashPassword := rabbit.Base64EncodedSaltedPasswordHashSHA256(string(password))
	return &pw{
		Password:     string(password),
		PasswordHash: hashPassword,
		Algorithm:    rabbit.HashingAlgorithmSHA256,
	}, nil
}
func getClient(ctx context.Context, res *genv1alpha1.RabbitMQ, kclient client.Client, namespace string) (*rabbit.Client, error) {
	if res.Spec.Auth.BasicAuth != nil {
		uri := fmt.Sprintf("%s:%v", res.Spec.Server.Host, res.Spec.Server.Port)
		var secretRef v1.Secret
		err := kclient.Get(ctx, client.ObjectKey{Name: res.Spec.Auth.BasicAuth.PasswordSecretRef.Name, Namespace: namespace}, &secretRef)
		if err != nil {
			return nil, fmt.Errorf(errGetSecret, err)
		}
		password, ok := secretRef.Data[res.Spec.Auth.BasicAuth.PasswordSecretRef.Key]
		if !ok {
			return nil, fmt.Errorf(errGetKey, res.Spec.Auth.BasicAuth.PasswordSecretRef.Key)
		}
		if res.Spec.Server.TLS {
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			}
			return rabbit.NewTLSClient(uri, res.Spec.Auth.BasicAuth.Username, string(password), transport)
		}
		return rabbit.NewClient(uri, res.Spec.Auth.BasicAuth.Username, string(password))
	}
	return nil, errors.New(errInvalidAuthMethod)
}

func parseSpec(data []byte) (*genv1alpha1.RabbitMQ, error) {
	var spec genv1alpha1.RabbitMQ
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// Cleanup cleans up generated RabbitMQ users.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

// GetCleanupPolicy returns the cleanup policy for this generator.
func (g *Generator) GetCleanupPolicy(_ *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	return nil, nil
}

// LastActivityTime returns the last activity time for generated resources.
func (g *Generator) LastActivityTime(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

// GetKeys returns the keys generated by this generator.
func (g *Generator) GetKeys() map[string]string {
	return map[string]string{
		"password": "Generated password for the RabbitMQ user",
	}
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return genv1alpha1.RabbitMQGeneratorKind
}
