module github.com/external-secrets/external-secrets-e2e

go 1.18

replace github.com/external-secrets/external-secrets => ../

replace (
	github.com/external-secrets/external-secrets v0.0.0 => ../
	github.com/go-check/check => github.com/go-check/check v0.0.0-20180628173108-788fd7840127
	k8s.io/api => k8s.io/api v0.27.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.27.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.27.1
	k8s.io/apiserver => k8s.io/apiserver v0.27.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.27.1
	k8s.io/client-go => k8s.io/client-go v0.27.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.27.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.27.1
	k8s.io/code-generator => k8s.io/code-generator v0.27.1
	k8s.io/component-base => k8s.io/component-base v0.27.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.27.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.27.1
	k8s.io/cri-api => k8s.io/cri-api v0.27.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.27.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.27.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.27.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.27.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.27.1
	k8s.io/kubectl => k8s.io/kubectl v0.27.1
	k8s.io/kubelet => k8s.io/kubelet v0.27.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.27.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.27.1
	k8s.io/metrics => k8s.io/metrics v0.27.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.27.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.27.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.27.1
)

require (
	cloud.google.com/go/secretmanager v1.11.1
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.12
	github.com/DelineaXPM/dsv-sdk-go/v2 v2.1.0
	github.com/akeylesslabs/akeyless-go-cloud-id v0.3.4
	github.com/akeylesslabs/akeyless-go/v3 v3.3.12
	github.com/aliyun/alibaba-cloud-sdk-go v1.62.271
	github.com/aws/aws-sdk-go v1.44.294
	github.com/external-secrets/external-secrets v0.0.0
	github.com/fluxcd/helm-controller/api v0.22.2
	github.com/fluxcd/pkg/apis/meta v0.14.2
	github.com/fluxcd/source-controller/api v0.25.11
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/hashicorp/vault/api v1.9.2
	github.com/onsi/ginkgo/v2 v2.11.0
	github.com/onsi/gomega v1.27.8
	github.com/oracle/oci-go-sdk/v56 v56.1.0
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.17
	github.com/xanzy/go-gitlab v0.86.0
	golang.org/x/oauth2 v0.9.0
	google.golang.org/api v0.129.0
	k8s.io/api v0.27.3
	k8s.io/apiextensions-apiserver v0.27.3
	k8s.io/apimachinery v0.27.3
	k8s.io/client-go v1.5.2
	k8s.io/utils v0.0.0-20230505201702-9f6742963106
	sigs.k8s.io/controller-runtime v0.15.0
	sigs.k8s.io/yaml v1.3.0
	software.sslmate.com/src/go-pkcs12 v0.2.0
)

require (
	cloud.google.com/go/compute v1.20.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.10.2 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fluxcd/pkg/apis/acl v0.0.3 // indirect
	github.com/fluxcd/pkg/apis/kustomize v0.4.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20230602150820-91b7bce49751 // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.11.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.4 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.7 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.10.0 // indirect
	golang.org/x/net v0.11.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	golang.org/x/term v0.9.0 // indirect
	golang.org/x/text v0.10.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.10.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230629202037-9506855d4529 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230629202037-9506855d4529 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230629202037-9506855d4529 // indirect
	google.golang.org/grpc v1.56.1 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	grpc.go4.org v0.0.0-20170609214715-11d0a25b4919 // indirect
	k8s.io/component-base v0.27.3 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230523194449-df37dd07aa00 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
