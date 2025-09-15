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

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/external-secrets/external-secrets/pkg/provider/plugin/proto"
	"google.golang.org/grpc"
)

var (
	endpoint = flag.String("endpoint", "unix:///tmp/fake-plugin.sock", "Plugin endpoint (unix socket or tcp address)")
	version  = flag.String("version", "1.0.0", "Plugin version")
	name     = flag.String("name", "fake-plugin", "Plugin name")
)

// FakeSecretPlugin implements a fake secrets plugin for testing and demonstration
type FakeSecretPlugin struct {
	proto.UnimplementedSecretsPluginServiceServer
	
	// In-memory storage for secrets
	secrets map[string][]byte
	mutex   sync.RWMutex
}

// NewFakeSecretPlugin creates a new instance of the fake plugin
func NewFakeSecretPlugin() *FakeSecretPlugin {
	plugin := &FakeSecretPlugin{
		secrets: make(map[string][]byte),
	}
	
	// Pre-populate with some example secrets
	plugin.secrets["example/username"] = []byte("admin")
	plugin.secrets["example/password"] = []byte("supersecret123")
	plugin.secrets["example/config"] = []byte(`{"host":"localhost","port":8080,"ssl":true}`)
	plugin.secrets["example/api-key"] = []byte("fake-api-key-12345")
	plugin.secrets["database/host"] = []byte("db.example.com")
	plugin.secrets["database/port"] = []byte("5432")
	plugin.secrets["database/credentials"] = []byte(`{"username":"dbuser","password":"dbpass123"}`)
	
	return plugin
}

// GetInfo returns plugin information and capabilities
func (p *FakeSecretPlugin) GetInfo(ctx context.Context, req *proto.GetInfoRequest) (*proto.GetInfoResponse, error) {
	log.Printf("GetInfo called")
	
	return &proto.GetInfoResponse{
		Name:    *name,
		Version: *version,
		Capabilities: []proto.Capability{
			proto.Capability_CAPABILITY_READ_WRITE, // This plugin supports both read and write operations
		},
		Metadata: map[string]string{
			"description": "A fake plugin for testing and demonstration purposes",
			"provider":    "fake",
			"author":      "external-secrets-community",
		},
	}, nil
}

// GetSecret retrieves a single secret from the plugin
func (p *FakeSecretPlugin) GetSecret(ctx context.Context, req *proto.GetSecretRequest) (*proto.GetSecretResponse, error) {
	log.Printf("GetSecret called for key: %s", req.Key)
	
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	value, exists := p.secrets[req.Key]
	if !exists {
		return nil, fmt.Errorf("secret not found: %s", req.Key)
	}
	
	// If a property is specified, try to extract it from JSON
	if req.Property != "" {
		var jsonData map[string]interface{}
		if err := json.Unmarshal(value, &jsonData); err != nil {
			return nil, fmt.Errorf("failed to parse secret as JSON for property extraction: %w", err)
		}
		
		if propValue, exists := jsonData[req.Property]; exists {
			if propBytes, err := json.Marshal(propValue); err == nil {
				// Remove quotes if it's a simple string
				if strings.HasPrefix(string(propBytes), `"`) && strings.HasSuffix(string(propBytes), `"`) {
					propBytes = propBytes[1 : len(propBytes)-1]
				}
				value = propBytes
			} else {
				return nil, fmt.Errorf("failed to marshal property value: %w", err)
			}
		} else {
			return nil, fmt.Errorf("property not found: %s", req.Property)
		}
	}
	
	return &proto.GetSecretResponse{
		Value: value,
		Metadata: map[string]string{
			"retrieved_at": time.Now().Format(time.RFC3339),
			"source":       "fake-plugin",
		},
	}, nil
}

// GetSecretMap retrieves multiple key-value pairs from the plugin
func (p *FakeSecretPlugin) GetSecretMap(ctx context.Context, req *proto.GetSecretMapRequest) (*proto.GetSecretMapResponse, error) {
	log.Printf("GetSecretMap called for key: %s", req.Key)
	
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	value, exists := p.secrets[req.Key]
	if !exists {
		return nil, fmt.Errorf("secret not found: %s", req.Key)
	}
	
	// Try to parse as JSON to extract multiple values
	var jsonData map[string]interface{}
	if err := json.Unmarshal(value, &jsonData); err != nil {
		// If not JSON, return the single value
		return &proto.GetSecretMapResponse{
			Values: map[string][]byte{
				"value": value,
			},
			Metadata: map[string]string{
				"retrieved_at": time.Now().Format(time.RFC3339),
				"source":       "fake-plugin",
			},
		}, nil
	}
	
	// Convert JSON to map[string][]byte
	values := make(map[string][]byte)
	for k, v := range jsonData {
		if valueBytes, err := json.Marshal(v); err == nil {
			// Remove quotes if it's a simple string
			if strings.HasPrefix(string(valueBytes), `"`) && strings.HasSuffix(string(valueBytes), `"`) {
				valueBytes = valueBytes[1 : len(valueBytes)-1]
			}
			values[k] = valueBytes
		}
	}
	
	return &proto.GetSecretMapResponse{
		Values: values,
		Metadata: map[string]string{
			"retrieved_at": time.Now().Format(time.RFC3339),
			"source":       "fake-plugin",
		},
	}, nil
}

