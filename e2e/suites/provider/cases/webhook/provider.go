/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package webhook exercises the webhook provider against an HTTP backend that
// runs inside this test process. The provider's spec.url accepts any address,
// so the backend needs no vendor account and no repo secrets, which keeps the
// leg fork-safe and runnable on every PR.
//
// The backend is deliberately in-process rather than a deployed image: it
// records every request it receives, so a PushSecret spec can assert the exact
// body and headers the controller sent instead of inferring them from a status
// or scraping pod logs.
//
// IMPORTANT: this only works when the suite itself runs inside the cluster,
// which is what e2e/run.sh does (it launches the suite as a pod). Running the
// suite from a workstation against a kind cluster leaves the Service with an
// endpoint the controller cannot reach; setUpBackend fails early and says so.
package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	// backendBasePort is offset by the ginkgo process index. The suite runs with
	// `ginkgo -p`, so several processes share one pod: each needs its own
	// listener port, and gets its own in-memory store with it.
	backendBasePort = 18080

	// serviceName is created per test namespace so it is torn down with the
	// namespace rather than leaking for the life of the suite.
	serviceName = "webhook-e2e-backend"
	portName    = "http"

	// namespaceFile is how a process running in a pod learns its own namespace.
	namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	// The provider reads every Secret through getStoreSecret, which requires
	// this label on all of them, the auth ones included.
	storeTypeLabel = "external-secrets.io/type"
	storeTypeValue = "webhook"

	authSecretName  = "webhook-e2e-auth"
	authSecretKey   = "token"
	authSecretValue = "e2e-secret-token"

	// kvPath serves reads, pushes and deletes.
	kvPath = "/kv/"

	// The read and the push code paths template spec.url from disjoint variable
	// sets: a read exposes .remoteRef.key, while a push and a delete expose
	// .remoteRef.remoteKey. text/template runs with missingkey=default, so
	// naming the absent one renders the literal "<no value>" rather than an
	// empty string. Concatenating both therefore does not work, though a
	// fallback does ("{{ or .remoteRef.key .remoteRef.remoteKey }}"). The specs
	// use one explicit template per path anyway, so it stays obvious which
	// variable set each one exercises.
	readKeyTemplate = "{{ .remoteRef.key }}"
	pushKeyTemplate = "{{ .remoteRef.remoteKey }}"
)

// recordedRequest is what the backend saw on the wire. Specs assert against
// this, which is the reason the backend lives in-process.
type recordedRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   string
}

// backend is the in-process HTTP store. Its zero value is not usable; use
// newBackend.
type backend struct {
	mu       sync.Mutex
	values   map[string]string
	requests []recordedRequest
}

func newBackend() *backend {
	return &backend{values: map[string]string{}}
}

func (b *backend) set(key, value string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.values[key] = value
}

func (b *backend) delete(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.values, key)
}

func (b *backend) value(key string) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.values[key]
	return v, ok
}

// reset clears state between specs. Specs in one process share a backend, so
// without this a key left behind by a failed spec would leak into the next.
func (b *backend) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.values = map[string]string{}
	b.requests = nil
}

func (b *backend) record(r recordedRequest) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.requests = append(b.requests, r)
}

// requestsFor returns every recorded request whose path ends in key, oldest
// first.
//
// Results are not guaranteed to belong only to the current spec. Namespace
// deletion does not block, so a previous spec's ExternalSecret can still be
// reconciling against the same listener for a few seconds. Match on the last
// entry or filter by method; do not assert an exact count.
func (b *backend) requestsFor(key string) []recordedRequest {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []recordedRequest
	for _, r := range b.requests {
		if strings.HasSuffix(r.Path, "/"+key) {
			out = append(out, r)
		}
	}
	return out
}

