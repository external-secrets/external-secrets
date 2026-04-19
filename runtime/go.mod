module github.com/external-secrets/external-secrets/runtime

go 1.26.2

require (
	dario.cat/mergo v1.0.1
	github.com/Masterminds/goutils v1.1.1
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/aws/aws-sdk-go-v2 v1.39.3
	github.com/external-secrets/external-secrets/apis v0.0.0
	github.com/external-secrets/external-secrets/proto v0.0.0
	github.com/external-secrets/external-secrets/providers/v2/adapter/store v0.0.0
	github.com/external-secrets/external-secrets/providers/v2/common v0.0.0
	github.com/go-logr/logr v1.4.3
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/golang-lru v1.0.2
	github.com/huandu/xstrings v1.5.0
	github.com/lestrrat-go/jwx/v2 v2.1.6
	github.com/mitchellh/copystructure v1.2.0
	github.com/oracle/oci-go-sdk/v65 v65.102.1
	github.com/prometheus/client_golang v1.23.2
	github.com/shopspring/decimal v1.4.0
	github.com/spf13/cast v1.7.0
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.49.0
	google.golang.org/grpc v1.79.3
	k8s.io/api v0.35.2
	k8s.io/apiextensions-apiserver v0.35.2
	k8s.io/apimachinery v0.35.2
	k8s.io/client-go v0.35.2
	sigs.k8s.io/controller-runtime v0.23.3
	sigs.k8s.io/yaml v1.6.0
	software.sslmate.com/src/go-pkcs12 v0.6.0
)

require (
	github.com/aws/smithy-go v1.23.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/jsonreference v0.21.5 // indirect
	github.com/go-openapi/swag v0.25.5 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.5 // indirect
	github.com/go-openapi/swag/conv v0.25.5 // indirect
	github.com/go-openapi/swag/fileutils v0.25.5 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.5 // indirect
	github.com/go-openapi/swag/loading v0.25.5 // indirect
	github.com/go-openapi/swag/mangling v0.25.5 // indirect
	github.com/go-openapi/swag/netutils v0.25.5 // indirect
	github.com/go-openapi/swag/stringutils v0.25.5 // indirect
	github.com/go-openapi/swag/typeutils v0.25.5 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.5 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/gofrs/flock v0.10.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lestrrat-go/blackmagic v1.0.3 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260316180232-0b37fe3546d5 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20260304202019-5b3e3fdb0acf // indirect
	k8s.io/utils v0.0.0-20260210185600-b8788abfbbc2 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2 // indirect
)

replace (
	github.com/external-secrets/external-secrets/apis => ../apis
	github.com/external-secrets/external-secrets/proto => ../providers/v2/common/proto
	github.com/external-secrets/external-secrets/providers/v2/adapter/store => ../providers/v2/adapter/store
	github.com/external-secrets/external-secrets/providers/v2/common => ../providers/v2/common
)
