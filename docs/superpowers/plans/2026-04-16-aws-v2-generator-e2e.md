# AWS V2 Generator E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add AWS v2 generator e2e coverage for `ECRAuthorizationToken` and `STSSessionToken`, and make the default `test.v2` loop execute generator suites in v2 mode.

**Architecture:** Keep the existing generator e2e suite structure and make its global ESO bootstrap v2-aware when `E2E_PROVIDER_MODE=v2` is set. Add AWS-focused generator helpers and v2-labeled cases in the generator suite, then extend the `test.v2` target and its dry-run tests so provider and generator suites run together.

**Tech Stack:** Go, Ginkgo/Gomega, Kubernetes e2e framework, Helm-based test installs, Make

---

### Task 1: Make The Generator Suite Bootstrap V2-Aware

**Files:**
- Modify: `e2e/suites/generator/suite_test.go`

- [ ] **Step 1: Write the failing test expectation as a code diff target**

Add a v2 bootstrap branch in `e2e/suites/generator/suite_test.go` so the generator suite matches the provider suite install path.

```go
var _ = SynchronizedBeforeSuite(func() []byte {
	if framework.IsV2ProviderMode() {
		By("installing eso in generator v2 mode")
		addon.InstallGlobalAddon(addon.NewESO(
			addon.WithCRDs(),
			addon.WithAllowGenericTargets(),
			addon.WithV2Namespace(),
			addon.WithV2KubernetesProvider(),
			addon.WithV2FakeProvider(),
			addon.WithV2AWSProvider(),
		))
		return nil
	}

	cfg := &addon.Config{}
	cfg.KubeConfig, cfg.KubeClientSet, cfg.CRClient = util.NewConfig()

	By("installing eso")
	addon.InstallGlobalAddon(addon.NewESO(addon.WithCRDs(), addon.WithAllowGenericTargets()))

	return nil
}, func([]byte) {
	// noop
})
```

- [ ] **Step 2: Run the affected generator package build to verify the old code path fails the intended review**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e/suites/generator -run TestE2E -count=1
```

Expected: PASS before the change, which confirms this task is a bootstrap behavior change rather than a missing-package compile failure. The red check for this task comes from the next task's explicit v2 generator coverage and from `test.v2` wiring assertions.

- [ ] **Step 3: Write the minimal implementation**

Update `e2e/suites/generator/suite_test.go` to:

- import `github.com/external-secrets/external-secrets-e2e/framework`
- branch on `framework.IsV2ProviderMode()`
- keep `addon.WithAllowGenericTargets()` in both classic and v2 installs
- install AWS, fake, and kubernetes v2 providers only in v2 mode

```go
import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)
```

- [ ] **Step 4: Run the generator package test again to verify it still passes**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e/suites/generator -run TestE2E -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/generator/suite_test.go
git commit -m "Make generator suite bootstrap v2-aware"
```

### Task 2: Add Shared AWS Generator Helpers

**Files:**
- Create: `e2e/suites/generator/aws.go`

- [ ] **Step 1: Write the failing helper usage target**

Define a helper file that both AWS generator cases can use for:

- creating the namespaced AWS credential secret
- building `genv1alpha1.AWSAuth`
- skipping when static credentials are unavailable

```go
package generator

const awsCredsSecretName = "aws-creds"

func skipIfAWSGeneratorCredentialsMissing() {
	if os.Getenv("AWS_REGION") == "" || os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		Skip("AWS static generator credentials are required")
	}
}

func createAWSGeneratorCredentialsSecret(f *framework.Framework) {
	err := f.CRClient.Create(GinkgoT().Context(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      awsCredsSecretName,
			Namespace: f.Namespace.Name,
		},
		Data: map[string][]byte{
			"akid": []byte(os.Getenv("AWS_ACCESS_KEY_ID")),
			"sak":  []byte(os.Getenv("AWS_SECRET_ACCESS_KEY")),
			"st":   []byte(os.Getenv("AWS_SESSION_TOKEN")),
		},
	})
	Expect(err).ToNot(HaveOccurred())
}

func awsGeneratorAuth() genv1alpha1.AWSAuth {
	auth := genv1alpha1.AWSAuth{
		SecretRef: &genv1alpha1.AWSAuthSecretRef{
			AccessKeyID: esmeta.SecretKeySelector{Name: awsCredsSecretName, Key: "akid"},
			SecretAccessKey: esmeta.SecretKeySelector{Name: awsCredsSecretName, Key: "sak"},
		},
	}
	if os.Getenv("AWS_SESSION_TOKEN") != "" {
		auth.SecretRef.SessionToken = &esmeta.SecretKeySelector{Name: awsCredsSecretName, Key: "st"}
	}
	return auth
}
```

