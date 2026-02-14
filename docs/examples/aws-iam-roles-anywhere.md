# Authenticate to AWS IAM Roles Anywhere (IAMRA) using Helm

## Helm Values
```
extraEnv:
- name: 'AWS_EC2_METADATA_SERVICE_ENDPOINT'
  value: 'http://127.0.0.1:9911/'
- name: 'AWS_REGION'
  value: <AWS_REGION>
extraContainers:
- name: <IAMRA_CONTAINER_NAME>
  image: <IAMRA_IMAGE> # See https://gallery.ecr.aws/rolesanywhere/credential-helper
  imagePullPolicy: 'Always'
  env:
  - name: 'TRUST_ANCHOR_ARN'
    value: <TRUST_ANCHOR_ARN>
  - name: 'PROFILE_ARN'
    value: <PROFILE_ARN>
  - name: 'ROLE_ARN'
    value: <ROLE_ARN>
  volumeMounts:
  - name: <TLS_SECRET_NAME>
    mountPath: <MOUNT_PATH>
    readOnly: true
  ports: []
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop: ['ALL']
    privileged: false
    readOnlyRootFilesystem: true
    runAsGroup: <IAMRA_GROUP_ID>
    runAsNonRoot: true
    runAsUser: <IAMRA_USER_ID>
extraVolumes:
- name: <TLS_SECRET_NAME>
  secret:
    secretName: <TLS_SECRET_NAME>
```

Note that we have assumed the port on which the signing helper is running is 9911, which is the default. You can configure the port to something else.

## Core idea
The AWS SDK will look for certain environment variables to authenticate when it isn't given explicit credentials.

One of these is 'AWS_EC2_METADATA_SERVICE_ENDPOINT'. By supplying this (and creating the appropriate backing service 'aws_signing_helper') that it points to, the AWS SDK that External Secrets makes use of will get credentials without External Secrets needing to carry out any IAM Roles Anywhere-specific auth process.

### Parts
* Appropriate environment variables made available to External Secrets:
    * AWS_EC2_METADATA_SERVICE_ENDPOINT, e.g. http://127.0.0.1:9911/
    * AWS_REGION (also needed)
* IAMRA sidecar container:
    * You can build this yourself from the binary releases AWS release, or use [the container AWS releases](https://gallery.ecr.aws/rolesanywhere/credential-helper).
* Appropriate configuration for the IAMRA sidecar container:
    * If you're using the container AWS releases, see [their documentation](https://github.com/aws/rolesanywhere-credential-helper/blob/main/docker_image_resources/README.md#environment-variables)
    * Otherwise, you'll probably need:
        * TRUST_ANCHOR_ARN - Format: arn:aws:rolesanywhere:region:account:trust-anchor/id - to tell AWS which mutually agreed root of trust you are using - a root certificate.
        * PROFILE_ARN - Format: arn:aws:rolesanywhere:region:account:profile/id
        * ROLE_ARN - Format: arn:aws:iam::account:role/role-name
        * CERTIFICATE_PATH - Path to your certificate (default: iamra/certificate.pem)
        * PRIVATE_KEY_PATH - Path to your private key (default: tests/private_key.pem)
