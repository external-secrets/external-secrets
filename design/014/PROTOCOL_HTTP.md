# ESO ⇄ Provider Contract: HTTP Binding

How the contract in [PROTOCOL.md](./PROTOCOL.md) travels as JSON over HTTP. A
runtime config selects it with `spec.protocol: http`. It is reserved: until the core
implements it, the runtime config controller sets `Ready=False` with reason
`ProtocolNotSupported`. This document adds no meaning to the contract; where
the two disagree, PROTOCOL.md is right.

## Why a second binding

The audience is providers built without gRPC tooling: a small service in a
language whose gRPC story is weak, a shell around an existing HTTP API, or a
prototype someone wants to `curl`. Everything such a provider needs is an HTTP
server and a JSON library. The bar this binding must clear is that a provider
can implement it by hand, from this document alone, without generated code.

## Choice: Connect's unary JSON protocol

The binding is the unary part of the [Connect protocol](https://connectrpc.com/docs/protocol)
with JSON payloads:

- every operation is `POST /<package>.<Service>/<Operation>`;
- the body is the request message in proto3 JSON;
- a success is `200` with the response message;
- an error is a non-`200` status with a small JSON error object.

Alternatives considered:

- **A REST mapping** (`GET /v1alpha1/secrets/{key}`, through
  `google.api.http` annotations and grpc-gateway). It needs per-operation URL
  design, and the request is not a resource: `providerConfig` and `context` do
  not fit a path or a query string. It is a second contract to review, and the
  generic contract would leak into URL templates.
- **Our own JSON-over-HTTP envelope.** It would be equivalent to Connect for
  unary calls, but without existing client and server libraries
  (`connect-go`, `connect-es`, `connect-python`, ...), conformance tests or
  tooling. Choosing Connect costs nothing for a hand-written provider and saves
  everyone else the work.
- **gRPC-Web or gRPC-JSON transcoding through a proxy.** Both require
  something in the path that the core does not ship.

The core implements this binding with `connect-go` and the same generated
types as the gRPC binding, so the two bindings share one data model by
construction (P10).

## Summary

| Contract concept | HTTP encoding |
| --- | --- |
| contract version `v1alpha1` | path segment `externalsecrets.provider.v1alpha1` |
| operation address | `POST /externalsecrets.provider.v1alpha1.<Service>/<Operation>`, with `<Service>` one of `Runtime`, `SecretStoreProvider`, `GeneratorProvider` |
| health `Check` | `POST /grpc.health.v1.Health/Check` |
| messages | proto3 JSON (rules below), `Content-Type: application/json` |
| success | `200 OK`, body is the response message |
| error code | `code` in the error body; HTTP status as in the table below |
| error detail | `details[]` entry of type `externalsecrets.provider.v1alpha1.ProviderError` |
| request id | header `X-Eso-Request-Id` |
| deadline | header `Connect-Timeout-Ms` |
| bearer token *(reserved)* | header `Authorization: Bearer <token>` |
| protocol marker | header `Connect-Protocol-Version: 1` (sent always; providers may ignore it) |
| message size limit (16 MiB) | request and response bodies; a larger request may be rejected with `413` (maps to `Unknown`) |
| HTTP version | HTTP/1.1 or HTTP/2, over TLS (ALPN `h2` or `http/1.1`), cleartext in `HTTP(dev)`, or a Unix socket |

The core always sends `POST` with JSON. Connect's `GET` form for side-effect-free
calls and its `application/proto` encoding are not used: a provider that can
produce protobuf should use the gRPC binding.

## JSON encoding

These are the canonical proto3 JSON mapping rules, stated here so that a
hand-written provider does not need the protobuf specification.

| Proto type | JSON | Example |
| --- | --- | --- |
| message | object | `{"remoteRef": {"key": "/a"}}` |
| field name | lowerCamelCase of the proto name; receivers must also accept the snake_case name | `provider_config` → `providerConfig` |
| `string` | string | `"team-a"` |
| `bool` | `true` / `false` | |
| `int64` (`generation`, `providerGeneration`) | **string** of decimal digits; receivers must also accept a number | `"7"` |
| `bytes` (`value`, `secretData` values, `metadata`) | **standard base64** with padding; receivers must also accept unpadded and URL-safe | `"czNjcjN0LXA0c3M="` |
| enum | value name as string; receivers must also accept the number | `"VALIDATION_RESULT_VALID"` |
| `repeated T` | array | `["team-b"]` |
| `map<string, T>` | object | `{"team": "payments"}` |

Rules:

- A field with its default value (empty string, `0`, `false`, empty list or
  map, absent message) may be omitted, and an omitted field means the default.
  `null` also means the default.
- **Unknown fields must be ignored** by both sides. This is what makes additive
  changes within a version safe (P5). In Go, `protojson.UnmarshalOptions{DiscardUnknown: true}`,
  which `connect-go` already sets.
- Map keys and string values are UTF-8. Secret bytes are always `bytes` and
  therefore always base64, never a JSON string.

## Errors

An error response has a non-`200` status, `Content-Type: application/json`, and
this body:

```json
{
  "code": "failed_precondition",
  "message": "human-readable message",
  "details": [
    {
      "type": "externalsecrets.provider.v1alpha1.ProviderError",
      "value": "<base64 of the protobuf-encoded ProviderError>",
      "debug": { "code": "ExplicitNamespaceNotAllowed", "message": "..." }
    }
  ]
}
```

- `code` is required. The core decides on `code`, not on the HTTP status.
- `message` is optional.
- `details` is optional (P3). When present, `value` is the protobuf encoding of
  `ProviderError` as Connect specifies it, and the core reads it. `debug` is the
  same message in JSON, for humans and logs. The core does not read it.
- A provider that cannot produce protobuf omits `details`. It loses only
  structured detail: every core outcome depends on `code` alone.

| Contract code | `code` | HTTP status |
| --- | --- | --- |
| `NotFound` | `not_found` | 404 |
| `InvalidArgument` | `invalid_argument` | 400 |
| `FailedPrecondition` | `failed_precondition` | 400 |
| `PermissionDenied` | `permission_denied` | 403 |
| `Unauthenticated` | `unauthenticated` | 401 |
| `Unimplemented` | `unimplemented` | 501 |
| `AlreadyExists` | `already_exists` | 409 |
| `ResourceExhausted` | `resource_exhausted` | 429 |
| `Unavailable` | `unavailable` | 503 |
| `DeadlineExceeded` | `deadline_exceeded` | 504 |
| `Internal` | `internal` | 500 |
| `Unknown` | `unknown`, and any value not listed | 500 |

A non-`200` response **without** a Connect error body (a proxy page, a
framework default) is mapped from its status code, as Connect specifies:

| Status | Contract code |
| --- | --- |
| 400 | `Internal` |
| 401 | `Unauthenticated` |
| 403 | `PermissionDenied` |
| 404 | `Unimplemented` |
| 429, 502, 503, 504 | `Unavailable` |
| anything else | `Unknown` |

A `404` from a path the provider does not route therefore becomes
`Unimplemented`. That is exactly what version negotiation (P5) expects when the
core asks for a version the runtime does not serve.

## Headers, deadline, connections

- `X-Eso-Request-Id`: always set by the core.
- `Connect-Timeout-Ms`: always set, from `RuntimeConfig.spec.timeout` (`30000` by
  default). A provider should stop work after it. The core also enforces it on
  its side.
- `Authorization: Bearer <projected token>`: only with
  `auth.serviceAccountToken` *(reserved)*. Never sent without TLS (CEL already
  forbids `auth` without TLS).
- `Content-Encoding`: the core does not compress requests. A provider may gzip a
  response when the request's `Accept-Encoding` allows it.
- No other header is part of the contract (P11). Providers must not depend on
  `User-Agent` or on anything the core's HTTP library adds.
- **Network endpoints.** `https://<endpoint.address>`, with TLS, server
  verification and client certificates from the runtime config composition, exactly as
  for gRPC ([mTLS.md](./mTLS.md)). In `HTTP(dev)` the scheme is `http://`.
- **IPC.** HTTP over the Unix socket at `endpoint.socket` (a
  `ClusterRuntimeConfig` only), with `Host: localhost`.
- **Connection reuse.** One `http.Client` per runtime config, built on
  `esutils.CloneDefaultHTTPTransport()` (AGENTS.md) and keyed as for gRPC: the
  runtime config generation plus the `resourceVersion` of the client certificate Secret
  and of the `caRef` object.

## Health

`POST /grpc.health.v1.Health/Check` with body `{}` or `{"service": ""}`. The
response is `{"status": "SERVING"}` when healthy. Any other status, or an error,
is handled as in the gRPC binding. This is the standard gRPC health service
over Connect, so a provider built with a Connect library registers the stock
health handler, and a hand-written one adds a single route.

## Examples

The examples use the call from PROTOCOL.md "Example: one call, fully
populated". `$CERTS` stands for `--cacert ca.crt --cert tls.crt --key tls.key`,
the managed mutual-TLS composition.

### GetSecret, success

```bash
curl $CERTS \
  -H 'Content-Type: application/json' \
  -H 'Connect-Protocol-Version: 1' \
  -H 'Connect-Timeout-Ms: 30000' \
  -H 'X-Eso-Request-Id: 01JD3X8Q4M7R2N5V6W9Y0Z1A2B' \
  https://provider-aws.external-secrets-system.svc:8443/externalsecrets.provider.v1alpha1.SecretStoreProvider/GetSecret \
  -d @- <<'EOF'
{
  "providerConfig": {
    "gvkRef": {
      "apiVersion": "provider.external-secrets.io/v1alpha1",
      "kind": "SecretsManager",
      "namespace": "external-secrets-system",
      "name": "aws-shared"
    }
  },
  "context": {
    "request": {
      "apiVersion": "external-secrets.io/v1",
      "kind": "ExternalSecret",
      "namespace": "team-b",
      "name": "app-config",
      "uid": "6f1c2d3e-0000-4000-8000-00000000000b",
      "generation": "4"
    },
    "source": {
      "apiVersion": "external-secrets.io/v1",
      "kind": "ClusterSecretStore",
      "name": "aws-shared",
      "uid": "9a8b7c6d-0000-4000-8000-0000000000aa",
      "generation": "2"
    },
    "callerNamespace": "team-b",
    "requestLabels": { "app": "billing", "team": "payments" },
    "requestAnnotations": { "provider.external-secrets.io/purpose": "db" },
    "credentialNamespace": "team-b",
    "allowExplicitCredentialNamespace": true
  },
  "remoteRef": { "key": "/team-b/prod/db/password" }
}
EOF
```

```http
HTTP/2 200
content-type: application/json

{
  "value": "czNjcjN0LXA0c3M=",
  "resolved": {
    "credentialNamespaces": ["team-b"],
    "providerUid": "3c4d5e6f-0000-4000-8000-0000000000cc",
    "providerGeneration": "7"
  }
}
```

### GetSecret, remote secret missing

```http
HTTP/2 404
content-type: application/json

{
  "code": "not_found",
  "message": "secret /team-b/prod/db/password not found",
  "details": [{
    "type": "externalsecrets.provider.v1alpha1.ProviderError",
    "value": "ChlSZXNvdXJjZU5vdEZvdW5kRXhjZXB0aW9uEjBTZWNyZXRzIE1hbmFnZXIgY2FuJ3QgZmluZCB0aGUgc3BlY2lmaWVkIHNlY3JldC4",
    "debug": { "code": "ResourceNotFoundException", "message": "Secrets Manager can't find the specified secret." }
  }]
}
```

The core maps this to `esv1.NoSecretErr`. A hand-written provider can answer
the same thing with just `{"code": "not_found"}`, and the outcome is
identical.

### Validate through a SecretStore with a forbidden explicit namespace

```bash
curl $CERTS -H 'Content-Type: application/json' -H 'Connect-Timeout-Ms: 30000' \
  -H 'X-Eso-Request-Id: 01JD3XA0000000000000000000' \
  https://provider-aws.external-secrets-system.svc:8443/externalsecrets.provider.v1alpha1.SecretStoreProvider/Validate \
  -d '{"providerConfig":{"gvkRef":{"apiVersion":"provider.external-secrets.io/v1alpha1","kind":"SecretsManager","namespace":"team-a","name":"aws-prod"}},
       "context":{"source":{"apiVersion":"external-secrets.io/v1","kind":"SecretStore","namespace":"team-a","name":"aws-prod","uid":"...","generation":"1"},
                  "credentialNamespace":"team-a"}}'
```

`allowExplicitCredentialNamespace` is omitted because it is `false`, the
default.

```http
HTTP/2 400
content-type: application/json

{
  "code": "failed_precondition",
  "message": "auth.jwt.serviceAccountRef.namespace \"team-b\" is not allowed through a SecretStore in \"team-a\"",
  "details": [{
    "type": "externalsecrets.provider.v1alpha1.ProviderError",
    "value": "ChtFeHBsaWNpdE5hbWVzcGFjZU5vdEFsbG93ZWQSXmF1dGguand0LnNlcnZpY2VBY2NvdW50UmVmLm5hbWVzcGFjZSAidGVhbS1iIiBpcyBub3QgYWxsb3dlZCB0aHJvdWdoIGEgU2VjcmV0U3RvcmUgaW4gInRlYW0tYSI",
    "debug": { "code": "ExplicitNamespaceNotAllowed" }
  }]
}
```

### Describe and health over IPC

```bash
curl --unix-socket /var/run/external-secrets/aws.sock \
  -H 'Content-Type: application/json' -H 'Connect-Timeout-Ms: 30000' \
  -H 'X-Eso-Request-Id: 01JD3XB0000000000000000000' \
  http://localhost/externalsecrets.provider.v1alpha1.Runtime/Describe -d '{}'
```

```json
{
  "providerName": "aws",
  "providerVersion": "v2.16.0",
  "protocolVersions": ["v1alpha1"],
  "serves": [
    { "apiVersion": "provider.external-secrets.io/v1alpha1", "kind": "SecretsManager" },
    { "apiVersion": "provider.external-secrets.io/v1alpha1", "kind": "ParameterStore" }
  ],
  "capabilities": ["GetSecret", "GetSecretMap", "GetAllSecrets", "PushSecret", "DeleteSecret", "SecretExists"],
  "generates": [
    { "apiVersion": "provider.external-secrets.io/v1alpha1", "kind": "ECRAuthorizationToken" }
  ]
}
```

```bash
curl --unix-socket /var/run/external-secrets/aws.sock \
  -H 'Content-Type: application/json' \
  http://localhost/grpc.health.v1.Health/Check -d '{}'
# {"status":"SERVING"}
```

## What a hand-written provider implements

1. One `POST` route per operation it supports: `Runtime/Describe` and the
   health route always; `Validate`, `Capabilities` and the advertised store
   operations if it serves store kinds; `Generate` and `Cleanup` if it serves
   generator kinds. Answer `404` for the rest, which the core reads as
   `Unimplemented`.
2. JSON decoding that ignores unknown fields, and encoding by the table above
   (`int64` as a string, `bytes` as base64).
3. The error body with at least `code`. A missing remote secret must be
   `404` **with** `{"code": "not_found"}`. A bare `404` means `Unimplemented`,
   and the core would not apply `deletionPolicy`.
4. The authentication of its runtime config composition: TLS server certificate,
   client-certificate verification, or nothing over a Unix socket.
5. Everything in PROTOCOL.md "Obligations". The conformance suite runs over
   this binding as well as over gRPC, so the obligations are checked the same
   way.