- [ ] **Step 2: Run a targeted package compile to verify the helper file is still absent before implementation**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e/suites/generator -run TestE2E -count=1
```

Expected: PASS before implementation because no callers exist yet. The helper becomes necessary in Tasks 3 and 4.

- [ ] **Step 3: Write the minimal implementation**

Create `e2e/suites/generator/aws.go` with:

- package-local constant `awsCredsSecretName`
- `skipIfAWSGeneratorCredentialsMissing()`
- `createAWSGeneratorCredentialsSecret(f *framework.Framework)`
- `awsGeneratorAuth() genv1alpha1.AWSAuth`

Imports should include:

```go
import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)
```

- [ ] **Step 4: Run the package tests to verify the helper file compiles cleanly**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e/suites/generator -run TestE2E -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/generator/aws.go
git commit -m "Add shared AWS generator e2e helpers"
```

### Task 3: Add AWS V2 ECR Generator Coverage

**Files:**
- Create: `e2e/suites/generator/ecr_v2.go`
- Read for reference: `e2e/suites/generator/ecr.go`
- Read for reference: `providers/v2/aws/generator/ecr.go`

- [ ] **Step 1: Write the failing v2 ECR generator test**

Add a new v2-only AWS-labeled test file:

```go
var _ = Describe("ecr generator v2", Label("aws", "ecr", "v2"), func() {
	f := framework.New("ecr-v2")

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		skipIfAWSGeneratorCredentialsMissing()
	})

	injectGenerator := func(tc *testCase) {
		createAWSGeneratorCredentialsSecret(f)
		tc.Generator = &genv1alpha1.ECRAuthorizationToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.ECRAuthorizationTokenKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.ECRAuthorizationTokenSpec{
				Region: os.Getenv("AWS_REGION"),
				Auth:   awsGeneratorAuth(),
			},
		}
	}

	customResourceGenerator := func(tc *testCase) {
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{{
			SourceRef: &esv1.StoreGeneratorSourceRef{
				GeneratorRef: &esv1.GeneratorRef{
					Kind: "ECRAuthorizationToken",
					Name: generatorName,
				},
			},
		}}
		tc.AfterSync = func(secret *v1.Secret) {
			Expect(string(secret.Data["username"])).To(Equal("AWS"))
			Expect(string(secret.Data["password"])).ToNot(BeEmpty())
			Expect(string(secret.Data["proxy_endpoint"])).ToNot(BeEmpty())
			Expect(string(secret.Data["expires_at"])).ToNot(BeEmpty())
		}
	}

	DescribeTable("generate ecr auth tokens through the v2 aws provider", generatorTableFunc,
		Entry("using custom resource generator", f, injectGenerator, customResourceGenerator),
	)
})
```

- [ ] **Step 2: Run the focused v2 generator e2e selection**

Run:

```bash
cd e2e && TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 GINKGO_LABELS='aws && v2 && ecr' TEST_SUITES='generator'
```

Expected: if Task 1 has not landed yet, FAIL because the generator suite still installs classic ESO without the v2 provider sidecars. After Task 1 and Task 2, this command becomes the first green verification for the new ECR v2 coverage.

- [ ] **Step 3: Write the minimal implementation**

Create `e2e/suites/generator/ecr_v2.go` with:

- `Label("aws", "ecr", "v2")`
- `framework.IsV2ProviderMode()` gate
- shared AWS helper usage
- assertions for `username`, `password`, `proxy_endpoint`, and `expires_at`

Imports should include:

```go
import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)
```

- [ ] **Step 4: Run the generator package tests to verify the new case compiles**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e/suites/generator -run TestE2E -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/generator/ecr_v2.go e2e/suites/generator/aws.go
git commit -m "Add AWS v2 ECR generator e2e coverage"
```

### Task 4: Add AWS V2 STS Generator Coverage

**Files:**
- Create: `e2e/suites/generator/sts_v2.go`
- Read for reference: `providers/v2/aws/generator/sts.go`
- Read for reference: `apis/generators/v1alpha1/types_sts.go`

- [ ] **Step 1: Write the failing v2 STS generator test**

Create a new STS generator case that validates the output contract from `providers/v2/aws/generator/sts.go`:

```go
var _ = Describe("sts generator v2", Label("aws", "sts", "v2"), func() {
	f := framework.New("sts-v2")

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		skipIfAWSGeneratorCredentialsMissing()
	})

	injectGenerator := func(tc *testCase) {
		createAWSGeneratorCredentialsSecret(f)
		tc.Generator = &genv1alpha1.STSSessionToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.STSSessionTokenKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.STSSessionTokenSpec{
				Region: os.Getenv("AWS_REGION"),
				Auth:   awsGeneratorAuth(),
			},
		}
	}

	customResourceGenerator := func(tc *testCase) {
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{{
			SourceRef: &esv1.StoreGeneratorSourceRef{
				GeneratorRef: &esv1.GeneratorRef{
					Kind: "STSSessionToken",
					Name: generatorName,
				},
			},
		}}
		tc.AfterSync = func(secret *v1.Secret) {
			Expect(string(secret.Data["access_key_id"])).ToNot(BeEmpty())
			Expect(string(secret.Data["secret_access_key"])).ToNot(BeEmpty())
			Expect(string(secret.Data["session_token"])).ToNot(BeEmpty())
			Expect(string(secret.Data["expiration"])).ToNot(BeEmpty())
		}
	}

	DescribeTable("generate sts session tokens through the v2 aws provider", generatorTableFunc,
		Entry("using custom resource generator", f, injectGenerator, customResourceGenerator),
	)
})
```

- [ ] **Step 2: Run the focused v2 generator e2e selection**

Run:

```bash
cd e2e && TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 GINKGO_LABELS='aws && v2 && sts' TEST_SUITES='generator'
```

Expected: PASS once the new test and shared helpers are in place. If it fails, the failure should come from STS generator wiring or assertion mismatches, not from classic-mode skips.

- [ ] **Step 3: Write the minimal implementation**

Create `e2e/suites/generator/sts_v2.go` with:

- `Label("aws", "sts", "v2")`
- shared AWS helper usage
- `framework.IsV2ProviderMode()` guard
- assertions for the four STS generator output keys

Imports should include:

```go
import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)
```

- [ ] **Step 4: Run the generator package tests to verify the new STS case compiles**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e/suites/generator -run TestE2E -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/generator/sts_v2.go
git commit -m "Add AWS v2 STS generator e2e coverage"
```