// handler implements the read, push and delete verbs the provider issues.
//
// A read returns {"value": "<stored>"} and the store sets result.jsonPath to
// $.value. That shape serves both provider entry points: GetSecret returns the
// string as-is, and GetSecretMap re-parses a string result as JSON, so
// dataFrom.extract works when the stored value is a JSON object.
//
// A miss must be 404 and nothing else: the provider maps 404 to NoSecretError,
// which is what makes SecretExists report false instead of erroring, and what
// makes a delete of an absent key succeed.
func (b *backend) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(kvPath, func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, kvPath)
		body, _ := io.ReadAll(r.Body)
		b.record(recordedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Header: r.Header.Clone(),
			Body:   string(body),
		})

		switch r.Method {
		case http.MethodGet:
			value, ok := b.value(key)
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"value": value})
		case http.MethodPost, http.MethodPut:
			b.set(key, string(body))
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if _, ok := b.value(key); !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			b.delete(key)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return mux
}

// addressType picks the EndpointSlice address family from the resolved address,
// so a dual-stack or IPv6-first cluster does not fail API validation.
func addressType(ip string) discoveryv1.AddressType {
	if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() == nil {
		return discoveryv1.AddressTypeIPv6
	}
	return discoveryv1.AddressTypeIPv4
}

// The listener is per process, so it is started once and shared by every spec
// that process runs. The Service is per namespace and so is created per spec.
var (
	setUpOnce   sync.Once
	sharedState *backend
	sharedPort  int
	sharedPodIP string
	sharedSetUp error
)

// Provider wires the in-process backend to the framework's table tests.
type Provider struct {
	framework *framework.Framework
	backend   *backend
	baseURL   string
}

func NewProvider(f *framework.Framework) *Provider {
	prov := &Provider{framework: f}
	// Registered as BeforeEach rather than run here: this constructor executes
	// during tree construction in every parallel process, including ones that
	// will not run a single webhook spec.
	BeforeEach(prov.BeforeEach)
	return prov
}

func (p *Provider) BeforeEach() {
	setUpOnce.Do(setUpBackend)
	Expect(sharedSetUp).ToNot(HaveOccurred())

	p.backend = sharedState
	p.backend.reset()
	p.exposeBackend()
	p.CreateAuthSecret(authSecretName, true)
	p.CreateStore()
}

// setUpBackend binds this process's port and starts serving. Errors are stored
// rather than asserted so the failure surfaces inside a spec.
func setUpBackend() {
	sharedState = newBackend()

	// Order matters: the namespace file is the only reliable in-a-pod signal, so
	// check it before resolving an address. Hostname resolution succeeds off
	// cluster too (to 127.0.1.1 on Debian, to a LAN address on macOS), which
	// would otherwise publish an EndpointSlice the controller cannot use and
	// leave every spec failing with an opaque connection error.
	if err := assertRunningInCluster(); err != nil {
		sharedSetUp = err
		return
	}

	ip, err := podIP()
	if err != nil {
		sharedSetUp = err
		return
	}
	sharedPodIP = ip
	sharedPort = backendBasePort + GinkgoParallelProcess()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", sharedPort))
	if err != nil {
		sharedSetUp = fmt.Errorf("cannot listen on port %d: %w", sharedPort, err)
		return
	}
	server := &http.Server{Handler: sharedState.handler()}
	go func() {
		defer GinkgoRecover()
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// The suite is tearing down; nothing left to assert against.
			_, _ = fmt.Fprintf(GinkgoWriter, "backend stopped: %v\n", err)
		}
	}()
}

// podIP returns the address the controller will connect back to.
func podIP() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("cannot determine hostname: %w", err)
	}
	addrs, err := net.LookupHost(host)
	if err != nil || len(addrs) == 0 {
		return "", fmt.Errorf("cannot resolve own address (%q): %w", host, err)
	}
	return addrs[0], nil
}

// assertRunningInCluster fails when the suite is not executing inside a pod.
// The projected serviceaccount namespace file is the discriminator; hostname
// resolution is not, because it succeeds off cluster and yields an address the
// controller cannot route to.
func assertRunningInCluster() error {
	if _, err := os.Stat(namespaceFile); err != nil {
		return fmt.Errorf(
			"%s is absent, so this is not running in a pod: the webhook suite "+
				"exposes an in-process backend to the controller and must run "+
				"in-cluster, the way e2e/run.sh launches it: %w", namespaceFile, err)
	}
	return nil
}

