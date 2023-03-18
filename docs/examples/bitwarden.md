# Bitwarden support using webhook provider

Bitwarden is an integrated open source password management solution for individuals, teams, and business organizations.

## How is it working ?

To make external-secret compatible with BitWarden, we need *
* External-Secret >= 0.8.0
* To use the Webhook Provider
* 2 (Cluster)SecretStores
* BitWarden CLI image running `bw serve`

When you create a new external-secret object,
External-Secret Webhook provider will do a query to the Bitwarden CLI pod,
which is synced with the BitWarden server.

## Requirements

* Bitwarden account (it works also with VaultWarden)
* A Kubernetes secret which contains your BitWarden Credentials
* You need a Docker image with BitWarden CLI installed
  You could use `registry.gitlab.com/ttblt-oss/docker-bw:2023.1.0` or build your own.

Here an example of Dockerfile use to build this image:
```
FROM debian:sid

ENV BW_CLI_VERSION=2023.1.0

RUN apt update && \
    apt install -y wget unzip && \
    wget https://github.com/bitwarden/clients/releases/download/cli-v${BW_CLI_VERSION}/bw-linux-${BW_CLI_VERSION}.zip && \
    unzip bw-linux-${BW_CLI_VERSION}.zip && \
    chmod +x bw && \
    mv bw /usr/local/bin/bw && \
    rm -rfv *.zip

COPY entrypoint.sh /

CMD ["/entrypoint.sh"]
```

And the content of `entrypoint.sh`
```
#!/bin/bash

set -e

bw config server ${BW_HOST}

export BW_SESSION=$(bw login ${BW_USER} --passwordenv BW_PASSWORD --raw)
#export BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw)

bw unlock --check

echo 'Running `bw server` on port 8087'
bw serve --hostname 0.0.0.0 #--disable-origin-protection
```


## Deploy Bitwarden Credentials

```
apiVersion: v1
data:
  BW_HOST: ...
  BW_USERNAME: ...
  BW_PASSWORD: ....
kind: Secret
metadata:
  name: bitwarden-cli
  namespace: bitwarden
type: Opaque
```

## Deploy Bitwarden CLI container

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bitwarden-cli
  namespace: bitwarden
  labels:
    app.kubernetes.io/instance: bitwarden-cli
    app.kubernetes.io/name: bitwarden-cli
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app.kubernetes.io/name: bitwarden-cli
      app.kubernetes.io/instance: bitwarden-cli
  template:
    metadata:
      labels:
        app.kubernetes.io/name: bitwarden-cli
        app.kubernetes.io/instance: bitwarden-cli
    spec:
      containers:
        - name: bitwarden-cli
          image: YOUR_BITWARDEN_CLI_IMAGE
          imagePullPolicy: IfNotPresent
          env:
            - name: BW_HOST
              valueFrom:
                secretKeyRef:
                  name: bitwarden-cli
                  key: BW_HOST
            - name: BW_USER
              valueFrom:
                secretKeyRef:
                  name: bitwarden-cli
                  key: BW_USERNAME
            - name: BW_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: bitwarden-cli
                  key: BW_PASSWORD
          ports:
            - name: http
              containerPort: 8087
              protocol: TCP
          livenessProbe:
            exec:
              command:
                - wget
                - -q
                - http://127.0.0.1:8087/sync
                - --post-data=''
            initialDelaySeconds: 20
            failureThreshold: 3
            timeoutSeconds: 1
            periodSeconds: 120
          readinessProbe:
            tcpSocket:
              port: 8087
            initialDelaySeconds: 20
            failureThreshold: 3
            timeoutSeconds: 1
            periodSeconds: 10
          startupProbe:
            tcpSocket:
              port: 8087
            initialDelaySeconds: 10
            failureThreshold: 30
            timeoutSeconds: 1
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: bitwarden-cli
  namespace: bitwarden
  labels:
    app.kubernetes.io/instance: bitwarden-cli
    app.kubernetes.io/name: bitwarden-cli
  annotations:
spec:
  type: ClusterIP
  ports:
  - port: 8087
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app.kubernetes.io/name: bitwarden-cli
    app.kubernetes.io/instance: bitwarden-cli
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  namespace: bitwarden
  name: external-secret-2-bw-cli
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/instance: bitwarden-cli
      app.kubernetes.io/name: bitwarden-cli
  ingress:
  - from:
      - podSelector:
          matchLabels:
            app.kubernetes.io/instance: external-secrets
            app.kubernetes.io/name: external-secrets
```

> NOTE: Deploying a network policy is recommended since, there is no authentication to query the BitWarden CLI, which means that your secrets are exposed.

> NOTE: In this example the Liveness probe is quering /sync to ensure that the BitWarden CLI is able to connect to the server and also to sync secrets. (The secret sync is only every 2 minutes in this example)

## Deploy ClusterSecretStore (Or SecretStore)

Here the two ClusterSecretStore to deploy

```
---
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: bitwarden-login
spec:
  provider:
    webhook:
      url: "http://bitwarden-cli:8087/object/item/{{ .remoteRef.key }}"
      headers:
        Content-Type: application/json
      result:
        jsonPath: "$.data.login.{{ .remoteRef.property }}"
---
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: bitwarden-fields
spec:
  provider:
    webhook:
      url: "http://bitwarden-cli:8087/object/item/{{ .remoteRef.key }}"
      result:
        jsonPath: "$.data.fields[?@.name==\"{{ .remoteRef.property }}\"].value"
```


## How to use it ?

* If you need the `username` or the `password` of a secret, you have to use `bitwarden-login`
* If you need a custom field of a secret, you have to use `bitwarden-fields`
* The `key` is the ID of a secret, which can be find in the URL with the `itemId` value:
  `https://myvault.com/#/vault?itemId=........-....-....-....-............`
* The `property` is the name of the field:
  * `username` for the username of a secret (`bitwarden-login` SecretStore)
  * `password` for the password of a secret (`bitwarden-login` SecretStore)
  * `name_of_the_custom_field` for any custom field (`bitwarden-fields` SecretStore)

```
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: my-db-secrets
  namespace: default
spec:
  target:
    name: my-db-secrets
    deletionPolicy: Delete
    template:
      type: Opaque
      data:
        username: |-
          {{ .username }}
        password: |-
          {{ .password }}
        postgres-password: |-
          {{ .postgres_password }}
        postgres-replication-password: |-
          {{ .postgres_replication_password }}
        db_url: |-
          postgresql://{{ .username }}:{{ .password }}@my-postgresql:5432/mydb
  data:
    - secretKey: username
      sourceRef:
        storeRef:
          name: bitwarden-login
          kind: ClusterSecretStore  # or SecretStore
      remoteRef:
        key: aaaabbbb-cccc-dddd-eeee-000011112222
        property: username
    - secretKey: password
      sourceRef:
        storeRef:
          name: bitwarden-login
          kind: ClusterSecretStore  # or SecretStore
      remoteRef:
        key: aaaabbbb-cccc-dddd-eeee-000011112222
        property: password
    - secretKey: postgres_password
      sourceRef:
        storeRef:
          name: bitwarden-fields
          kind: ClusterSecretStore  # or SecretStore
      remoteRef:
        key: aaaabbbb-cccc-dddd-eeee-000011112222
        property: admin-password
    - secretKey: postgres_replication_password
      sourceRef:
        storeRef:
          name: bitwarden-fields
          kind: ClusterSecretStore  # or SecretStore
      remoteRef:
        key: aaaabbbb-cccc-dddd-eeee-000011112222
        property: postgres-replication-password
```
