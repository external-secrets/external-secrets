module github.com/external-secrets/external-secrets

go 1.25.6

replace (
	github.com/external-secrets/external-secrets/apis => ./apis
	github.com/external-secrets/external-secrets/generators/v1/acr => ./generators/v1/acr
	github.com/external-secrets/external-secrets/generators/v1/cloudsmith => ./generators/v1/cloudsmith
	github.com/external-secrets/external-secrets/generators/v1/ecr => ./generators/v1/ecr
	github.com/external-secrets/external-secrets/generators/v1/fake => ./generators/v1/fake
	github.com/external-secrets/external-secrets/generators/v1/gcr => ./generators/v1/gcr
	github.com/external-secrets/external-secrets/generators/v1/github => ./generators/v1/github
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
	github.com/external-secrets/external-secrets/providers/v1/ngrok => ./providers/v1/ngrok
	github.com/external-secrets/external-secrets/providers/v1/onboardbase => ./providers/v1/onboardbase
	github.com/external-secrets/external-secrets/providers/v1/onepassword => ./providers/v1/onepassword
	github.com/external-secrets/external-secrets/providers/v1/onepasswordsdk => ./providers/v1/onepasswordsdk
	github.com/external-secrets/external-secrets/providers/v1/oracle => ./providers/v1/oracle
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
	cloud.google.com/go/iam v1.5.3 // indirect
	cloud.google.com/go/secretmanager v1.16.0 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.30 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.13 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.5.0 // indirect
	github.com/IBM/go-sdk-core/v5 v5.21.0 // indirect
	github.com/IBM/secrets-manager-go-sdk/v2 v2.0.16 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.0
	github.com/akeylesslabs/akeyless-go-cloud-id v0.3.5 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/go-logr/logr v1.4.3
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/hashicorp/vault/api v1.22.0 // indirect
	github.com/hashicorp/vault/api/auth/approle v0.11.0 // indirect
	github.com/hashicorp/vault/api/auth/kubernetes v0.10.0 // indirect
	github.com/hashicorp/vault/api/auth/ldap v0.11.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/onsi/ginkgo/v2 v2.27.2
	github.com/onsi/gomega v1.38.2
	github.com/oracle/oci-go-sdk/v65 v65.103.0 // indirect
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/yandex-cloud/go-genproto v0.34.0 // indirect
	github.com/yandex-cloud/go-sdk v0.27.0 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/oauth2 v0.32.0 // indirect
	google.golang.org/api v0.254.0 // indirect
	google.golang.org/genproto v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/grpc v1.76.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	grpc.go4.org v0.0.0-20170609214715-11d0a25b4919 // indirect
	k8s.io/api v0.34.1
	k8s.io/apiextensions-apiserver v0.34.1
	k8s.io/apimachinery v0.34.1
	k8s.io/client-go v0.34.1
	k8s.io/utils v0.0.0-20251002143259-bc988d571ff4
	sigs.k8s.io/controller-runtime v0.22.3
	sigs.k8s.io/controller-tools v0.19.0
)

require github.com/1Password/connect-sdk-go v1.5.3 // indirect

require (
	github.com/external-secrets/external-secrets/apis v0.0.0
	github.com/external-secrets/external-secrets/generators/v1/acr v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/cloudsmith v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/ecr v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/fake v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/gcr v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/github v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/grafana v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/mfa v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/password v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/quay v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/sshkey v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/sts v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/uuid v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/vault v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/generators/v1/webhook v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/akeyless v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/aws v0.0.0-20251103072335-a9b233b6936f
	github.com/external-secrets/external-secrets/providers/v1/azure v0.0.0-20251103072335-a9b233b6936f
	github.com/external-secrets/external-secrets/providers/v1/barbican v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/beyondtrust v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/bitwarden v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/chef v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/cloudru v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/conjur v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/delinea v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/doppler v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/dvls v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/fake v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/fortanix v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/gcp v0.0.0-20251104073127-4d2c8fd13e10
	github.com/external-secrets/external-secrets/providers/v1/github v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/gitlab v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/ibm v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/infisical v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/keepersecurity v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/kubernetes v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/ngrok v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/onboardbase v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/onepassword v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/onepasswordsdk v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/oracle v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/passbolt v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/passworddepot v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/previder v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/pulumi v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/scaleway v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/secretserver v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/senhasegura v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/vault v0.0.0-20251103080423-08fa383f42e5
	github.com/external-secrets/external-secrets/providers/v1/volcengine v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/providers/v1/webhook v0.0.0-20251103080423-08fa383f42e5
	github.com/external-secrets/external-secrets/providers/v1/yandex v0.0.0-00010101000000-000000000000
	github.com/external-secrets/external-secrets/runtime v0.0.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.12.0
	sigs.k8s.io/yaml v1.6.0
)