// exposeBackend publishes this process's listener into the test namespace. The
// Service carries no selector, and the EndpointSlice names the pod address
// explicitly, so it resolves regardless of which namespace the suite pod runs
// in and regardless of the labels run.sh happens to set on it.
func (p *Provider) exposeBackend() {
	ns := p.framework.Namespace.Name
	port := int32(sharedPort)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       portName,
				Port:       port,
				TargetPort: intstr.FromInt32(port),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	Expect(p.framework.CRClient.Create(GinkgoT().Context(), svc)).To(Succeed())

	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ns,
			Labels:    map[string]string{discoveryv1.LabelServiceName: serviceName},
		},
		AddressType: addressType(sharedPodIP),
		Endpoints: []discoveryv1.Endpoint{{
			Addresses:  []string{sharedPodIP},
			Conditions: discoveryv1.EndpointConditions{Ready: new(true)},
		}},
		Ports: []discoveryv1.EndpointPort{{
			Name:     new(portName),
			Port:     new(port),
			Protocol: new(corev1.ProtocolTCP),
		}},
	}
	Expect(p.framework.CRClient.Create(GinkgoT().Context(), slice)).To(Succeed())

	p.baseURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s",
		serviceName, ns, sharedPort, kvPath)
}

// CreateAuthSecret creates the Secret the store references from spec.secrets.
// Pass labelled=false to build the same Secret without the
// external-secrets.io/type label, which is how the negative specs prove the
// provider refuses it.
func (p *Provider) CreateAuthSecret(name string, labelled bool) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.framework.Namespace.Name,
		},
		Data: map[string][]byte{authSecretKey: []byte(authSecretValue)},
	}
	if labelled {
		secret.Labels = map[string]string{storeTypeLabel: storeTypeValue}
	}
	Expect(p.framework.CRClient.Create(GinkgoT().Context(), secret)).To(Succeed())
}

// CreateSecret implements framework.SecretStoreProvider by writing straight
// into the backing map, so the shared table in cases/common applies here.
func (p *Provider) CreateSecret(key string, val framework.SecretEntry) {
	p.backend.set(key, val.Value)
}

func (p *Provider) DeleteSecret(key string) {
	p.backend.delete(key)
}

// CreateStore installs the read-oriented SecretStore the specs sync through.
//
// spec.method is deliberately left unset. It is shared by the read and the push
// path but its default differs per path: GET for a read, POST for a push, with
// a delete always DELETE. Pinning it to GET here would silently turn every
// push into a GET.
func (p *Provider) CreateStore() {
	By("creating a webhook secret store")
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.framework.Namespace.Name,
			Namespace: p.framework.Namespace.Name,
		},
		Spec: p.storeSpec(authSecretName, readKeyTemplate),
	}
	Expect(p.framework.CRClient.Create(GinkgoT().Context(), store)).To(Succeed())
}

// storeSpec builds a webhook store whose url and X-Remote-Key header address
// the remote key through keyTemplate, so callers choose between the read and
// the push variable set. The negative specs reuse it to point the same store at
// a differently-built Secret.
func (p *Provider) storeSpec(secretName, keyTemplate string) esv1.SecretStoreSpec {
	return esv1.SecretStoreSpec{
		Provider: &esv1.SecretStoreProvider{
			Webhook: &esv1.WebhookProvider{
				URL: p.baseURL + keyTemplate,
				Headers: map[string]string{
					// Proves spec.secrets values are addressed as
					// .<name>.<keyInSecret> in a header template.
					"Authorization": "Bearer {{ .creds.token }}",
					"X-Remote-Key":  keyTemplate,
				},
				Secrets: []esv1.WebhookSecret{{
					Name: "creds",
					SecretRef: esmeta.SecretKeySelector{
						Name: secretName,
						Key:  authSecretKey,
					},
				}},
				Result: esv1.WebhookResult{JSONPath: "$.value"},
			},
		},
	}
}
