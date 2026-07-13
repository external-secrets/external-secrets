module github.com/external-secrets/external-secrets

go 1.26.5

replace (
	github.com/external-secrets/external-secrets/apis => ./apis
	github.com/external-secrets/external-secrets/generators/v1/acr => ./generators/v1/acr
	github.com/external-secrets/external-secrets/generators/v1/beyondtrustworkloadcredentials => ./generators/v1/beyondtrustworkloadcredentials
	github.com/external-secrets/external-secrets/generators/v1/cloudsmith => ./generators/v1/cloudsmith
	github.com/external-secrets/external-secrets/generators/v1/ecr => ./generators/v1/ecr
	github.com/external-secrets/external-secrets/generators/v1/fake => ./generators/v1/fake
	github.com/external-secrets/external-secrets/generators/v1/gcr => ./generators/v1/gcr
	github.com/external-secrets/external-secrets/generators/v1/github => ./generators/v1/github
	github.com/external-secrets/external-secrets/generators/v1/gitlab => ./generators/v1/gitlab
	github.com/external-secrets/external-secrets/generators/v1/grafana => ./generators/v1/grafana
	github.com/external-secrets/external-secrets/generators/v1/mfa => ./generators/v1/mfa
	github.com/external-secrets/external-secrets/generators/v1/password => ./generators/v1/password
	github.com/external-secrets/external-secrets/generators/v1/quay => ./generators/v1/quay
	github.com/external-secrets/external-secrets/generators/v1/sshkey => ./generators/v1/sshkey
	github.com/external-secrets/external-secrets/generators/v1/sts => ./generators/v1/sts
	github.com/external-secrets/external-secrets/generators/v1/uuid => ./generators/v1/uuid
	github.com/external-secrets/external-secrets/generators/v1/vault => ./generators/v1/vault
	github.com/external-secrets/external-secrets/generators/v1/webhook => ./generators/v1/webhook
	github.com/external-secrets/external-secrets/providers/v1/akeyless => ./providers/v1/akeyless
	github.com/external-secrets/external-secrets/providers/v1/aws => ./providers/v1/aws
	github.com/external-secrets/external-secrets/providers/v1/azure => ./providers/v1/azure
	github.com/external-secrets/external-secrets/providers/v1/barbican => ./providers/v1/barbican
	github.com/external-secrets/external-secrets/providers/v1/beyondtrust => ./providers/v1/beyondtrust
	github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials => ./providers/v1/beyondtrustworkloadcredentials
	github.com/external-secrets/external-secrets/providers/v1/bitwarden => ./providers/v1/bitwarden
	github.com/external-secrets/external-secrets/providers/v1/chef => ./providers/v1/chef
	github.com/external-secrets/external-secrets/providers/v1/cloudru => ./providers/v1/cloudru
	github.com/external-secrets/external-secrets/providers/v1/conjur => ./providers/v1/conjur
	github.com/external-secrets/external-secrets/providers/v1/delinea => ./providers/v1/delinea
	github.com/external-secrets/external-secrets/providers/v1/doppler => ./providers/v1/doppler
	github.com/external-secrets/external-secrets/providers/v1/dvls => ./providers/v1/dvls
	github.com/external-secrets/external-secrets/providers/v1/fake => ./providers/v1/fake
	github.com/external-secrets/external-secrets/providers/v1/fortanix => ./providers/v1/fortanix
	github.com/external-secrets/external-secrets/providers/v1/gcp => ./providers/v1/gcp
	github.com/external-secrets/external-secrets/providers/v1/github => ./providers/v1/github
	github.com/external-secrets/external-secrets/providers/v1/gitlab => ./providers/v1/gitlab
	github.com/external-secrets/external-secrets/providers/v1/ibm => ./providers/v1/ibm
	github.com/external-secrets/external-secrets/providers/v1/infisical => ./providers/v1/infisical
	github.com/external-secrets/external-secrets/providers/v1/keepersecurity => ./providers/v1/keepersecurity
	github.com/external-secrets/external-secrets/providers/v1/kubernetes => ./providers/v1/kubernetes
	github.com/external-secrets/external-secrets/providers/v1/nebius => ./providers/v1/nebius
	github.com/external-secrets/external-secrets/providers/v1/ngrok => ./providers/v1/ngrok
	github.com/external-secrets/external-secrets/providers/v1/onboardbase => ./providers/v1/onboardbase
	github.com/external-secrets/external-secrets/providers/v1/onepassword => ./providers/v1/onepassword
	github.com/external-secrets/external-secrets/providers/v1/onepasswordsdk => ./providers/v1/onepasswordsdk
	github.com/external-secrets/external-secrets/providers/v1/openbao => ./providers/v1/openbao
	github.com/external-secrets/external-secrets/providers/v1/oracle => ./providers/v1/oracle
	github.com/external-secrets/external-secrets/providers/v1/ovh => ./providers/v1/ovh
	github.com/external-secrets/external-secrets/providers/v1/passbolt => ./providers/v1/passbolt
	github.com/external-secrets/external-secrets/providers/v1/passworddepot => ./providers/v1/passworddepot
	github.com/external-secrets/external-secrets/providers/v1/previder => ./providers/v1/previder
	github.com/external-secrets/external-secrets/providers/v1/pulumi => ./providers/v1/pulumi
	github.com/external-secrets/external-secrets/providers/v1/scaleway => ./providers/v1/scaleway
	github.com/external-secrets/external-secrets/providers/v1/secretserver => ./providers/v1/secretserver
	github.com/external-secrets/external-secrets/providers/v1/senhasegura => ./providers/v1/senhasegura
	github.com/external-secrets/external-secrets/providers/v1/vault => ./providers/v1/vault
	github.com/external-secrets/external-secrets/providers/v1/volcengine => ./providers/v1/volcengine
	github.com/external-secrets/external-secrets/providers/v1/webhook => ./providers/v1/webhook
	github.com/external-secrets/external-secrets/providers/v1/yandex => ./providers/v1/yandex
	github.com/external-secrets/external-secrets/runtime => ./runtime
)

