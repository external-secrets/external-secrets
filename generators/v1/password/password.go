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

// Package password provides functionality for generating secure random passwords.
package password

import (
	"context"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/sethvargo/go-password/password"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// Generator implements secure random password generation functionality.
type Generator struct{}

const (
	defaultLength      = 24
	defaultSymbolChars = "~!@#$%^&*()_+`-={}|[]\\:\"<>?,./"
	digitFactor        = 0.25
	symbolFactor       = 0.25

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"
	errSecretKey = "secretKeys must be non-empty and unique"
)

type generateFunc func(
	length int,
	symbols int,
	symbolCharacters string,
	digits int,
	noUpper bool,
	allowRepeat bool,
) (string, error)

// Generate creates a secure random password based on the provided configuration.
func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		jsonSpec,
		generateSafePassword,
	)
}

// Cleanup performs any necessary cleanup after password generation.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(jsonSpec *apiextensions.JSON, passGen generateFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	config := extractPasswordConfig(res)
	keys, err := validateSecretKeys(res.Spec.SecretKeys)
	if err != nil {
		return nil, nil, err
	}

	passwords, err := generatePasswords(keys, config, passGen)
	if err != nil {
		return nil, nil, err
	}

	return passwords, nil, nil
}

type passwordConfig struct {
	length           int
	digits           int
	symbols          int
	symbolCharacters string
	encoding         string
	noUpper          bool
	allowRepeat      bool
}

func extractPasswordConfig(res *genv1alpha1.Password) passwordConfig {
	config := passwordConfig{
		symbolCharacters: defaultSymbolChars,
		length:           defaultLength,
		encoding:         "raw",
	}

	if res.Spec.SymbolCharacters != nil {
		config.symbolCharacters = *res.Spec.SymbolCharacters
	}
	if res.Spec.Length > 0 {
		config.length = res.Spec.Length
	}
	config.digits = int(float32(config.length) * digitFactor)
	if res.Spec.Digits != nil {
		config.digits = *res.Spec.Digits
	}
	config.symbols = int(float32(config.length) * symbolFactor)
	if res.Spec.Symbols != nil {
		config.symbols = *res.Spec.Symbols
	}
	if res.Spec.Encoding != nil {
		config.encoding = *res.Spec.Encoding
	}
	config.noUpper = res.Spec.NoUpper
	config.allowRepeat = res.Spec.AllowRepeat

	return config
}

func validateSecretKeys(keys []string) ([]string, error) {
	if len(keys) == 0 {
		keys = []string{"password"}
	}
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == "" {
			return nil, errors.New(errSecretKey)
		}
		if _, ok := seen[key]; ok {
			return nil, errors.New(errSecretKey)
		}
		seen[key] = struct{}{}
	}
	return keys, nil
}

func generatePasswords(keys []string, config passwordConfig, passGen generateFunc) (map[string][]byte, error) {
	passwords := make(map[string][]byte, len(keys))
	for _, key := range keys {
		pass, err := passGen(
			config.length,
			config.symbols,
			config.symbolCharacters,
			config.digits,
			config.noUpper,
			config.allowRepeat,
		)
		if err != nil {
			return nil, err
		}
		passwords[key] = encodePassword([]byte(pass), config.encoding)
	}
	return passwords, nil
}

func generateSafePassword(
	passLen int,
	symbols int,
	symbolCharacters string,
	digits int,
	noUpper bool,
	allowRepeat bool,
) (string, error) {
	gen, err := password.NewGenerator(&password.GeneratorInput{
		Symbols: symbolCharacters,
	})
	if err != nil {
		return "", err
	}
	return gen.Generate(
		passLen,
		digits,
		symbols,
		noUpper,
		allowRepeat,
	)
}

func encodePassword(b []byte, encoding string) []byte {
	var encodedString string
	switch encoding {
	case "base64url":
		encodedString = base64.URLEncoding.EncodeToString(b)
	case "raw":
		return b
	case "base32":
		encodedString = base32.StdEncoding.EncodeToString(b)
	case "hex":
		encodedString = hex.EncodeToString(b)
	case "base64":
		encodedString = base64.StdEncoding.EncodeToString(b)
	default:
		return b
	}
	return []byte(encodedString)
}

func parseSpec(data []byte) (*genv1alpha1.Password, error) {
	var spec genv1alpha1.Password
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindPassword)
}
