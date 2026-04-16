# V2 Operational E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add reusable v2 operational e2e coverage for provider outage, restart, readiness recovery, and connection-scaling behavior across read, push, and generator-backed flows, then wire at least one real operational scenario into CI.

**Architecture:** Extend the existing v2 e2e framework with shared pod-disruption and metrics helpers, then build provider-agnostic operational scenario builders in `e2e/suites/provider/cases/common`. Use provider-specific harnesses for `fake` and `kubernetes` to opt into shared scenarios, add fake-backed v2 generator outage coverage in the generator suite, and finish with a focused `test.e2e.v2.operational` CI slice that runs a deterministic real operational scenario.

**Tech Stack:** Go, Ginkgo/Gomega, Kubernetes kind e2e framework, controller-runtime client, Prometheus metrics scraping, GitHub Actions

---

## File Map

- Create: `e2e/framework/v2/operational.go`
- Create: `e2e/framework/v2/metrics_test.go`
- Modify: `e2e/framework/v2/helpers.go`
- Modify: `e2e/framework/v2/metrics.go`
- Create: `e2e/suites/provider/cases/common/operational_v2.go`
- Create: `e2e/suites/provider/cases/common/operational_v2_test.go`
- Create: `e2e/suites/provider/cases/fake/operational_v2.go`
- Modify: `e2e/suites/provider/cases/fake/provider_v2.go`
- Modify: `e2e/suites/provider/cases/fake/provider_v2_test.go`
- Create: `e2e/suites/provider/cases/kubernetes/operational_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/provider_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/clusterprovider_v2.go`
- Create: `e2e/suites/generator/operational_v2.go`
- Modify: `e2e/suites/generator/testcase.go`
- Modify: `e2e/Makefile`
- Modify: `e2e/makefile_test.go`
- Modify: `Makefile`
- Modify: `.github/actions/e2e/action.yml`
- Modify: `.github/workflows/e2e.yml`

### Task 1: Add Shared V2 Operational Framework Helpers

**Files:**
- Create: `e2e/framework/v2/operational.go`
- Create: `e2e/framework/v2/metrics_test.go`
- Modify: `e2e/framework/v2/helpers.go`
- Modify: `e2e/framework/v2/metrics.go`

- [ ] **Step 1: Write the failing metric-helper tests**

Create `e2e/framework/v2/metrics_test.go` with focused tests for the metric helpers the operational suite will need.

```go
package v2

import "testing"

func TestSumMetricValues(t *testing.T) {
	metrics := MetricsMap{
		"grpc_pool_connections_total": {
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 1},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 2},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-b"}, Value: 4},
		},
	}

	got := SumMetricValues(metrics, "grpc_pool_connections_total", map[string]string{"address": "provider-a"})
	if got != 3 {
		t.Fatalf("expected sum 3, got %v", got)
	}
}

func TestCountMetricSamples(t *testing.T) {
	metrics := MetricsMap{
		"grpc_pool_connections_total": {
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 1},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 2},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-b"}, Value: 4},
		},
	}

	got := CountMetricSamples(metrics, "grpc_pool_connections_total", map[string]string{"address": "provider-a"})
	if got != 2 {
		t.Fatalf("expected count 2, got %d", got)
	}
}
```

- [ ] **Step 2: Run the focused framework tests to verify they fail**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./framework/v2 -run 'TestSumMetricValues|TestCountMetricSamples' -count=1
```

Expected: FAIL with `undefined: SumMetricValues` and `undefined: CountMetricSamples`

- [ ] **Step 3: Write the minimal framework implementation**

Create `e2e/framework/v2/operational.go` with pod and deployment helpers that the provider harnesses can reuse.

```go
package v2

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type BackendTarget struct {
	Namespace        string
	DeploymentName   string
	PodLabelSelector string
}

func WaitForClusterProviderNotReady(f *framework.Framework, name string, timeout time.Duration) *esv1.ClusterProvider {
	return WaitForClusterProviderCondition(f, name, metav1.ConditionFalse, timeout)
}