require (
	cloud.google.com/go/iam v1.11.0 // indirect
	cloud.google.com/go/secretmanager v1.20.0 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.30 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.13 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.2 // indirect
	github.com/IBM/go-sdk-core/v5 v5.22.1 // indirect
	github.com/IBM/secrets-manager-go-sdk/v2 v2.0.22 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.0
	github.com/akeylesslabs/akeyless-go-cloud-id v0.3.8 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/go-logr/logr v1.4.3
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/gax-go/v2 v2.23.0 // indirect
	github.com/hashicorp/vault/api v1.23.0 // indirect
	github.com/hashicorp/vault/api/auth/approle v0.12.0 // indirect
	github.com/hashicorp/vault/api/auth/kubernetes v0.12.0 // indirect
	github.com/hashicorp/vault/api/auth/ldap v0.12.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.41.0
	github.com/oracle/oci-go-sdk/v65 v65.120.0 // indirect
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	github.com/tidwall/gjson v1.19.0 // indirect
	github.com/yandex-cloud/go-genproto v0.95.0 // indirect
	github.com/yandex-cloud/go-sdk v0.32.0 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.uber.org/zap v1.28.0
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	google.golang.org/api v0.288.0 // indirect
	google.golang.org/genproto v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/grpc v1.82.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	grpc.go4.org v0.0.0-20170609214715-11d0a25b4919 // indirect
	k8s.io/api v0.36.2
	k8s.io/apiextensions-apiserver v0.36.2
	k8s.io/apimachinery v0.36.2
	k8s.io/client-go v0.36.2
	k8s.io/utils v0.0.0-20260707023825-cf1189d6abe3
	sigs.k8s.io/controller-runtime v0.24.1
	sigs.k8s.io/controller-tools v0.21.0
)

