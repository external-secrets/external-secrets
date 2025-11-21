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

// Package sendgrid implements SendGrid API key generator.
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
package sendgrid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sendgrid/rest"
	sendgridapi "github.com/sendgrid/sendgrid-go"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errNoSpec               = "no config spec provided"
	errParseSpec            = "unable to parse spec: %w"
	errFetchSecretRef       = "could not fetch secret ref: %w"
	errDeleteAPIKey         = "failed to delete existing API key: %w"
	errCreateAPIKey         = "failed to create new API key: %w"
	errGetAPIKeys           = "failed to get API keys: %w"
	errBuildPayload         = "failed to build payload: %w"
	errProcessResponse      = "failed to process response: %w"
	errBuildRequest         = "failed to build SendGrid request: %w"
	errMissingPreviousState = "missing previous state"
)

// SecretKey is a SendGrid API key.
type SecretKey struct {
	ID     string   `json:"api_key_id,omitempty"`
	Key    string   `json:"api_key,omitempty"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes,omitempty"`
}

// State is the state of a SendGrid API key.
type State struct {
	APIKeyID   string `json:"apiKeyID,omitempty"`
	APIKeyName string `json:"apiKeyName,omitempty"`
}

// Client is a SendGrid client interface.
type Client interface {
	API(request rest.Request) (*rest.Response, error)
	GetRequest(apiKey, endpoint, host string) rest.Request
	SetDataResidency(request rest.Request, dataResidency string) (rest.Request, error)
}

// Impl is a SendGrid client implementation.
type Impl struct{}

// API sends a request to SendGrid.
func (c *Impl) API(request rest.Request) (*rest.Response, error) {
	return sendgridapi.API(request)
}

// GetRequest returns a new request to SendGrid.
func (c *Impl) GetRequest(apiKey, endpoint, host string) rest.Request {
	return sendgridapi.GetRequest(apiKey, endpoint, host)
}

// SetDataResidency sets the data residency for the request.
func (c *Impl) SetDataResidency(request rest.Request, dataResidency string) (rest.Request, error) {
	return sendgridapi.SetDataResidency(request, dataResidency)
}

// Generator implements SendGrid API key generation.
type Generator struct{}

// Generate generates SendGrid API keys.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	client := &Impl{}
	return g.generate(ctx, jsonSpec, kube, namespace, client)
}

func (g *Generator) generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string, client Client) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	secretName := fmt.Sprintf("Managed By ESO Generator: %s %s", res.ObjectMeta.Name, res.ObjectMeta.UID)

	dataResidency := res.Spec.DataResidency
	apiKey, err := getFromSecretRef(ctx, &res.Spec.Auth.SecretRef.APIKey, "", kube, namespace)
	if err != nil {
		return nil, nil, err
	}

	createdSecret, err := g.createAPIKey(secretName, res.Spec.Scopes, apiKey, dataResidency, client)
	if err != nil {
		return nil, nil, err
	}
	stateData := State{
		APIKeyID:   createdSecret.ID,
		APIKeyName: createdSecret.Name,
	}
	rawState, err := json.Marshal(stateData)
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		"apiKey": []byte(createdSecret.Key),
	}, &apiextensions.JSON{Raw: rawState}, nil
}

func (g *Generator) buildSendGridRequest(apiKey, dataResidency string, method rest.Method, endpoint string, client Client) (rest.Request, error) {
	request := client.GetRequest(apiKey, endpoint, "")
	request.Method = method
	request, err := client.SetDataResidency(request, dataResidency)
	if err != nil {
		return request, fmt.Errorf(errBuildRequest, err)
	}
	return request, nil
}

func (g *Generator) deleteAPIKey(keyID, apiKey, dataResidency string, client Client) error {
	path := fmt.Sprintf("/v3/api_keys/%s", keyID)
	deleteRequest, err := g.buildSendGridRequest(apiKey, dataResidency, rest.Delete, path, client)
	if err != nil {
		return fmt.Errorf(errBuildRequest, err)
	}
	if _, err := client.API(deleteRequest); err != nil {
		return fmt.Errorf(errDeleteAPIKey, err)
	}
	return nil
}

func (g *Generator) createAPIKey(secretName string, scopes []string, apiKey, dataResidency string, client Client) (SecretKey, error) {
	createAPIKeyRequest, err := g.buildSendGridRequest(apiKey, dataResidency, rest.Post, "/v3/api_keys", client)
	if err != nil {
		return SecretKey{}, fmt.Errorf(errBuildRequest, err)
	}

	apiKeyData := SecretKey{
		Name:   secretName,
		Scopes: scopes,
	}
	body, err := json.Marshal(apiKeyData)
	if err != nil {
		return SecretKey{}, fmt.Errorf(errBuildPayload, err)
	}
	createAPIKeyRequest.Body = body

	response, err := client.API(createAPIKeyRequest)
	if err != nil {
		return SecretKey{}, fmt.Errorf(errCreateAPIKey, err)
	}

	var createdSecret SecretKey
	if err := json.Unmarshal([]byte(response.Body), &createdSecret); err != nil {
		return SecretKey{}, fmt.Errorf(errProcessResponse, err)
	}

	return createdSecret, nil
}

func parseSpec(data []byte) (*genv1alpha1.SendgridAuthorizationToken, error) {
	var spec genv1alpha1.SendgridAuthorizationToken

	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseState(data []byte) (*State, error) {
	var state State
	err := json.Unmarshal(data, &state)
	return &state, err
}

func getFromSecretRef(ctx context.Context, keySelector *esmeta.SecretKeySelector, storeKind string, kube client.Client, namespace string) (string, error) {
	value, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, keySelector)
	if err != nil {
		return "", fmt.Errorf(errFetchSecretRef, err)
	}

	return value, err
}

func (g *Generator) cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, previousState genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string, client Client) error {
	if jsonSpec == nil {
		return errors.New(errNoSpec)
	}

	if previousState == nil {
		return errors.New(errMissingPreviousState)
	}

	status, err := parseState(previousState.Raw)
	if err != nil {
		return err
	}
	gen, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return err
	}
	apiKey, err := getFromSecretRef(ctx, &gen.Spec.Auth.SecretRef.APIKey, "", kclient, namespace)
	if err != nil {
		return err
	}

	err = g.deleteAPIKey(status.APIKeyID, apiKey, gen.Spec.DataResidency, client)
	if err != nil {
		return err
	}
	return nil
}

// Cleanup cleans up generated SendGrid API keys.
func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, previousState genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	client := &Impl{}
	return g.cleanup(ctx, jsonSpec, previousState, kclient, namespace, client)
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
		"apiKey": "SendGrid API key for authenticated API requests",
	}
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return genv1alpha1.SendgridKind
}
