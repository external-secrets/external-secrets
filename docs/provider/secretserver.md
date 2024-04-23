## Delinea Secret Server

External Secrets Operator integrates with [Delinea Secret Server](https://docs.delinea.com/online-help/secret-server/start.htm).

### Creating a SecretStore

You need a username, password and a fully qualified Secret Server tenant URL to authenticate i.e. `https://yourTenantName.secretservercloud.com`.

Both username and password can be specified either directly in the `SecretStore`, or by referencing a kubernetes secret.

To acquire a username and password, refer to the  [user management](https://docs.delinea.com/online-help/secret-server/users/creating-users/index.htm) documentation.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secret-server-store
spec:
  provider:
    secretserver:
      serverURL: <SERVER_URL>
      username:
        value: <USERNAME>
      password:
        secretRef:
          name: <NAME_OF_KUBE_SECRET>
          key: <KEY_IN_KUBE_SECRET>
```

Both `username` and `password` can either be specified directly via the `value` field or can reference a kubernetes secret.


### Referencing Secrets

Secrets must be referenced by ID. `Getting a specific version of a secret is not yet supported.`

Note that because all Secret Server secrets are JSON objects, you must specify `remoteRef.property`. You can access nested values or arrays using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md).

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
    name: secret-server-external-secret
spec:
    refreshInterval: 15s
    secretStoreRef:
        kind: SecretStore
        name: secret-server-store
    data:
      - secretKey: SecretServerValue #<KEY_IN_KUBE_SECRET>
        remoteRef:
          key: "52622" #<SECRET_ID>
          property: "Items.0.ItemValue" #<GJSON_PROPERTY>
```
### Example
Using the json formatted secret below to retrieve the "ItemValue" for "FieldName" .. "Data"

spec.data.remoteRef.key = 52622 (id of the secret)

spec.data.remoteRef.property = Items.0.ItemValue (gjson path )

```JSON
{
  "Name": "external secret testing",
  "FolderID": 73,
  "ID": 52622,
  "SiteID": 1,
  "SecretTemplateID": 6098,
  "SecretPolicyID": -1,
  "PasswordTypeWebScriptID": -1,
  "LauncherConnectAsSecretID": -1,
  "CheckOutIntervalMinutes": -1,
  "Active": true,
  "CheckedOut": false,
  "CheckOutEnabled": false,
  "AutoChangeEnabled": false,
  "CheckOutChangePasswordEnabled": false,
  "DelayIndexing": false,
  "EnableInheritPermissions": true,
  "EnableInheritSecretPolicy": true,
  "ProxyEnabled": false,
  "RequiresComment": false,
  "SessionRecordingEnabled": false,
  "WebLauncherRequiresIncognitoMode": false,
  "Items": [
    {
      "ItemID": 280265,
      "FieldID": 439,
      "FileAttachmentID": 0,
      "FieldName": "Data",
      "Slug": "data",
      "FieldDescription": "json text field",
      "Filename": "",
      "ItemValue": "{\"key\":\"value\"}",
      "IsFile": false,
      "IsNotes": false,
      "IsPassword": false
    }
  ]
}
```