func WaitForClusterProviderCondition(f *framework.Framework, name string, status metav1.ConditionStatus, timeout time.Duration) *esv1.ClusterProvider {
	var clusterProvider esv1.ClusterProvider
	Eventually(func() bool {
		err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: name}, &clusterProvider)
		if err != nil {
			return false
		}
		for _, condition := range clusterProvider.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == status {
				return true
			}
		}
		return false
	}, timeout, time.Second).Should(BeTrue())
	return &clusterProvider
}

func ScaleDeployment(f *framework.Framework, namespace, name string, replicas int32) {
	var deployment appsv1.Deployment
	Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, &deployment)).To(Succeed())
	deployment.Spec.Replicas = &replicas
	Expect(f.CRClient.Update(context.Background(), &deployment)).To(Succeed())
}

func DeleteOneProviderPod(f *framework.Framework, namespace, labelSelector string) {
	var podList corev1.PodList
	Expect(f.CRClient.List(context.Background(), &podList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{}),
	})).To(Succeed())
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			Expect(f.CRClient.Delete(context.Background(), &pod)).To(Succeed())
			return
		}
	}
	Fail(fmt.Sprintf("no running pod found for selector %s", labelSelector))
}
```

Modify `e2e/framework/v2/metrics.go` to add bounded-scaling helpers.

```go
func SumMetricValues(metrics MetricsMap, metricName string, matchLabels map[string]string) float64 {
	samples, exists := metrics[metricName]
	if !exists {
		return 0
	}
	var total float64
	for _, sample := range samples {
		if labelsMatch(sample.Labels, matchLabels) {
			total += sample.Value
		}
	}
	return total
}

func CountMetricSamples(metrics MetricsMap, metricName string, matchLabels map[string]string) int {
	samples, exists := metrics[metricName]
	if !exists {
		return 0
	}
	count := 0
	for _, sample := range samples {
		if labelsMatch(sample.Labels, matchLabels) {
			count++
		}
	}
	return count
}
```

Also add the missing imports in `e2e/framework/v2/operational.go`:

```go
import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)
```

- [ ] **Step 4: Run the framework tests again to verify they pass**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./framework/v2 -run 'TestSumMetricValues|TestCountMetricSamples|TestCreateKubernetesProviderUsesProvidedCABundle|TestGetClusterCABundleWaitsForRootCAConfigMap' -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/framework/v2/operational.go e2e/framework/v2/metrics.go e2e/framework/v2/metrics_test.go e2e/framework/v2/helpers.go
git commit -m "Add v2 operational e2e framework helpers"
```

### Task 2: Add Shared Operational Scenario Builders

**Files:**
- Create: `e2e/suites/provider/cases/common/operational_v2.go`
- Create: `e2e/suites/provider/cases/common/operational_v2_test.go`

- [ ] **Step 1: Write the failing shared-runtime tests**

Create `e2e/suites/provider/cases/common/operational_v2_test.go`.

```go
package common

import "testing"

func TestOperationalRuntimeSupportsDisruptionLifecycle(t *testing.T) {
	runtimeWithoutHooks := &OperationalRuntime{}
	if runtimeWithoutHooks.SupportsDisruptionLifecycle() {
		t.Fatalf("expected false when all hooks are nil")
	}

	runtimeWithBreakOnly := &OperationalRuntime{
		MakeUnavailable: func() {},
	}
	if runtimeWithBreakOnly.SupportsDisruptionLifecycle() {
		t.Fatalf("expected false when Restore is nil")
	}

	runtimeWithRestoreOnly := &OperationalRuntime{
		RestoreAvailability: func() {},
	}
	if runtimeWithRestoreOnly.SupportsDisruptionLifecycle() {
		t.Fatalf("expected false when MakeUnavailable is nil")
	}

	runtimeWithBoth := &OperationalRuntime{
		MakeUnavailable:     func() {},
		RestoreAvailability: func() {},
	}
	if !runtimeWithBoth.SupportsDisruptionLifecycle() {
		t.Fatalf("expected true when both hooks exist")
	}
}

func TestOperationalRuntimeSupportsRestart(t *testing.T) {
	runtime := &OperationalRuntime{}
	if runtime.SupportsRestart() {
		t.Fatalf("expected false when RestartBackend is nil")
	}

	runtime.RestartBackend = func() {}
	if !runtime.SupportsRestart() {
		t.Fatalf("expected true when RestartBackend is present")
	}
}
```

