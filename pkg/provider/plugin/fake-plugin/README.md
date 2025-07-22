# Fake Plugin Example

This is a comprehensive example of an External Secrets Operator plugin implementation. It demonstrates how to build a plugin that implements all the required gRPC interfaces for the External Secrets Operator plugin system.

## Overview

The fake plugin is a simple in-memory secret store that pre-populates some example secrets and supports all CRUD operations. It's designed for testing, development, and as a reference implementation for building your own plugins.

## Features

This plugin implements all the External Secrets Operator plugin interfaces:

- **GetInfo**: Returns plugin metadata and capabilities
- **GetSecret**: Retrieves a single secret by key
- **GetSecretMap**: Retrieves multiple key-value pairs from a JSON secret
- **GetAllSecrets**: Retrieves all secrets matching path/regex filters
- **PushSecret**: Stores a secret in the plugin
- **DeleteSecret**: Removes a secret from the plugin  
- **SecretExists**: Checks if a secret exists
- **Validate**: Validates plugin configuration

## Capabilities

The plugin reports `CAPABILITY_READ_WRITE`, meaning it supports both reading and writing secrets.

## Pre-populated Secrets

The plugin starts with these example secrets:

```
example/username = "admin"
example/password = "supersecret123" 
example/config = {"host":"localhost","port":8080,"ssl":true}
example/api-key = "fake-api-key-12345"
database/host = "db.example.com"
database/port = "5432"
database/credentials = {"username":"dbuser","password":"dbpass123"}
```

## Building

```bash
cd example/fake-plugin
go mod download
go build -o fake-plugin main.go
```

## Running

### Unix Socket (default)
```bash
./fake-plugin
# Listens on unix:///tmp/fake-plugin.sock
```

### TCP
```bash
./fake-plugin -endpoint="tcp://localhost:9999"
# Listens on localhost:9999
```

### Custom Socket Path
```bash
./fake-plugin -endpoint="/path/to/custom/socket.sock"
```

### With Custom Name/Version
```bash
./fake-plugin -name="my-plugin" -version="2.0.0"
```

## Testing with External Secrets

### 1. Create a SecretStore

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: fake-plugin-store
  namespace: default
spec:
  provider:
    plugin:
      endpoint: "unix:///tmp/fake-plugin.sock"
      timeout: "10s"
```

### 2. Create an ExternalSecret

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: fake-plugin-secret
  namespace: default
spec:
  refreshInterval: 15s
  secretStoreRef:
    name: fake-plugin-store
    kind: SecretStore
  target:
    name: my-secret
    creationPolicy: Owner
  data:
  - secretKey: username
    remoteRef:
      key: example/username
  - secretKey: password
    remoteRef:
      key: example/password
  - secretKey: host
    remoteRef:
      key: example/config
      property: host
```

### 3. Using dataFrom to get all keys from a JSON secret

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: fake-plugin-config
  namespace: default
spec:
  refreshInterval: 15s
  secretStoreRef:
    name: fake-plugin-store
    kind: SecretStore
  target:
    name: config-secret
    creationPolicy: Owner
  dataFrom:
  - sourceRef:
      key: example/config
```

### 4. Using find to get multiple secrets

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: fake-plugin-database
  namespace: default
spec:
  refreshInterval: 15s
  secretStoreRef:
    name: fake-plugin-store
    kind: SecretStore
  target:
    name: database-secrets
    creationPolicy: Owner
  dataFrom:
  - find:
      path: database/
```

## Plugin Development Guide

This example demonstrates several important concepts for plugin development:

### 1. Thread Safety
The plugin uses `sync.RWMutex` to protect concurrent access to the in-memory secret store.

### 2. Property Extraction
The `GetSecret` method supports extracting specific properties from JSON secrets using the `property` field.

### 3. Secret Map Parsing
The `GetSecretMap` method can parse JSON secrets and return all key-value pairs.

### 4. Path Filtering
The `GetAllSecrets` method supports filtering by path prefix and regex patterns.

### 5. Error Handling
All methods include appropriate error handling and return meaningful error messages.

### 6. Metadata
Response messages include metadata like timestamps and source information.

### 7. Endpoint Parsing
The plugin supports both Unix socket and TCP endpoints with flexible parsing.

### 8. Graceful Shutdown
The server handles SIGINT and SIGTERM signals for graceful shutdown.

## Implementation Notes

### Protocol Buffers
The plugin uses the proto definitions from `github.com/external-secrets/external-secrets/pkg/provider/plugin/proto` which define the gRPC service interface.

### Capabilities
The plugin reports its capabilities through the `GetInfo` method. This tells External Secrets Operator what operations the plugin supports.

### Configuration Validation
The `Validate` method can be used to check plugin-specific configuration. This fake plugin accepts any configuration.

### Logging
The plugin includes basic logging to help with debugging and monitoring.

## Extending This Example

To create your own plugin based on this example:

1. Replace the in-memory storage with your actual secret backend
2. Implement authentication using the `auth` fields in requests
3. Add configuration validation logic in the `Validate` method
4. Add your own metadata and error handling
5. Update the plugin name, version, and capabilities

## Troubleshooting

### Permission Issues
Make sure the socket path is writable by the plugin process and readable by the External Secrets Operator.

### Connection Issues
Check that the endpoint configuration matches between the plugin and the SecretStore.

### Secret Not Found Errors
Verify the secret key exists using the `SecretExists` operation or check the plugin logs.