require (
	al.essio.dev/pkg/shellescape v1.6.0 // indirect
	cloud.google.com/go/auth v0.17.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/1password/onepassword-sdk-go v0.3.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.19.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates v1.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys v1.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/internal v1.2.0 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/BeyondTrust/go-client-library-passwordsafe v1.0.0 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/DelineaXPM/dsv-sdk-go/v2 v2.2.0 // indirect
	github.com/DelineaXPM/tss-sdk-go/v3 v3.0.1 // indirect
	github.com/Devolutions/go-dvls v0.15.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Onboardbase/go-cryptojs-aes-decrypt v0.0.0-20230430095000-27c0d3a9016d // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/ProtonMail/go-mime v0.0.0-20230322103455-7d82a3887f2f // indirect
	github.com/ProtonMail/gopenpgp/v2 v2.9.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/akeylesslabs/akeyless-go/v4 v4.3.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aws/aws-sdk-go-v2 v1.39.6 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.19 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.51.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.38.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.39.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.67.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.40.1 // indirect
	github.com/aws/smithy-go v1.23.2 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/bradleyfalzon/ghinstallation/v2 v2.17.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/charmbracelet/bubbles v0.21.0 // indirect
	github.com/charmbracelet/bubbletea v1.3.10 // indirect
	github.com/charmbracelet/colorprofile v0.3.2 // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.10.2 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/cheggaaa/pb v1.0.29 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cloudru-tech/iam-sdk v1.0.4 // indirect
	github.com/cloudru-tech/secret-manager-sdk v1.1.1 // indirect
	github.com/cyberark/conjur-api-go v0.13.8 // indirect
	github.com/cyphar/filepath-securejoin v0.6.0 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/djherbis/times v1.6.0 // indirect
	github.com/dylibso/observe-sdk/go v0.0.0-20240828172851-9145d8ad07e1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/extism/go-sdk v1.7.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fortanix/sdkms-client-go v0.4.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.11 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-git/go-git/v5 v5.16.3 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.24.0 // indirect
	github.com/go-openapi/loads v0.23.1 // indirect
	github.com/go-openapi/runtime v0.29.0 // indirect
	github.com/go-openapi/spec v0.22.0 // indirect
	github.com/go-openapi/strfmt v0.24.0 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.1 // indirect
	github.com/go-openapi/swag/conv v0.25.1 // indirect
	github.com/go-openapi/swag/fileutils v0.25.1 // indirect
	github.com/go-openapi/swag/jsonname v0.25.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.1 // indirect
	github.com/go-openapi/swag/loading v0.25.1 // indirect
	github.com/go-openapi/swag/mangling v0.25.1 // indirect
	github.com/go-openapi/swag/netutils v0.25.1 // indirect
	github.com/go-openapi/swag/stringutils v0.25.1 // indirect
	github.com/go-openapi/swag/typeutils v0.25.1 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.1 // indirect
	github.com/go-openapi/validate v0.25.0 // indirect
	github.com/go-playground/validator/v10 v10.28.0 // indirect
	github.com/go-resty/resty/v2 v2.16.5 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/golang/glog v1.2.5 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-github/v56 v56.0.0 // indirect
	github.com/google/go-github/v75 v75.0.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/gophercloud/gophercloud/v2 v2.8.0 // indirect
	github.com/grafana/grafana-openapi-client-go v0.0.0-20250925215610-d92957c70d5c // indirect
	github.com/hashicorp/go-secure-stdlib/awsutil v0.3.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	github.com/hashicorp/vault/api/auth/aws v0.11.0 // indirect
	github.com/hashicorp/vault/api/auth/gcp v0.11.0 // indirect
	github.com/hashicorp/vault/api/auth/userpass v0.11.0 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20250628045327-2d64ad6b7ec5 // indirect
	github.com/infisical/go-sdk v0.5.100 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/keeper-security/secrets-manager-go/core v1.6.4 // indirect
	github.com/kevinburke/ssh_config v1.4.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.6 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/ngrok/ngrok-api-go/v7 v7.6.0 // indirect
	github.com/opentracing/basictracer-go v1.1.0 // indirect
	github.com/passbolt/go-passbolt v0.7.2 // indirect
	github.com/pgavlin/fx v0.1.6 // indirect
	github.com/pgavlin/fx/v2 v2.0.12 // indirect
	github.com/pjbgf/sha1cd v0.5.0 // indirect
	github.com/previder/vault-cli v0.1.3 // indirect
	github.com/pulumi/appdash v0.0.0-20231130102222-75f619a67231 // indirect
	github.com/pulumi/esc v0.19.0 // indirect
	github.com/pulumi/esc-sdk/sdk v0.12.3 // indirect
	github.com/pulumi/pulumi/sdk/v3 v3.205.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.35 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/sethvargo/go-password v0.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skeema/knownhosts v1.3.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/tetratelabs/wabin v0.0.0-20230304001439-f6f874872834 // indirect
	github.com/tetratelabs/wazero v1.9.0 // indirect
	github.com/texttheater/golang-levenshtein v1.0.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/volcengine/volc-sdk-golang v1.0.225 // indirect
	github.com/volcengine/volcengine-go-sdk v1.1.46 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zalando/go-keyring v0.2.6 // indirect
	github.com/zclconf/go-cty v1.17.0 // indirect
	gitlab.com/gitlab-org/api/client-go v0.157.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.8.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/sync v0.19.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/ghodss/yaml.v1 v1.0.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/code-generator v0.34.1 // indirect
	k8s.io/gengo/v2 v2.0.0-20250922181213-ec3ebc5fd46b // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	lukechampine.com/frand v1.5.1 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
	software.sslmate.com/src/go-pkcs12 v0.6.0 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.7 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.1 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.2 // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/go-autorest/tracing v0.6.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/PaesslerAG/gval v1.2.4 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-chef/chef v0.30.1 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/errors v0.22.3 // indirect
	github.com/go-openapi/jsonpointer v0.22.1 // indirect
	github.com/go-openapi/jsonreference v0.21.2 // indirect; indirectgithub.com/go-openapi/strfmt v0.21.7 // indirect
	github.com/go-openapi/swag v0.25.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/pprof v0.0.0-20251007162407-5df77e3f7d1d // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
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
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/common v0.67.2 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.mongodb.org/mongo-driver v1.17.6 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/time v0.14.0
	golang.org/x/tools v0.39.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/gengo v0.0.0-20250922181213-ec3ebc5fd46b // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.130.1
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
)