- [ ] **Step 2: Run the common package tests to verify they fail**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/provider/cases/common -run 'TestOperationalRuntimeSupportsDisruptionLifecycle|TestOperationalRuntimeSupportsRestart' -count=1
```

Expected: FAIL with `undefined: OperationalRuntime`

- [ ] **Step 3: Write the shared operational scenario builders**

Create `e2e/suites/provider/cases/common/operational_v2.go` with a small runtime contract and reusable cases for namespaced-provider, cluster-provider, and push-secret operational behavior.

```go
package common

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

type OperationalRuntime struct {
	ProviderRef          esv1.SecretStoreRef
	ClusterProviderName  string
	BackendAddress       string
	MakeUnavailable      func()
	RestoreAvailability  func()
	RestartBackend       func()
}

func (r *OperationalRuntime) SupportsDisruptionLifecycle() bool {
	return r != nil && r.MakeUnavailable != nil && r.RestoreAvailability != nil
}

func (r *OperationalRuntime) SupportsRestart() bool {
	return r != nil && r.RestartBackend != nil
}

type OperationalExternalSecretHarness struct {
	PrepareNamespaced func(tc *framework.TestCase) *OperationalRuntime
	PrepareCluster    func(tc *framework.TestCase, cfg ClusterProviderConfig) *OperationalRuntime
}

type OperationalPushSecretHarness struct {
	PrepareNamespaced func(tc *framework.TestCase) *OperationalRuntime
	PrepareCluster    func(tc *framework.TestCase, cfg ClusterProviderConfig) *OperationalRuntime
}

func NamespacedProviderUnavailable(f *framework.Framework, harness OperationalExternalSecretHarness, remoteKey, expectedValue string) (string, func(*framework.TestCase)) {
	return "[common] should surface Provider unavailability and recover after backend restoration", func(tc *framework.TestCase) {
		tc.ExternalSecret.ObjectMeta.Name = "operational-unavailable-es"
		tc.ExternalSecret.Spec.Target.Name = "operational-unavailable-target"
		tc.Secrets = map[string]framework.SecretEntry{
			remoteKey: {Value: jsonSecretValue(expectedValue)},
		}
		var runtime *OperationalRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.PrepareNamespaced(tc)
			tc.ProviderOverride = nil
			tc.ExternalSecret.Spec.SecretStoreRef = runtime.ProviderRef
			runtime.MakeUnavailable()
		}
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			frameworkv2.WaitForProviderConnectionNotReady(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Spec.SecretStoreRef.Name, time.Minute)
			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionFalse)
			runtime.RestoreAvailability()
			frameworkv2.WaitForProviderConnectionReady(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Spec.SecretStoreRef.Name, time.Minute)
		}
	}
}
```

In the same file, add matching builders for:

- `NamespacedProviderRestart`
- `ClusterProviderUnavailable`
- `ClusterProviderRestart`
- `NamespacedPushSecretUnavailable`
- `ClusterProviderPushUnavailable`

Use the existing helper style from `clusterprovider.go` and `push_secret.go`: set the resource spec in `tc.Prepare`, use `waitForExternalSecretStatus` or `waitForPushSecretStatus`, and call the shared v2 readiness helpers for the resource-level assertions.

- [ ] **Step 4: Run the common package tests again to verify they pass**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/provider/cases/common -run 'TestOperationalRuntimeSupportsDisruptionLifecycle|TestOperationalRuntimeSupportsRestart|TestClusterProviderExternalSecretRuntimeSupportsAuthLifecycle|TestClusterProviderPushRuntimeSupportsAuthLifecycle' -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/provider/cases/common/operational_v2.go e2e/suites/provider/cases/common/operational_v2_test.go
git commit -m "Add shared v2 operational scenario builders"
```

