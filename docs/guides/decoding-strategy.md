# Decoding Strategies
The External Secrets Operator has the feature to allow multiple decoding strategies during an object generation.

The `decodingStrategy` field allows the user to set the following Decoding Strategies based on their needs. `decodingStrategy` can be placed under `spec.data.remoteRef`, `spec.dataFrom.extract` or `spec.dataFrom.find`. It will configure the decoding strategy for that specific operation, leaving others with the default behavior if not set.

### None (default)
ESO will not try to decode the secret value.

### Base64
ESO will try to decode the secret value using [base64](https://datatracker.ietf.org/doc/html/rfc4648#section-4) method. If the decoding fails, an error is produced.

### Base64URL
ESO will try to decode the secret value using [base64url](https://datatracker.ietf.org/doc/html/rfc4648#section-5) method. If the decoding fails, an error is produced.

### Auto
ESO will try to decode using Base64/Base64URL strategies. If the decoding fails, ESO will apply decoding strategy None. No error is produced to the user.

## Examples

### Setting Decoding strategy Auto in a DataFrom.Extract
Given that we have the given secret information:
```
{
    "name": "Gustavo",
    "surname": "Fring",
    "address":"aGFwcHkgc3RyZWV0",
}
```
if we apply the following dataFrom:
```
spec:
  dataFrom:
  - extract:
      key: my-secret
      decodingStrategy: Auto
```
It will render the following Kubernetes Secret:
```
data:
  name: R3VzdGF2bw==        #Gustavo
  surname: RnJpbmc=         #Fring
  address: aGFwcHkgc3RyZWV0 #happy street
```

## Limitations

At this time, decoding Strategy Auto is only trying to check if the original input is valid to perform Base64 operations. As there is no reliable way to detect base64 encoded values, this means that some non-encoded secret values might end up being decoded, producing gibberish. For example, this is the case for alphanumeric values with a length divisible by 4, like `1234` or `happy/street`. 

!!! note 
    If you are using `decodeStrategy: Auto` and start to see ESO pulling completely wrong secret values into your kubernetes secret, consider changing it to `None` to investigate it.
