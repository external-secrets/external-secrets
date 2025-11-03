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

// Package sshkey provides functionality for generating SSH key pairs.
package sshkey

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// Generator implements SSH key pair generation functionality.
type Generator struct{}

const (
	defaultKeyType = "rsa"
	defaultKeySize = 2048

	errNoSpec      = "no config spec provided"
	errParseSpec   = "unable to parse spec: %w"
	errGenerateKey = "unable to generate SSH key: %w"
	errUnsupported = "unsupported key type: %s"
)

type generateFunc func(keyType string, keySize *int, comment string) (privateKey, publicKey []byte, err error)

// Generate creates a new SSH key pair.
func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		jsonSpec,
		generateSSHKey,
	)
}

// Cleanup performs any necessary cleanup after key generation.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(jsonSpec *apiextensions.JSON, keyGen generateFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	keyType := defaultKeyType
	if res.Spec.KeyType != "" {
		keyType = res.Spec.KeyType
	}

	privateKey, publicKey, err := keyGen(keyType, res.Spec.KeySize, res.Spec.Comment)
	if err != nil {
		return nil, nil, fmt.Errorf(errGenerateKey, err)
	}

	return map[string][]byte{
		"privateKey": privateKey,
		"publicKey":  publicKey,
	}, nil, nil
}

func generateSSHKey(keyType string, keySize *int, comment string) (privateKey, publicKey []byte, err error) {
	switch keyType {
	case "rsa":
		bits := 2048
		if keySize != nil {
			bits = *keySize
		}
		return generateRSAKey(bits, comment)
	case "ed25519":
		return generateEd25519Key(comment)
	default:
		return nil, nil, fmt.Errorf(errUnsupported, keyType)
	}
}

func generateRSAKey(keySize int, comment string) (privateKey, publicKey []byte, err error) {
	// Generate RSA private key
	rsaKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, err
	}

	// Create SSH private key in OpenSSH format
	sshPrivateKey, err := ssh.MarshalPrivateKey(rsaKey, comment)
	if err != nil {
		return nil, nil, err
	}

	// Create SSH public key
	sshPublicKey, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(sshPublicKey)
	if comment != "" {
		// Remove the newline and add comment
		publicKeyStr := string(publicKeyBytes[:len(publicKeyBytes)-1]) + " " + comment + "\n"
		publicKeyBytes = []byte(publicKeyStr)
	}

	return pem.EncodeToMemory(sshPrivateKey), publicKeyBytes, nil
}

func generateEd25519Key(comment string) (privateKey, publicKey []byte, err error) {
	// Generate Ed25519 private key
	_, ed25519PrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Create SSH private key in OpenSSH format
	sshPrivateKey, err := ssh.MarshalPrivateKey(ed25519PrivateKey, comment)
	if err != nil {
		return nil, nil, err
	}

	// Create SSH public key
	sshPublicKey, err := ssh.NewPublicKey(ed25519PrivateKey.Public())
	if err != nil {
		return nil, nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(sshPublicKey)
	if comment != "" {
		// Remove the newline and add comment
		publicKeyStr := string(publicKeyBytes[:len(publicKeyBytes)-1]) + " " + comment + "\n"
		publicKeyBytes = []byte(publicKeyStr)
	}

	return pem.EncodeToMemory(sshPrivateKey), publicKeyBytes, nil
}

func parseSpec(data []byte) (*genv1alpha1.SSHKey, error) {
	var spec genv1alpha1.SSHKey
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}


// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindSSHKey)
}