### Task 3: Add Fake Provider Operational Coverage

**Files:**
- Create: `e2e/suites/provider/cases/fake/operational_v2.go`
- Modify: `e2e/suites/provider/cases/fake/provider_v2.go`
- Modify: `e2e/suites/provider/cases/fake/provider_v2_test.go`

- [ ] **Step 1: Write the failing fake-provider helper test**

Extend `e2e/suites/provider/cases/fake/provider_v2_test.go` with a small pure test for the backend selector helper you will add.

```go
func TestFakeBackendTargetUsesProviderNamespaceAndSelector(t *testing.T) {
	target := fakeBackendTarget()
	if target.Namespace != frameworkv2.ProviderNamespace {
		t.Fatalf("expected provider namespace %q, got %q", frameworkv2.ProviderNamespace, target.Namespace)
	}
	if target.PodLabelSelector != "app.kubernetes.io/name=external-secrets-provider-fake" {
		t.Fatalf("unexpected selector %q", target.PodLabelSelector)
	}
}
```

- [ ] **Step 2: Run the fake package tests to verify they fail**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/provider/cases/fake -run 'TestFakeBackendTargetUsesProviderNamespaceAndSelector|TestUpsertFakeProviderDataReplacesMatchingEntry' -count=1
```

Expected: FAIL with `undefined: fakeBackendTarget`

- [ ] **Step 3: Implement fake operational harnesses and specs**

Modify `e2e/suites/provider/cases/fake/provider_v2.go` to expose a backend target helper and operational runtime factories.

```go
func fakeBackendTarget() frameworkv2.BackendTarget {
	return frameworkv2.BackendTarget{
		Namespace:        frameworkv2.ProviderNamespace,
		PodLabelSelector: "app.kubernetes.io/name=external-secrets-provider-fake",
	}
}

func (s *ProviderV2) prepareNamespacedOperationalRuntime() *common.OperationalRuntime {
	return &common.OperationalRuntime{
		ProviderRef: esv1.SecretStoreRef{
			Name: s.framework.Namespace.Name,
			Kind: esv1.ProviderKindStr,
		},
		BackendAddress: frameworkv2.ProviderAddress("fake"),
		MakeUnavailable: func() {
			frameworkv2.ScaleDeploymentBySelector(s.framework, fakeBackendTarget(), 0)
		},
		RestoreAvailability: func() {
			frameworkv2.ScaleDeploymentBySelector(s.framework, fakeBackendTarget(), 1)
		},
		RestartBackend: func() {
			frameworkv2.DeleteOneProviderPodBySelector(s.framework, fakeBackendTarget())
		},
	}
}
```

Create `e2e/suites/provider/cases/fake/operational_v2.go` with operational labels and shared common entries.

```go
package fake

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[fake] v2 operational", Label("fake", "v2", "operational"), func() {
	f := framework.New("eso-fake-v2-operational")
	prov := NewProviderV2(f)

	DescribeTable("external secret operational behavior",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.NamespacedProviderUnavailable(f, newFakeOperationalExternalSecretHarness(f, prov), "fake-operational-unavailable", "recovered")),
		Entry(common.NamespacedProviderRestart(f, newFakeOperationalExternalSecretHarness(f, prov), "fake-operational-restart", "restarted")),
		Entry(common.ClusterProviderUnavailable(f, newFakeOperationalExternalSecretHarness(f, prov), "fake-operational-cluster", "cluster-recovered", esv1.AuthenticationScopeManifestNamespace)),
	)

	DescribeTable("push secret operational behavior",
		framework.TableFuncWithPushSecret(f, prov, nil),
		Entry(common.NamespacedPushSecretUnavailable(f, newFakeOperationalPushHarness(f, prov))),
		Entry(common.ClusterProviderPushUnavailable(f, newFakeOperationalPushHarness(f, prov), esv1.AuthenticationScopeManifestNamespace)),
	)
})
```

- [ ] **Step 4: Run fake unit tests and focused fake operational e2e**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/provider/cases/fake -run 'TestFakeBackendTargetUsesProviderNamespaceAndSelector|TestUpsertFakeProviderDataReplacesMatchingEntry|TestRemoveFakeProviderDataRemovesOnlyExactMatch' -count=1
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 V2_GINKGO_LABELS='fake && v2 && operational' TEST_SUITES='provider'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/provider/cases/fake/provider_v2.go e2e/suites/provider/cases/fake/provider_v2_test.go e2e/suites/provider/cases/fake/operational_v2.go
git commit -m "Add fake v2 operational e2e coverage"
```