require github.com/1Password/connect-sdk-go v1.5.3 // indirect

require (
	github.com/external-secrets/external-secrets/apis v0.0.0
	github.com/external-secrets/external-secrets/generators/v1/acr v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/beyondtrustworkloadcredentials v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/cloudsmith v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/ecr v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/fake v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/gcr v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/github v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/gitlab v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/grafana v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/mfa v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/password v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/quay v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/sshkey v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/sts v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/uuid v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/vault v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/generators/v1/webhook v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/akeyless v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/aws v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/azure v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/barbican v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/beyondtrust v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/bitwarden v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/chef v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/cloudru v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/conjur v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/delinea v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/doppler v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/dvls v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/fake v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/fortanix v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/gcp v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/github v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/gitlab v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/ibm v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/infisical v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/keepersecurity v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/kubernetes v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/nebius v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/ngrok v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/onboardbase v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/onepassword v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/onepasswordsdk v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/openbao v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/oracle v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/ovh v0.0.0-20260713105520-df71ed0b968c
	github.com/external-secrets/external-secrets/providers/v1/passbolt v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/passworddepot v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/previder v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/pulumi v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/scaleway v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/secretserver v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/senhasegura v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/vault v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/volcengine v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/webhook v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/providers/v1/yandex v0.0.0-20260713095151-d0f709714e0a
	github.com/external-secrets/external-secrets/runtime v0.0.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.12.2
	github.com/robfig/cron/v3 v3.0.1
	sigs.k8s.io/yaml v1.6.0
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	cel.dev/expr v0.25.2 // indirect
	cloud.google.com/go/auth v0.21.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/1password/onepassword-sdk-go v0.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.22.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.14.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates v1.5.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys v1.5.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.5.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/internal v1.2.0 // indirect
	github.com/Azure/go-ntlmssp v0.1.1 // indirect
	github.com/BeyondTrust/go-client-library-passwordsafe v1.3.2 // indirect
	github.com/DelineaXPM/dsv-sdk-go/v2 v2.2.0 // indirect
	github.com/DelineaXPM/tss-sdk-go/v3 v3.0.2 // indirect
	github.com/Devolutions/go-dvls v0.19.1 // indirect
	github.com/Onboardbase/go-cryptojs-aes-decrypt v0.0.0-20230430095000-27c0d3a9016d // indirect
	github.com/ProtonMail/go-crypto v1.4.1 // indirect
	github.com/ProtonMail/gopenpgp/v3 v3.4.1 // indirect
	github.com/akeylesslabs/akeyless-go/v4 v4.3.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.42.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.29 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.28 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.59.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.40.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.43.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.71.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.32.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.37.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.44.0 // indirect
	github.com/aws/smithy-go v1.27.3 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bradleyfalzon/ghinstallation/v2 v2.19.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cloudflare/circl v1.6.4 // indirect
	github.com/cloudru-tech/iam-sdk v1.0.4 // indirect
	github.com/cloudru-tech/secret-manager-sdk v1.1.1 // indirect
	github.com/cyberark/conjur-api-go v0.15.3 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/dylibso/observe-sdk/go v0.0.0-20240828172851-9145d8ad07e1 // indirect
	github.com/extism/go-sdk v1.7.1 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fortanix/sdkms-client-go v0.4.2 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.25.3 // indirect
	github.com/go-openapi/loads v0.24.0 // indirect
	github.com/go-openapi/runtime v0.32.4 // indirect
	github.com/go-openapi/runtime/server-middleware v0.32.4 // indirect
	github.com/go-openapi/spec v0.22.6 // indirect
	github.com/go-openapi/strfmt v0.26.4 // indirect
	github.com/go-openapi/swag/cmdutils v0.27.0 // indirect
	github.com/go-openapi/swag/conv v0.27.0 // indirect
	github.com/go-openapi/swag/fileutils v0.27.0 // indirect
	github.com/go-openapi/swag/jsonname v0.27.0 // indirect
	github.com/go-openapi/swag/jsonutils v0.27.0 // indirect
	github.com/go-openapi/swag/loading v0.27.0 // indirect
	github.com/go-openapi/swag/mangling v0.27.0 // indirect
	github.com/go-openapi/swag/netutils v0.27.0 // indirect
	github.com/go-openapi/swag/stringutils v0.27.0 // indirect
	github.com/go-openapi/swag/typeutils v0.27.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.27.0 // indirect
	github.com/go-openapi/validate v0.26.0 // indirect
	github.com/go-playground/validator/v10 v10.30.3 // indirect
	github.com/go-resty/resty/v2 v2.17.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/cel-go v0.29.2 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/go-github/v56 v56.0.0 // indirect
	github.com/google/go-github/v88 v88.0.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/gophercloud/gophercloud/v2 v2.13.0 // indirect
	github.com/grafana/grafana-openapi-client-go v0.0.0-20260608140303-399c66621c54 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/hashicorp/go-secure-stdlib/awsutil v0.3.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/vault/api/auth/aws v0.12.0 // indirect
	github.com/hashicorp/vault/api/auth/gcp v0.12.0 // indirect
	github.com/hashicorp/vault/api/auth/userpass v0.12.0 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20260505044615-1ff4bf46051f // indirect
	github.com/infisical/go-sdk v0.8.0 // indirect
	github.com/keeper-security/secrets-manager-go/core v1.7.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.7 // indirect
	github.com/nebius/gosdk v0.2.37 // indirect
	github.com/ngrok/ngrok-api-go/v9 v9.0.0 // indirect
	github.com/oapi-codegen/runtime v1.5.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/openbao/openbao/api/auth/approle/v2 v2.6.0 // indirect
	github.com/openbao/openbao/api/auth/userpass/v2 v2.6.0 // indirect
	github.com/openbao/openbao/api/v2 v2.6.0 // indirect
	github.com/ovh/okms-sdk-go v0.5.3 // indirect
	github.com/passbolt/go-passbolt v0.8.1 // indirect
	github.com/previder/vault-cli v0.1.5 // indirect
	github.com/pulumi/esc-sdk/sdk v0.14.0 // indirect
	github.com/rs/zerolog v1.35.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.36 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/sethvargo/go-password v0.3.1 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sony/gobreaker/v2 v2.4.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/tetratelabs/wabin v0.0.0-20230304001439-f6f874872834 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/volcengine/volc-sdk-golang v1.0.251 // indirect
	github.com/volcengine/volcengine-go-sdk v1.2.42 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/zalando/go-keyring v0.2.8 // indirect
	gitlab.com/gitlab-org/api/client-go v1.46.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.69.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260709172345-9ea1abe57597 // indirect
	golang.org/x/sync v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/ghodss/yaml.v1 v1.0.0 // indirect
	k8s.io/apiserver v0.36.2 // indirect
	k8s.io/code-generator v0.36.2 // indirect
	k8s.io/component-base v0.36.2 // indirect
	k8s.io/gengo/v2 v2.0.0-20260408192533-25e2208e0dc3 // indirect
	k8s.io/kube-openapi v0.0.0-20260706235625-cdb1db5517a0 // indirect
	k8s.io/streaming v0.36.2 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.36.0 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.2 // indirect
	software.sslmate.com/src/go-pkcs12 v0.7.3 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.7 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.1 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.2 // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/go-autorest/tracing v0.6.1 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/PaesslerAG/gval v1.2.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-chef/chef v0.30.1 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/errors v0.22.8 // indirect
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/go-openapi/jsonreference v1.0.0 // indirect
	github.com/go-openapi/swag v0.27.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20260709232956-b9395ee17fa0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.18 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/common v0.70.0 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.15.0
	golang.org/x/tools v0.48.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/gengo v0.0.0-20260408192533-25e2208e0dc3 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.140.0
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
)
