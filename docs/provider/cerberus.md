# Cerberus

## Cerberus
External Secrets Operator integrates with [Cerberus](https://engineering.nike.com/cerberus/) - a secure property store for applications.

## Authentication
Cerberus Provider supports AWS IAM STS [authentication](https://engineering.nike.com/cerberus/docs/authentication/aws-iam-sts-authentication).

The easiest way is to use [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) mechanism for connecting the ServiceAccount and IAM Role with permissions to access Cerberus. 

## Creating a Secret Store
For creating a SecretStore you need to specify the Cerberus region, Safe Deposit Box name and URL of Cerberus. Authentication is done using IAM Role, as described in [authentication section](#authentication).

```yml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: cerberus-pokedex
spec:
  provider:
    cerberus:
      region: us-west-2
      sdb: pokedex
      cerberusURL: https://cerberus.my.domain
      auth:
        jwt:
          serviceAccountRef:
            name: cerberus-creds
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cerberus-creds
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::000111222333:role/cerberus-read-access
```

## Creating External Secret
To get a secret from Cerberus and create it as a secret on the cluster, an `ExternalSecret` object is required.

```yml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: bulbazaur
spec:
  secretStoreRef:
    name: cerberus-pokedex
    kind: SecretStore
  refreshInterval: "1m"
  target:
    deletionPolicy: "Delete"
  data:
  - secretKey: pokemon-type 
      key: cerberus-pokedex
      property: type
  dataFrom:
  - extract:
      key: pokemons/jigglypuff
      version: b28afb82-b6e3-4c6e-b141-c54808edb632
  - find:
      path: pokemons/
      name:
        regexp: ".*zaur.*"
```

Above `ExternalSecret` will extract:
- `type` property from the `cerberus-pokedex` secret in Cerberus and put it in the `.spec.secretStoreRef` secret as `pokemon-type`,
- all keys from the `pokemons/jigglypuff` secret in Cerberus with the given version,
- all keys from the secrets matching `.*zaur.*` regex with `pokemons/` path.

## Creating Push Secret
Cerberus Provider supports the PushSecret mechanism.

```yml
apiVersion: v1
stringData:
  id: "0039"
  type: Fairy
immutable: false
kind: Secret
metadata:
  name: jigglypuff
type: Opaque
---
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: jigglypuff
spec:
  deletionPolicy: Delete
  refreshInterval: 10s
  secretStoreRefs:
  - name: cerberus-pokedex
    kind: SecretStore
  selector:
   secret:
      name: jigglypuff
  data:
  - match:
      secretKey: id
      remoteRef:
        property: id
        remoteKey: pokemons/jigglypuff
  - match:
      secretKey: type
      remoteRef:
        property: type
        remoteKey: pokemons/jigglypuff
```

Above PushSecret will create a secret in Cerberus and set `id` and `type` properties. Specifying `.spec.data[].match.remoteRef.property` is required.