### Task 4: Add Kubernetes Provider Operational Coverage

**Files:**
- Create: `e2e/suites/provider/cases/kubernetes/operational_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/provider_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/clusterprovider_v2.go`

- [ ] **Step 1: Write the failing Kubernetes helper test**

Add `TestKubernetesBackendTargetUsesProviderNamespaceAndSelector` to `e2e/framework/v2/helpers_test.go` or create `e2e/suites/provider/cases/kubernetes/operational_v2_test.go`.

```go
func TestKubernetesBackendTargetUsesProviderNamespaceAndSelector(t *testing.T) {
	target := kubernetesBackendTarget()
	if target.Namespace != frameworkv2.ProviderNamespace {
		t.Fatalf("expected provider namespace %q, got %q", frameworkv2.ProviderNamespace, target.Namespace)
	}
	if target.PodLabelSelector != "app.kubernetes.io/name=external-secrets-provider-kubernetes" {
		t.Fatalf("unexpected selector %q", target.PodLabelSelector)
	}
}
```

- [ ] **Step 2: Run the Kubernetes package tests to verify they fail**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/provider/cases/kubernetes -run 'TestKubernetesBackendTargetUsesProviderNamespaceAndSelector' -count=1
```

Expected: FAIL with `undefined: kubernetesBackendTarget`

- [ ] **Step 3: Implement the Kubernetes operational harness and specs**

Create `e2e/suites/provider/cases/kubernetes/operational_v2.go` and expose backend helpers from the provider files.

```go
func kubernetesBackendTarget() frameworkv2.BackendTarget {
	return frameworkv2.BackendTarget{
		Namespace:        frameworkv2.ProviderNamespace,
		PodLabelSelector: "app.kubernetes.io/name=external-secrets-provider-kubernetes",
	}
}

func newKubernetesOperationalExternalSecretHarness(f *framework.Framework) common.OperationalExternalSecretHarness {
	return common.OperationalExternalSecretHarness{
		PrepareNamespaced: func(tc *framework.TestCase) *common.OperationalRuntime {
			return &common.OperationalRuntime{
				ProviderRef: esv1.SecretStoreRef{Name: f.Namespace.Name, Kind: esv1.ProviderKindStr},
				BackendAddress: frameworkv2.ProviderAddress("kubernetes"),
				MakeUnavailable: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 0)
				},
				RestoreAvailability: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 1)
				},
				RestartBackend: func() {
					frameworkv2.DeleteOneProviderPodBySelector(f, kubernetesBackendTarget())
				},
			}
		},
	}
}
```

Populate the spec file with deterministic namespaced and cluster-provider entries, reusing the existing remote-secret setup in `provider_v2.go` and `clusterprovider_v2.go`.

- [ ] **Step 4: Run Kubernetes unit tests and focused operational e2e**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/provider/cases/kubernetes -run 'TestKubernetesBackendTargetUsesProviderNamespaceAndSelector|TestCreateKubernetesProviderUsesProvidedCABundle' -count=1
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 V2_GINKGO_LABELS='kubernetes && v2 && operational' TEST_SUITES='provider'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/provider/cases/kubernetes/operational_v2.go e2e/suites/provider/cases/kubernetes/provider_v2.go e2e/suites/provider/cases/kubernetes/clusterprovider_v2.go
git commit -m "Add kubernetes v2 operational e2e coverage"
```

