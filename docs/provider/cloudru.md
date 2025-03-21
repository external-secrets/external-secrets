External Secrets Operator integrates with [Cloud.ru](https://cloud.ru) for secret management.

Cloud.ru Secret Manager works in conjunction with the Key Manager cryptographic key management system to ensure secure
encryption of secrets.

### Authentication

* Before you can use the Cloud.ru Secret Manager, you need to create a service account in
  the [Cloud.ru Console](https://console.cloud.ru).
* Create a [Service Account](https://cloud.ru/ru/docs/console_api/ug/topics/guides__service_accounts_create.html)
  and [Access Key](https://cloud.ru/ru/docs/console_api/ug/topics/guides__service_accounts_key.html) for it.

**NOTE:** To interact with the SecretManager API, you need to use the access token. You can get it by running the
following command, using the Access Key, created above:

```shell
curl -i --data-urlencode 'grant_type=access_key' \
  --data-urlencode "client_id=$KEY_ID" \
  --data-urlencode "client_secret=$SECRET" \
  https://id.cloud.ru/auth/system/openid/token
```

### Creating Cloud.ru secret

To make External Secrets Operator sync a k8s secret with a Cloud.ru secret:

* Navigate to the [Cloud.ru Console](https://console.cloud.ru/).
* Click the menu at upper-left corner, scroll down to the `Management` section and click on `Secret Manager`.
* Click on `Create secret`.
* Fill in the secret name and secret value.
* Click on `Create`.

Also, you can use [SecretManager API](https://cloud.ru/ru/docs/scsm/ug/topics/guides__add-secret.html) to create the
secret:

```shell
curl --location 'https://secretmanager.api.cloud.ru/v1/secrets' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer ${ACCESS_TOKEN}' \
--data '{
    "description": "your secret description",
    "labels": {
        "env": "production"
    },
    "name": "my_first_secret",
    "parent_id": "50000000-4000-3000-2000-100000000001",
    "payload": {
        "data": {
            "value": "aGksIHRoZXJlJ3Mgbm90aGluZyBpbnRlcmVzdGluZyBoZXJlCg=="
        }
    }
}'
```

* `ACCESS_TOKEN` is the access token for the Cloud.ru API. See **Authentication** section
* `parent_id` parent service instance identifier: ServiceInstanceID. To get the ID value, in your personal account on
  the top left panel, click the Button with nine dots, select **Management** → **Secret Manager** and copy the value
  from the Service Instance ID field.
* `name` is the name of the secret.
* `description` is the description of the secret.
* `labels` are the labels(tags) for the secret. Is used in the search.
* `payload.data.value` is the base64-encoded secret value.

**NOTE:** To create the Multi KeyValue secret in Cloud.ru, you can use the following format (json):

```json
{
  "key1": "value1",
  "key2": "value2"
}
```

### Creating ExternalSecret

* Create the k8s Secret, it will be used for authentication in SecretStore:
    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
        name: csm-secret
        labels:
          type: csm
    type: Opaque
    stringData:
        key_id: '000000000000000000001'
        key_secret: '000000000000000000002'
    ```
    * `key_id` is the AccessKey key_id.
    * `key_secret` is the AccessKey key_secret
* Create a [SecretStore](../api/secretstore.md) pointing to `csm-secret` k8s Secret:
    ```yaml
    apiVersion: external-secrets.io/v1beta1
    kind: SecretStore
    metadata:
      name: csm
    spec:
      provider:
        cloudrusm:
          auth:
            secretRef:
              accessKeyIDSecretRef:
                name: csm-secret
                key: key_id
              accessKeySecretSecretRef:
                name: csm-secret
                key: key_secret
          projectID: 50000000-4000-3000-2000-100000000001
    ```
    * `accessKeyIDSecretRef` is the reference to the k8s Secret with the AccessKey.
    * `projectID`  is the project identifier. To get the project id value, in your
      personal account on the top left, click on project name, In the opening window,
      click at 3 points next to the name of the necessary project, then the button "Copy the Project ID".
#### Create an [ExternalSecret](../api/externalsecret.md) pointing to SecretStore.
  * Classic, non-json:
    ```yaml
    apiVersion: external-secrets.io/v1beta1
    kind: ExternalSecret
    metadata:
      name: csm-ext-secret
    spec:
      refreshInterval: 10s
      secretStoreRef:
        name: csm
        kind: SecretStore
      target:
        name: my-awesome-secret
        creationPolicy: Owner
      data:
        - secretKey: target_key
          remoteRef:
            key: my_first_secret # or you can use the secret.id (e.g. 50000000-4000-3000-2000-100000000001)
    ```
  * From Multi KeyValue, value MUST be in **json format**:
  NOTE: You can use *either* `name` or `tags` to filter the secrets. Here are basic examples of both:
    ```yaml
    apiVersion: external-secrets.io/v1beta1
    kind: ExternalSecret
    metadata:
      name: csm-ext-secret
    spec:
      refreshInterval: 10s
      secretStoreRef:
        name: csm
        kind: SecretStore
      target:
        name: my-awesome-secret
        creationPolicy: Owner
      data:
        - secretKey: target_key
          remoteRef:
            key: my_first_secret # or you can use the secret.id (e.g. 50000000-4000-3000-2000-100000000001)
            property: cloudru.secret.key # is the JSON path for the key in the secret value.
    ```

  * With all fields, value MUST be in **json format**:
    ```yaml
    apiVersion: external-secrets.io/v1beta1
    kind: ExternalSecret
    metadata:
      name: csm-ext-secret
    spec:
      refreshInterval: 10s
      secretStoreRef:
        name: csm
        kind: SecretStore
      target:
        name: my-awesome-secret
        creationPolicy: Owner
      dataFrom:
        - extract:
            key: my_first_secret # or you can use the secret.id (e.g. 50000000-4000-3000-2000-100000000001)
    ```
  * Search the secrets by the Name or Labels (tags):
    ```yaml
    apiVersion: external-secrets.io/v1beta1
    kind: ExternalSecret
    metadata:
      name: csm-ext-secret
    spec:
      refreshInterval: 10s
      secretStoreRef:
        name: csm
        kind: SecretStore
      target:
        name: my-awesome-secret
        creationPolicy: Owner
      dataFrom:
        - find: # You can use the name and tags separately or together to search for secrets.
            tags:
              env: production
            name:
              regexp: "my.*secret"
    ```
