# All Keys, One Secret

To get multiple key-values from an external secret, not having to worry about how many, or what these keys are, we have to use the dataFrom field of the ExternalSecret resource, instead of the data field. We will give an example here with the gcp provider (should work with other providers in the same way).

Please follow the authentication and SecretStore steps of the [Google Cloud Secrets Manager guide](../provider/google-secrets-manager.md) to setup access to your google cloud account first.

Then create a secret in Google Cloud Secret Manager that contains a JSON string with multiple key values like this:

![secret-value](../pictures/screenshot_json_string_gcp_secret_value.png)

Let's call this secret all-keys-example-secret on Google Cloud.


### Creating dataFrom external secret

Now, when creating our ExternalSecret resource, instead of using the data field, we use the dataFrom field:

```yaml
{% include 'gcpsm-data-from-external-secret.yaml' %}
```

To check both values we can run:

```
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.username}' | base64 -d
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.surname}' | base64 -d
```