### Task 5: Add Fake V2 Generator Operational Coverage

**Files:**
- Create: `e2e/suites/generator/operational_v2.go`
- Modify: `e2e/suites/generator/testcase.go`

- [ ] **Step 1: Write the failing generator status helper test**

Create a small test file `e2e/suites/generator/operational_v2_test.go` or extend an existing generator test file with a helper-level test.

```go
func TestGetESCondReturnsNilWhenConditionMissing(t *testing.T) {
	got := getESCond(esv1.ExternalSecretStatus{}, esv1.ExternalSecretReady)
	if got != nil {
		t.Fatalf("expected nil condition, got %#v", got)
	}
}
```

This keeps the generator task grounded in the existing helper file before the operational flow adds negative-path waits.

- [ ] **Step 2: Run the focused generator tests**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/generator -run 'TestGetESCondReturnsNilWhenConditionMissing|TestAWSGeneratorAuthSessionTokenHandling' -count=1
```

Expected: PASS before the helper change, confirming the package is healthy before adding operational behavior

- [ ] **Step 3: Implement a fake generator outage/recovery spec**

Modify `e2e/suites/generator/testcase.go` to add a negative-path wait helper that can observe `ExternalSecretReady=False` before recovery.

```go
func waitForGeneratorExternalSecretStatus(f *framework.Framework, namespace, name string, expected v1.ConditionStatus) {
	Eventually(func() v1.ConditionStatus {
		var es esv1.ExternalSecret
		if err := f.CRClient.Get(GinkgoT().Context(), types.NamespacedName{Namespace: namespace, Name: name}, &es); err != nil {
			return ""
		}
		cond := getESCond(es.Status, esv1.ExternalSecretReady)
		if cond == nil {
			return ""
		}
		return cond.Status
	}).WithTimeout(30 * time.Second).Should(Equal(expected))
}
```

Create `e2e/suites/generator/operational_v2.go` with a fake-generator operational case.

```go
package generator

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("fake generator operational v2", Label("fake", "v2", "operational", "generator"), func() {
	f := framework.New("fake-generator-operational")

	It("recovers after the fake provider pod is restarted", func() {
		tc := &testCase{Framework: f}
		tc.Generator = &genv1alpha1.Fake{
			TypeMeta: metav1.TypeMeta{APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version, Kind: genv1alpha1.FakeKind},
			ObjectMeta: metav1.ObjectMeta{Name: generatorName, Namespace: f.Namespace.Name},
			Spec: genv1alpha1.FakeSpec{Data: map[string]string{"value": "recovered"}},
		}
		tc.ExternalSecret = &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{Name: "fake-generator-operational-es", Namespace: f.Namespace.Name},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				Target: esv1.ExternalSecretTarget{Name: "fake-generator-operational-target"},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{{
					SourceRef: &esv1.StoreGeneratorSourceRef{
						GeneratorRef: &esv1.GeneratorRef{Kind: "Fake", Name: generatorName},
					},
				}},
			},
		}

		Expect(f.CRClient.Create(GinkgoT().Context(), tc.Generator)).To(Succeed())
		Expect(f.CRClient.Create(GinkgoT().Context(), tc.ExternalSecret)).To(Succeed())
		frameworkv2.DeleteOneProviderPodBySelector(f, frameworkv2.BackendTarget{
			Namespace: frameworkv2.ProviderNamespace,
			PodLabelSelector: "app.kubernetes.io/name=external-secrets-provider-fake",
		})
		waitForGeneratorExternalSecretStatus(f, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, v1.ConditionFalse)
		waitForGeneratorExternalSecretStatus(f, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, v1.ConditionTrue)
	})
})
```

- [ ] **Step 4: Run the generator tests and focused fake operational generator e2e**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./suites/generator -run 'TestGetESCondReturnsNilWhenConditionMissing|TestAWSGeneratorAuthSessionTokenHandling|TestCreateAWSGeneratorCredentialsSecretUpdatesData' -count=1
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 V2_GINKGO_LABELS='fake && v2 && operational && generator' TEST_SUITES='generator'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/generator/testcase.go e2e/suites/generator/operational_v2.go
git commit -m "Add fake v2 generator operational e2e coverage"
```

