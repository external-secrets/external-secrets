## BeyondTrust Password Safe

External Secrets Operator integrates with [BeyondTrust Password Safe](https://www.beyondtrust.com/docs/beyondinsight-password-safe/).

Warning: The External Secrets Operator secure usage involves taking several measures. Please see [Security Best Practices](https://external-secrets.io/latest/guides/security-best-practices/) for more information.

Warning: If the BT provider secret is deleted it will still exist in the Kubernetes secrets.

### Prerequisites
The BT provider supports retrieval of a secret from BeyondInsight/Password Safe versions 23.1 or greater.

For this provider to retrieve a secret the Password Safe/Secrets Safe instance must be preconfigured with the secret in question and authorized to read it.

### Authentication

BeyondTrust [OAuth Authentication](https://www.beyondtrust.com/docs/beyondinsight-password-safe/ps/admin/configure-api-registration.htm).

1. Create an API access registration in BeyondInsight
2. Create or use an existing Secrets Safe Group
3. Create or use an existing Application User
4. Add API registration to the Application user
5. Add the user to the group
6. Add the Secrets Safe Feature to the group

> NOTE: The ClentID and ClientSecret must be stored in a Kubernetes secret in order for the SecretStore to read the configuration.

```sh
kubectl create secret generic bt-secret --from-literal ClientSecret="<your secret>"
kubectl create secret generic bt-id --from-literal ClientId="<your ID>"
```

### Client Certificate

If using `retrievalType: MANAGED_ACCOUNT`, you will also need to download the pfx certificate from Secrets Safe, extract that certificate and create two Kubernetes secrets.

```sh
openssl pkcs12 -in client_certificate.pfx -nocerts -out ps_key.pem -nodes
openssl pkcs12 -in client_certificate.pfx -clcerts -nokeys -out ps_cert.pem

# Copy the text from the ps_key.pem to a file.
-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----

# Copy the text from the ps_cert.pem to a file.
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----

kubectl create secret generic bt-certificate --from-file=ClientCertificate=./ps_cert.pem
kubectl create secret generic bt-certificatekey --from-file=ClientCertificateKey=./ps_key.pem
```

### Creating a SecretStore

You can follow the below example to create a `SecretStore` resource.
You can also use a `ClusterSecretStore` allowing you to reference secrets from all namespaces. [ClusterSecretStore](https://external-secrets.io/latest/api/clustersecretstore/)

```sh
kubectl apply -f secret-store.yml
```

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secretstore-beyondtrust
spec:
  provider:
    beyondtrust:
      server:
        apiUrl: https://example.com:443/BeyondTrust/api/public/v3/
        retrievalType: MANAGED_ACCOUNT # or SECRET
        verifyCA: true
        clientTimeOutSeconds: 45
      auth: 
        certificate: # omit certificates if retrievalType is SECRET
          secretRef:
            name: bt-certificate
            key: ClientCertificate
        certificateKey:
          secretRef:
            name: bt-certificatekey
            key: ClientCertificateKey
        clientSecret:
          secretRef:
            name: bt-secret
            key: ClientSecret
        clientId:
          secretRef:
            name: bt-id
            key: ClientId
```

### Creating a ExternalSecret

You can follow the below example to create a `ExternalSecret` resource. Secrets can be referenced by path.
You can also use a `ClusterExternalSecret` allowing you to reference secrets from all namespaces.

```sh
kubectl apply -f external-secret.yml
```

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: beyondtrust-external-secret
spec:
  refreshInterval: 300s
  secretStoreRef:
    kind: SecretStore
    name: secretstore-beyondtrust
  target:
    name: my-beyondtrust-secret # name of secret to create in k8s secrets (etcd)
    creationPolicy: Owner
  data:
    - secretKey: secretKey
      remoteRef:
        key: system01/managed_account01
```

### Get the K8s secret

```shell
# WARNING: this command will reveal the stored secret in plain text
kubectl get secret my-beyondtrust-secret -o jsonpath="{.data.secretKey}" | base64 --decode && echo
```