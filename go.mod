module github.com/external-secrets/external-secrets

go 1.23

replace github.com/Masterminds/sprig/v3 => github.com/external-secrets/sprig/v3 v3.3.0

require (
	cloud.google.com/go/iam v1.2.0
	cloud.google.com/go/secretmanager v1.14.0
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.29
	github.com/Azure/go-autorest/autorest/adal v0.9.24
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.13
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2
	github.com/IBM/go-sdk-core/v5 v5.17.4
	github.com/IBM/secrets-manager-go-sdk/v2 v2.0.5
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.0
	github.com/akeylesslabs/akeyless-go-cloud-id v0.3.5
	github.com/aws/aws-sdk-go v1.55.5
	github.com/go-logr/logr v1.4.2
	github.com/go-test/deep v1.0.4 // indirect
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.13.0
	github.com/hashicorp/vault/api v1.14.0
	github.com/hashicorp/vault/api/auth/approle v0.7.0
	github.com/hashicorp/vault/api/auth/kubernetes v0.7.0
	github.com/hashicorp/vault/api/auth/ldap v0.7.0
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/onsi/ginkgo/v2 v2.20.2
	github.com/onsi/gomega v1.34.1
	github.com/oracle/oci-go-sdk/v65 v65.73.0
	github.com/prometheus/client_golang v1.20.2
	github.com/prometheus/client_model v0.6.1
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/gjson v1.17.3
	github.com/xanzy/go-gitlab v0.108.0
	github.com/yandex-cloud/go-genproto v0.0.0-20240829130658-0568052c5a6a
	github.com/yandex-cloud/go-sdk v0.0.0-20240829131820-fa8ad79f88a4
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.26.0
	golang.org/x/oauth2 v0.22.0
	google.golang.org/api v0.195.0
	google.golang.org/genproto v0.0.0-20240827150818-7e3bb234dfed
	google.golang.org/grpc v1.66.0
	gopkg.in/yaml.v3 v3.0.1
	grpc.go4.org v0.0.0-20170609214715-11d0a25b4919
	k8s.io/api v0.31.0
	k8s.io/apiextensions-apiserver v0.31.0
	k8s.io/apimachinery v0.31.0
	k8s.io/client-go v0.31.0
	k8s.io/utils v0.0.0-20240821151609-f90d01438635
	sigs.k8s.io/controller-runtime v0.19.0
	sigs.k8s.io/controller-tools v0.16.2
)

require github.com/1Password/connect-sdk-go v1.5.3

require (
	dario.cat/mergo v1.0.1
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.14.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0
	github.com/BeyondTrust/go-client-library-passwordsafe v0.6.0
	github.com/DelineaXPM/dsv-sdk-go/v2 v2.1.2
	github.com/DelineaXPM/tss-sdk-go/v2 v2.0.1
	github.com/Onboardbase/go-cryptojs-aes-decrypt v0.0.0-20230430095000-27c0d3a9016d
	github.com/akeylesslabs/akeyless-go/v3 v3.6.3
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.0.9
	github.com/alibabacloud-go/kms-20160120/v3 v3.2.3
	github.com/alibabacloud-go/openapi-util v0.1.1
	github.com/alibabacloud-go/tea v1.2.2
	github.com/alibabacloud-go/tea-utils/v2 v2.0.6
	github.com/aliyun/credentials-go v1.3.9
	github.com/avast/retry-go/v4 v4.6.0
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/cyberark/conjur-api-go v0.12.4
	github.com/fortanix/sdkms-client-go v0.4.0
	github.com/go-openapi/strfmt v0.23.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/hashicorp/golang-lru v1.0.2
	github.com/hashicorp/vault/api/auth/aws v0.7.0
	github.com/hashicorp/vault/api/auth/userpass v0.7.0
	github.com/keeper-security/secrets-manager-go/core v1.6.3
	github.com/lestrrat-go/jwx/v2 v2.1.1
	github.com/maxbrunsfeld/counterfeiter/v6 v6.8.1
	github.com/passbolt/go-passbolt v0.7.1
	github.com/pulumi/esc-sdk/sdk v0.9.2
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.29
	github.com/sethvargo/go-password v0.3.1
	github.com/spf13/pflag v1.0.5
	github.com/tidwall/sjson v1.2.5
	k8s.io/kube-openapi v0.0.0-20240827152857-f7e401e7b4c2
	sigs.k8s.io/yaml v1.4.0
	software.sslmate.com/src/go-pkcs12 v0.4.0
)

require (
	cloud.google.com/go/auth v0.9.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.4 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/ProtonMail/go-mime v0.0.0-20230322103455-7d82a3887f2f // indirect
	github.com/ProtonMail/gopenpgp/v2 v2.7.5 // indirect
	github.com/alessio/shellescape v1.4.2 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-pop v0.0.6 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/darabonba-array v0.1.0 // indirect
	github.com/alibabacloud-go/darabonba-encode-util v0.0.2 // indirect
	github.com/alibabacloud-go/darabonba-map v0.0.2 // indirect
	github.com/alibabacloud-go/darabonba-signature-util v0.0.7 // indirect
	github.com/alibabacloud-go/darabonba-string v1.0.2 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/endpoint-util v1.1.1 // indirect
	github.com/alibabacloud-go/tea-xml v1.1.3 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/cloudflare/circl v1.4.0 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.5 // indirect
	github.com/go-jose/go-jose/v4 v4.0.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/validator/v10 v10.22.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/hashicorp/go-secure-stdlib/awsutil v0.3.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/zalando/go-keyring v0.2.5 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.54.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240827150818-7e3bb234dfed // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240827150818-7e3bb234dfed // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/ghodss/yaml.v1 v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/PaesslerAG/gval v1.2.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-chef/chef v0.30.1
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect; indirectgithub.com/go-openapi/strfmt v0.21.7 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gobuffalo/flect v1.0.2 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20240829160300-da1f7e9f2b25 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.8 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/common v0.57.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.mongodb.org/mongo-driver v1.16.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/term v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/gengo v0.0.0-20240826214909-a7b603a56eb7 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
)