### Task 6: Add Connection-Reuse And Provider-Fanout Metrics Scenarios

**Files:**
- Modify: `e2e/suites/provider/cases/fake/operational_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/metrics_v2.go`
- Modify: `e2e/framework/v2/metrics.go`

- [ ] **Step 1: Write the failing metrics-scaling expectation as a spec addition**

Add new operational entries that create many `ExternalSecret` resources against one backend and assert pooled connections stay bounded. Use the fake provider for the strictest deterministic check first.

```go
It("reuses one backend connection across many namespaced fake Provider consumers", func() {
	const consumerCount = 10
	// create ten ExternalSecrets using the same Provider backend
	// scrape controller metrics
	// assert grpc_pool_connections_total for address=provider-fake stays <= 2
})

It("does not create one pooled connection per ExternalSecret", func() {
	// assert the total connection count is less than consumerCount
})
```

- [ ] **Step 2: Run a focused fake operational label selection to verify the new scenarios are absent**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 V2_GINKGO_LABELS='fake && v2 && operational' TEST_SUITES='provider'
```

Expected: PASS before the scaling cases are added, with no metrics-scaling assertions yet

- [ ] **Step 3: Implement bounded-scaling and fanout assertions**

Extend `e2e/suites/provider/cases/fake/operational_v2.go` with deterministic scaling specs.

```go
metrics, err := frameworkv2.ScrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace)
Expect(err).ToNot(HaveOccurred())

total := frameworkv2.SumMetricValues(metrics, "grpc_pool_connections_total", map[string]string{
	"address": frameworkv2.ProviderAddress("fake"),
})
Expect(total).To(BeNumerically("<=", 2), "expected bounded connection reuse for one backend")
Expect(total).To(BeNumerically("<", consumerCount))
```

For provider-fanout, create multiple `Provider` CRs that all point at the same fake backend address and assert total connections remain bounded by backend identity rather than consumer count.

Also extend `e2e/suites/provider/cases/kubernetes/metrics_v2.go` with one reuse assertion for the Kubernetes provider using the same helper functions, but keep the bound loose enough to survive restarts.

- [ ] **Step 4: Run focused fake and kubernetes metrics checks**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 V2_GINKGO_LABELS='(fake || kubernetes) && v2 && operational' TEST_SUITES='provider'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/provider/cases/fake/operational_v2.go e2e/suites/provider/cases/kubernetes/metrics_v2.go e2e/framework/v2/metrics.go
git commit -m "Add v2 connection reuse and fanout e2e checks"
```

### Task 7: Wire A Focused Operational V2 Target Into Make And CI

**Files:**
- Modify: `e2e/Makefile`
- Modify: `e2e/makefile_test.go`
- Modify: `Makefile`
- Modify: `.github/actions/e2e/action.yml`
- Modify: `.github/workflows/e2e.yml`

- [ ] **Step 1: Write the failing make-target tests**

Extend `e2e/makefile_test.go` with explicit coverage for a new focused target.

```go
func TestV2OperationalMakeTarget(t *testing.T) {
	cmd := renderMakeDryRun(t, "test.v2.operational")
	if !strings.Contains(cmd, `V2_GINKGO_LABELS="v2 && operational && fake"`) {
		t.Fatalf("expected operational labels in target, got:\n%s", cmd)
	}
	if !strings.Contains(cmd, `TEST_SUITES="provider generator"`) {
		t.Fatalf("expected provider and generator suites in target, got:\n%s", cmd)
	}
}
```