### Task 5: Include Generator Suites In The Default V2 Loop

**Files:**
- Modify: `e2e/Makefile`
- Modify: `e2e/makefile_test.go`

- [ ] **Step 1: Write the failing dry-run assertions**

Update `e2e/makefile_test.go` so `test.v2` expects the generated run command to include `TEST_SUITES="provider generator"`.

Add assertions like:

```go
if !strings.Contains(defaultDryRun, `TEST_SUITES="provider generator"`) {
	t.Fatalf("expected default test.v2 dry-run to run provider and generator suites, output:\n%s", defaultDryRun)
}
```

and for the skipped build case:

```go
if !strings.Contains(skippedDryRun, `TEST_SUITES="provider generator"`) {
	t.Fatalf("expected skipped test.v2 dry-run to still run provider and generator suites, output:\n%s", skippedDryRun)
}
```

- [ ] **Step 2: Run the targeted makefile tests to verify they fail**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e -run 'TestV2MakeTarget' -count=1
```

Expected: FAIL because `e2e/Makefile` still uses `TEST_SUITES="provider"`

- [ ] **Step 3: Write the minimal implementation**

Change `e2e/Makefile`:

```make
KUBECTL_CONTEXT="$(KIND_CONTEXT)" GINKGO_LABELS="$(V2_GINKGO_LABELS)" E2E_PROVIDER_MODE="v2" TEST_SUITES="provider generator" E2E_SKIP_HELM_DEPENDENCY_UPDATE="true" ./run.sh
```

Update `e2e/makefile_test.go` to assert the new suite list in both v2 dry-run paths.

- [ ] **Step 4: Run the makefile tests to verify they pass**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e -run 'TestV2MakeTarget|TestClassicMakeTarget|TestV2MakeTargetPrunesDockerImagesInCI' -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/Makefile e2e/makefile_test.go
git commit -m "Run generator suites in the default v2 e2e target"
```

### Task 6: Verify AWS V2 Generator Coverage End-To-End

**Files:**
- Verify: `e2e/suites/generator/aws.go`
- Verify: `e2e/suites/generator/ecr_v2.go`
- Verify: `e2e/suites/generator/sts_v2.go`
- Verify: `e2e/suites/generator/suite_test.go`
- Verify: `e2e/Makefile`
- Verify: `e2e/makefile_test.go`

- [ ] **Step 1: Run the focused Go verification**

Run:

```bash
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./e2e ./e2e/suites/generator -count=1
```

Expected: PASS

- [ ] **Step 2: Run the focused v2 generator e2e labels**

Run:

```bash
cd e2e && TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 GINKGO_LABELS='aws && v2 && (ecr || sts)' TEST_SUITES='generator'
```

Expected: PASS

- [ ] **Step 3: Run the default v2 target smoke verification**

Run:

```bash
cd e2e && TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2
```

Expected: PASS for provider and generator suites with managed tests excluded by `V2_GINKGO_LABELS`

- [ ] **Step 4: Review the final diff and commit**

Run:

```bash
git status --short
git diff --stat
```

Expected: only the planned generator/bootstrap/makefile files are changed

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/generator/suite_test.go e2e/suites/generator/aws.go e2e/suites/generator/ecr_v2.go e2e/suites/generator/sts_v2.go e2e/Makefile e2e/makefile_test.go
git commit -m "Add AWS v2 generator e2e coverage"
```
