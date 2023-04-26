# Getting started

Jenkins is one of the most popular automation servers for continuous integration, automation, scheduling jobs and for generic pipelining. It has an extensive set of plugins that extend or provide additional functionality including the [kubernetes credentials plugin](https://github.com/jenkinsci/kubernetes-credentials-provider-plugin). This plugin takes kubernetes secrets and creates Jenkins credentials from them removing the need for manual entry of secrets, local storage and manual secret rotation.

## Examples

The Jenkins credentials plugin uses labels and annotations on a kubernetes secret to create a Jenkins credential.

The different types of Jenkins credentials that can be created are SecretText, privateSSHKey, UsernamePassword.


### SecretText

Here are some examples of SecretText with the Hashicorp Vault and AWS External Secrets providers:


#### Hashicorp Vault

``` yaml
{% include 'vault-jenkins-credential-sonarqube-api-token-external-secret.yaml' %}
```

#### AWS Secrets Manager

``` yaml
{% include 'aws-jenkins-credential-sonarqube-api-token-external-secret.yaml' %}
```


### UsernamePassword

Here are some examples of UsernamePassword credentials with the Hashicorp Vault and AWS External Secrets providers:


#### Hashicorp Vault

``` yaml
{% include 'vault-jenkins-credential-harbor-chart-robot-external-secret.yaml' %}
```

#### AWS Secrets Manager

``` yaml
{% include 'aws-jenkins-credentials-harbor-chart-robot-external-secret.yaml' %}
```



### basicSSHUserPrivateKey

Here are some examples of basicSSHUserPrivateKey credentials with the Hashicorp Vault and AWS External Secrets providers:


#### Hashicorp Vault

``` yaml
{% include 'vault-jenkins-credential-github-ssh-access-external-secret.yaml' %}
```

#### AWS Secrets Manager

``` yaml
{% include 'aws-jenkins-credential-github-ssh-external-secret.yaml' %}
```