- [ ] **Step 2: Run the makefile tests to verify they fail**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test . -run 'TestV2OperationalMakeTarget|TestV2MakeTarget|TestClassicMakeTarget' -count=1
```

Expected: FAIL with `test.v2.operational` missing

- [ ] **Step 3: Implement the focused operational targets and CI wiring**

Modify `e2e/Makefile`:

```make
test.v2.operational: ## Run focused operational v2 e2e tests
	$(MAKE) test.v2 V2_GINKGO_LABELS='v2 && operational && fake' TEST_SUITES='provider generator'
```

Modify the root `Makefile`:

```make
.PHONY: test.e2e.v2.operational
test.e2e.v2.operational: generate ## Run focused V2 operational E2E tests
	@$(INFO) go test v2 operational e2e-tests
	$(MAKE) -C ./e2e test.v2.operational
	@$(OK) go test v2 operational e2e-tests
```

Modify `.github/actions/e2e/action.yml` to allow the new target:

```bash
case "$MAKE_TARGET" in
  test.e2e|test.e2e.v2|test.e2e.v2.operational)
    make "$MAKE_TARGET"
    ;;
```

Modify `.github/workflows/e2e.yml` to add a focused operational suite entry that runs a real deterministic test in CI.

```yaml
        - name: v2-operational
          make_target: test.e2e.v2.operational
          allow_failure: true
```

Keep `allow_failure: true` for the first rollout so the real CI run happens without immediately turning the whole workflow red while the new slice stabilizes.

- [ ] **Step 4: Run the makefile tests and a local focused operational target**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test . -run 'TestV2OperationalMakeTarget|TestV2MakeTarget|TestClassicMakeTarget|TestV2MakeTargetPrunesDockerImagesInCI' -count=1
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2.operational
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add e2e/Makefile e2e/makefile_test.go Makefile .github/actions/e2e/action.yml .github/workflows/e2e.yml
git commit -m "Wire focused v2 operational e2e into CI"
```

### Task 8: Final Verification And CI Observation

**Files:**
- Modify as needed: any files from Tasks 1-7 based on verification failures

- [ ] **Step 1: Run the local verification matrix**

Run:

```bash
cd e2e
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test ./framework/v2 ./suites/provider/cases/common ./suites/provider/cases/fake ./suites/provider/cases/kubernetes ./suites/generator -count=1
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 V2_GINKGO_LABELS='(fake || kubernetes) && v2 && operational' TEST_SUITES='provider'
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off make test.v2 V2_GINKGO_LABELS='fake && v2 && operational && generator' TEST_SUITES='generator'
TMPDIR=/home/moritz/.cache/eso-tmp GOTMPDIR=/home/moritz/.cache/eso-tmp GOCACHE=/home/moritz/.cache/eso-go-build GOMODCACHE=/home/moritz/.cache/eso-go-mod GOWORK=off go test . -run 'TestV2OperationalMakeTarget|TestV2MakeTarget|TestClassicMakeTarget|TestV2MakeTargetPrunesDockerImagesInCI' -count=1
```

Expected: PASS

- [ ] **Step 2: Push the branch and watch the focused CI slice**

Run:

```bash
git push
```

Then observe the `v2-operational` workflow job and record:

- which deterministic operational scenario actually ran
- whether it passed in CI
- whether any follow-up stabilization changes were required

- [ ] **Step 3: Fix CI issues one at a time if the real operational run fails**

For each CI failure:

1. reproduce locally with the matching label set
2. change only the failing helper or scenario
3. rerun the matching local verification command
4. push and re-observe the CI job

- [ ] **Step 4: Commit any final stabilization fixes**

```bash
git add <exact files changed>
git commit -m "Stabilize v2 operational e2e coverage"
```