// GetAllSecrets retrieves all secrets matching the given criteria
func (p *FakeSecretPlugin) GetAllSecrets(ctx context.Context, req *proto.GetAllSecretsRequest) (*proto.GetAllSecretsResponse, error) {
	log.Printf("GetAllSecrets called for path: %s, name_regex: %s", req.Path, req.NameRegex)
	
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	values := make(map[string][]byte)
	
	// Compile regex if provided
	var nameRegex *regexp.Regexp
	if req.NameRegex != "" {
		var err error
		nameRegex, err = regexp.Compile(req.NameRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	
	// Filter secrets based on path and regex
	for key, value := range p.secrets {
		// Check path filter
		if req.Path != "" && !strings.HasPrefix(key, req.Path) {
			continue
		}
		
		// Check regex filter
		if nameRegex != nil {
			// Extract the name part (after last /)
			name := key
			if lastSlash := strings.LastIndex(key, "/"); lastSlash >= 0 {
				name = key[lastSlash+1:]
			}
			if !nameRegex.MatchString(name) {
				continue
			}
		}
		
		values[key] = value
	}
	
	return &proto.GetAllSecretsResponse{
		Values: values,
		Metadata: map[string]string{
			"retrieved_at": time.Now().Format(time.RFC3339),
			"source":       "fake-plugin",
			"total_count":  fmt.Sprintf("%d", len(values)),
		},
	}, nil
}

// PushSecret writes a secret to the plugin
func (p *FakeSecretPlugin) PushSecret(ctx context.Context, req *proto.PushSecretRequest) (*proto.PushSecretResponse, error) {
	log.Printf("PushSecret called for key: %s", req.Key)
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// Store the secret
	p.secrets[req.Key] = req.Value
	
	return &proto.PushSecretResponse{
		Success: true,
		Message: fmt.Sprintf("Secret '%s' stored successfully", req.Key),
	}, nil
}

// DeleteSecret deletes a secret from the plugin
func (p *FakeSecretPlugin) DeleteSecret(ctx context.Context, req *proto.DeleteSecretRequest) (*proto.DeleteSecretResponse, error) {
	log.Printf("DeleteSecret called for key: %s", req.Key)
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if _, exists := p.secrets[req.Key]; !exists {
		return &proto.DeleteSecretResponse{
			Success: false,
			Message: fmt.Sprintf("Secret '%s' not found", req.Key),
		}, nil
	}
	
	delete(p.secrets, req.Key)
	
	return &proto.DeleteSecretResponse{
		Success: true,
		Message: fmt.Sprintf("Secret '%s' deleted successfully", req.Key),
	}, nil
}

// SecretExists checks if a secret exists in the plugin
func (p *FakeSecretPlugin) SecretExists(ctx context.Context, req *proto.SecretExistsRequest) (*proto.SecretExistsResponse, error) {
	log.Printf("SecretExists called for key: %s", req.Key)
	
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	_, exists := p.secrets[req.Key]
	
	return &proto.SecretExistsResponse{
		Exists: exists,
	}, nil
}

// Validate validates the plugin configuration
func (p *FakeSecretPlugin) Validate(ctx context.Context, req *proto.ValidateRequest) (*proto.ValidateResponse, error) {
	log.Printf("Validate called with config: %+v", req.Config)
	
	// This fake plugin accepts any configuration
	return &proto.ValidateResponse{
		Valid:   true,
		Message: "Configuration is valid",
	}, nil
}

// generateRandomSecret generates a random secret for testing
func (p *FakeSecretPlugin) generateRandomSecret() []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		randByte := make([]byte, 1)
		rand.Read(randByte)
		b[i] = charset[randByte[0]%byte(len(charset))]
	}
	return b
}

func main() {
	flag.Parse()
	
	log.Printf("Starting fake plugin server...")
	log.Printf("Name: %s", *name)
	log.Printf("Version: %s", *version)
	log.Printf("Endpoint: %s", *endpoint)
	
	// Parse the endpoint
	var network, address string
	if strings.HasPrefix(*endpoint, "unix://") {
		network = "unix"
		address = strings.TrimPrefix(*endpoint, "unix://")
		
		// Ensure the directory exists
		dir := filepath.Dir(address)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create socket directory: %v", err)
		}
		
		// Remove existing socket file
		os.Remove(address)
	} else if strings.HasPrefix(*endpoint, "tcp://") {
		network = "tcp"
		address = strings.TrimPrefix(*endpoint, "tcp://")
	} else if strings.Contains(*endpoint, ":") {
		// Assume host:port format
		network = "tcp"
		address = *endpoint
	} else {
		// Assume unix socket path
		network = "unix"
		address = *endpoint
		
		// Ensure the directory exists
		dir := filepath.Dir(address)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create socket directory: %v", err)
		}
		
		// Remove existing socket file
		os.Remove(address)
	}
	
	// Create listener
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("Failed to listen on %s:%s: %v", network, address, err)
	}
	defer listener.Close()
	
	log.Printf("Listening on %s:%s", network, address)
	
	// Create gRPC server
	server := grpc.NewServer()
	
	// Register the plugin service
	plugin := NewFakeSecretPlugin()
	proto.RegisterSecretsPluginServiceServer(server, plugin)
	
	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping server...")
		server.GracefulStop()
	}()
	
	log.Println("Fake plugin server is ready to accept connections")
	
	// Start serving
	if err := server.Serve(listener); err != nil {
		log.Printf("Server stopped: %v", err)
	}
	
	log.Println("Server shutdown complete")
}
