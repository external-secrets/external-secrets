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

package addon

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// nolint

	. "github.com/onsi/ginkgo/v2"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

const (
	// authenticatorServicePort is the port the authenticator-service binary listens on.
	// This matches the CONJUR_AUTHENTICATOR_SERVICE_URL default of http://localhost:5681
	// and must not collide with the conjur-oss container's own PORT (8080), since both
	// containers share the pod's network namespace.
	authenticatorServicePort = "5681"

	// authenticatorServiceContainerName is the name of the sidecar container running
	// the authenticator-service binary inside the conjur-oss pod.
	authenticatorServiceContainerName = "authenticator-service"

	// SpiffeTrustDomain is the SPIFFE trust domain used for the authn-cert authenticator
	// operating in 'spiffe' host mode.
	SpiffeTrustDomain = "eso-tests.example.org"

	// SpiffeWorkloadID is the SPIFFE ID's workload path, embedded as a URI SAN in the
	// SPIFFE client certificate (spiffe://<trust-domain>/<SpiffeWorkloadID>).
	SpiffeWorkloadID = "vm-spiffe"

	// SpiffeIdentityPath is the Conjur policy path that the authn-cert authenticator
	// prepends to the SPIFFE workload path to derive the full host identity. The resulting
	// host must exist at policy path SpiffeIdentityPath/SpiffeWorkloadID.
	SpiffeIdentityPath = "eso-tests/spiffe"
)

type Conjur struct {
	chart        *HelmChart
	dataKey      string
	Namespace    string
	PodName      string
	ConjurClient *conjurapi.Client
	ConjurURL    string

	AdminApiKey    string
	ConjurServerCA []byte
	ClientCert     []byte
	ClientKey      []byte
	SpiffeCert     []byte
	SpiffeKey      []byte
	portForwarder  *PortForward
}

func NewConjur() *Conjur {
	repo := "conjur-conjur"
	dataKey := generateConjurDataKey()

	rootPem, rootKeyPEM, serverPem, serverKeyPem, err := genCertificates("conjur", "conjur-conjur-conjur-oss")
	if err != nil {
		Fail(err.Error())
	}

	// Generate client certificates for cert-based authentication.
	// The CN "vm-01" matches the host identity in the cert host policy.
	rootBlock, _ := pem.Decode(rootPem)
	if rootBlock == nil {
		Fail("unable to decode root cert PEM")
	}
	rootCert, err := x509.ParseCertificate(rootBlock.Bytes)
	if err != nil {
		Fail(fmt.Sprintf("unable to parse root cert: %v", err))
	}
	rootKeyBlock, _ := pem.Decode(rootKeyPEM)
	if rootKeyBlock == nil {
		Fail("unable to decode root key PEM")
	}
	rootKey, err := x509.ParsePKCS1PrivateKey(rootKeyBlock.Bytes)
	if err != nil {
		Fail(fmt.Sprintf("unable to parse root key: %v", err))
	}
	clientCertPem, clientKeyRSA, err := genClientAuthCert(rootCert, rootKey, "vm-01", "")
	if err != nil {
		Fail(fmt.Sprintf("unable to generate client cert: %v", err))
	}
	clientKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  privatePemType,
		Bytes: x509.MarshalPKCS1PrivateKey(clientKeyRSA),
	})

	// Generate a second client certificate carrying a SPIFFE URI SAN, used by the
	// authn-cert authenticator's 'spiffe' host mode, where the authenticated host's
	// identity is derived from the certificate itself rather than the request path.
	spiffeCertPem, spiffeKeyRSA, err := genClientAuthCert(rootCert, rootKey, "",
		fmt.Sprintf("spiffe://%s/%s", SpiffeTrustDomain, SpiffeWorkloadID))
	if err != nil {
		Fail(fmt.Sprintf("unable to generate spiffe client cert: %v", err))
	}
	spiffeKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  privatePemType,
		Bytes: x509.MarshalPKCS1PrivateKey(spiffeKeyRSA),
	})

	return &Conjur{
		dataKey: dataKey,
		chart: &HelmChart{
			Namespace:   "conjur",
			ReleaseName: "conjur-conjur",
			Chart:       fmt.Sprintf("%s/conjur-oss", repo),
			// Use latest version of Conjur OSS. To pin to a specific version, uncomment the following line.
			// ChartVersion: "2.0.7",
			Repo: ChartRepo{
				Name: repo,
				URL:  "https://cyberark.github.io/helm-charts",
			},
			Values: []string{filepath.Join(AssetDir(), "conjur.values.yaml")},
			Args: []string{
				"--create-namespace",
				"--set", "ssl.caCert=" + base64.StdEncoding.EncodeToString(rootPem),
				"--set", "ssl.caKey=" + base64.StdEncoding.EncodeToString(rootKeyPEM),
				"--set", "ssl.cert=" + base64.StdEncoding.EncodeToString(serverPem),
				"--set", "ssl.key=" + base64.StdEncoding.EncodeToString(serverKeyPem),
			},
			Vars: []StringTuple{
				{
					Key:   "dataKey",
					Value: dataKey,
				},
			},
		},
		Namespace:  "conjur",
		ClientCert: clientCertPem,
		ClientKey:  clientKeyPem,
		SpiffeCert: spiffeCertPem,
		SpiffeKey:  spiffeKeyPem,
	}
}

// Install sets up the conjur-oss chart, the authenticator-service sidecar, and configures
// the authenticators. The install-then-patch-then-exec sequence spans several kind/kubelet
// eventually-consistent steps (Deployment rollout, pod scheduling, kubelet exec/attach
// readiness); under load these can intermittently race even with generous waits at each
// step (e.g. a pod reported Ready via the API can still be replaced by a subsequent
// reconcile before it is actually exec-attachable). Rather than chase every individual
// timing window, retry the whole sequence a few times, tearing down between attempts.
func (l *Conjur) Install() error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			By(fmt.Sprintf("retrying conjur install (attempt %d/%d) after: %v", attempt, maxAttempts, lastErr))
			if err := l.teardown(); err != nil {
				return fmt.Errorf("unable to tear down conjur before retry: %w", err)
			}
		}

		if lastErr = l.installOnce(); lastErr == nil {
			return nil
		}
	}
	return fmt.Errorf("conjur install failed after %d attempts: %w", maxAttempts, lastErr)
}

func (l *Conjur) installOnce() error {
	if err := l.chart.Install(); err != nil {
		return err
	}

	if err := l.patchAuthenticatorServiceSidecar(); err != nil {
		return err
	}

	if err := l.initConjur(); err != nil {
		return err
	}

	return l.configureConjur()
}

// teardown removes the conjur-oss release and namespace so installOnce can run again from
// a clean slate. Errors are ignored: the namespace may not exist yet if chart.Install()
// itself failed on a previous attempt.
func (l *Conjur) teardown() error {
	if l.portForwarder != nil {
		l.portForwarder.Close()
		l.portForwarder = nil
	}
	_ = l.chart.Uninstall()
	err := l.chart.config.KubeClientSet.CoreV1().Namespaces().Delete(GinkgoT().Context(), l.chart.Namespace, metav1.DeleteOptions{})
	if err != nil {
		return nil //nolint:nilerr // namespace may already be gone; nothing to clean up.
	}
	return util.WaitForKubeNamespaceNotExist(l.chart.Namespace, l.chart.config.KubeClientSet)
}

// retryExecCmd retries util.ExecCmdWithContainer briefly to absorb the window where a
// freshly-created pod reports Ready via the API before the kubelet's exec/attach endpoint
// for its containers is actually available.
func retryExecCmd(client kubernetes.Interface, config *rest.Config, podName, containerName, namespace, command string) error {
	var lastErr error
	for i := 0; i < 10; i++ {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
		if _, lastErr = util.ExecCmdWithContainer(client, config, podName, containerName, namespace, command); lastErr == nil {
			return nil
		}
	}
	return lastErr
}

// waitForConjurServiceReady polls the Conjur Service URL from inside the cluster (via exec
// into the conjur-oss pod) until it gets a real HTTP response, absorbing kube-proxy
// Endpoints-sync lag after the sidecar rollout. curl's own TLS verification is disabled
// (-k) because this only checks that the Service routes to a live, responsive nginx; actual
// certificate validation is exercised by the real authentication flows later.
func (l *Conjur) waitForConjurServiceReady() error {
	cmd := fmt.Sprintf("curl -sk -o /dev/null -w '%%{http_code}' %s/status", l.ConjurURL)
	return wait.PollUntilContextTimeout(GinkgoT().Context(), 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		out, err := util.ExecCmdWithContainer(l.chart.config.KubeClientSet, l.chart.config.KubeConfig,
			l.PodName, "conjur-oss", l.Namespace, cmd)
		if err != nil {
			return false, nil
		}
		return strings.Contains(out, "200"), nil
	})
}

func (l *Conjur) initConjur() error {
	By("Waiting for conjur pods to be running")
	pl, err := util.WaitForPodsRunning(l.chart.config.KubeClientSet, 1, l.Namespace, metav1.ListOptions{
		LabelSelector: "app=conjur-oss",
	})
	if err != nil {
		return fmt.Errorf("error waiting for conjur to be running: %w", err)
	}
	l.PodName = pl.Items[0].Name

	By("Initializing conjur")
	// Get the auto generated certificates from the K8s secrets
	caCertSecret, err := util.GetKubeSecret(l.chart.config.KubeClientSet, l.Namespace, fmt.Sprintf("%s-conjur-ssl-ca-cert", l.chart.ReleaseName))
	if err != nil {
		return fmt.Errorf("error getting conjur ca cert: %w", err)
	}
	l.ConjurServerCA = caCertSecret.Data["tls.crt"]

	// Create "default" account. A freshly-created pod can report Ready in the API before
	// the kubelet's exec/attach endpoint for its containers is actually wired up, so the
	// first exec attempt against it can fail with "container not found" even though the
	// container is running; retry briefly to absorb that race.
	if err := retryExecCmd(l.chart.config.KubeClientSet, l.chart.config.KubeConfig,
		l.PodName, "conjur-oss", l.Namespace, "conjurctl account create default"); err != nil {
		return fmt.Errorf("error initializing conjur: %w", err)
	}

	// Retrieve the admin API key
	apiKey, err := util.ExecCmdWithContainer(
		l.chart.config.KubeClientSet,
		l.chart.config.KubeConfig,
		l.PodName, "conjur-oss", l.Namespace, "conjurctl role retrieve-key default:user:admin")
	if err != nil {
		return fmt.Errorf("error fetching admin API key: %w", err)
	}

	// Note: ExecCmdWithContainer includes the StdErr output with a warning about config directory.
	// Therefore we need to split the output and only use the first line.
	l.AdminApiKey = strings.Split(apiKey, "\n")[0]

	// This e2e test provider uses a local port-forwarded to talk to the vault API instead
	// of using the kubernetes service. This allows us to run the e2e test suite locally.
	l.portForwarder, err = NewPortForward(l.chart.config.KubeClientSet, l.chart.config.KubeConfig,
		"conjur-conjur-conjur-oss", l.chart.Namespace, 9443)
	if err != nil {
		return err
	}
	if err := l.portForwarder.Start(); err != nil {
		return err
	}

	l.ConjurURL = fmt.Sprintf("https://conjur-conjur-conjur-oss.%s.svc.cluster.local", l.Namespace)

	// The ExternalSecret controller reaches Conjur through this Service URL, which routes
	// through kube-proxy to whichever pod is currently in the Service's Endpoints. That
	// Endpoints list is reconciled asynchronously and can lag behind the pod-level readiness
	// checks in patchAuthenticatorServiceSidecar's rollout wait, especially right after the
	// sidecar-triggered rolling update. Probe through the Service URL itself (from inside the
	// cluster, exercising the same DNS/ClusterIP/kube-proxy path the controller will use)
	// before proceeding, so tests don't race a stale or not-yet-synced endpoint.
	if err := l.waitForConjurServiceReady(); err != nil {
		return fmt.Errorf("error waiting for conjur service to be reachable: %w", err)
	}

	cfg := conjurapi.Config{
		Account:      "default",
		ApplianceURL: fmt.Sprintf("https://localhost:%d", l.portForwarder.localPort),
		SSLCert:      string(l.ConjurServerCA),
	}

	l.ConjurClient, err = conjurapi.NewClientFromKey(cfg, authn.LoginPair{
		Login:  "admin",
		APIKey: l.AdminApiKey,
	})
	if err != nil {
		return fmt.Errorf("unable to create conjur client: %w", err)
	}

	return nil
}

// conjurReleaseVersion is the pinned cyberark/conjur GitHub release used to fetch the
// authenticator-service binary. Bump this when a newer release is needed.
const conjurReleaseVersion = "v1.27.0"

// authenticatorServiceInitScript downloads the authenticator-service binary matching the
// node's architecture from the pinned cyberark/conjur GitHub release (see
// conjurReleaseVersion), and writes its minimal config file, into a shared emptyDir volume
// mounted by the sidecar container.
//
// The authenticator-service is not distributed as a container image, only as raw
// per-architecture binaries attached to GitHub releases (see AUTHENTICATOR_SERVICE.md in
// the cyberark/conjur repo), so it must be fetched at runtime rather than referenced by tag.
const authenticatorServiceInitScript = `set -e
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) BIN_ARCH=amd64 ;;
  aarch64|arm64) BIN_ARCH=arm64 ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac
DOWNLOAD_URL=$(curl -s https://api.github.com/repos/cyberark/conjur/releases/tags/` + conjurReleaseVersion + `\
  | grep -o "\"browser_download_url\": *\"[^\"]*authenticator_linux_[0-9]*_${BIN_ARCH}\"" \
  | head -1 | sed -E 's/.*"(https:[^"]+)".*/\1/')
if [ -z "$DOWNLOAD_URL" ]; then
  echo "unable to determine authenticator-service download URL for arch ${BIN_ARCH}" >&2
  exit 1
fi
echo "downloading authenticator-service from ${DOWNLOAD_URL}"
curl -sL -o /authsvc/authenticator-service "$DOWNLOAD_URL"
chmod +x /authsvc/authenticator-service
printf '{"port": "%s", "http_timeout": "10s"}' "` + authenticatorServicePort + `" > /authsvc/config.json`

// patchAuthenticatorServiceSidecar adds an authenticator-service sidecar to the conjur-oss
// deployment and enables the feature flag that lets Conjur delegate authn-cert credential
// validation to it. Conjur OSS's authn-cert authenticator does not validate client
// certificates itself; it calls out to this separate HTTP service (see
// AUTHENTICATOR_SERVICE.md and CERTIFICATE_AUTH.md in the cyberark/conjur repo). The
// conjur-oss Helm chart has no hook for adding sidecars, so the running Deployment is
// patched directly after the initial chart install.
func (l *Conjur) patchAuthenticatorServiceSidecar() error {
	By("patching conjur-oss deployment with authenticator-service sidecar")
	clientSet := l.chart.config.KubeClientSet
	deploymentName := fmt.Sprintf("%s-conjur-oss", l.chart.ReleaseName)

	deployment, err := clientSet.AppsV1().Deployments(l.Namespace).Get(GinkgoT().Context(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get conjur-oss deployment: %w", err)
	}

	binVolume := corev1.Volume{
		Name:         "authsvc-bin",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	binVolumeMount := corev1.VolumeMount{
		Name:      binVolume.Name,
		MountPath: "/authsvc",
	}

	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, binVolume)
	deployment.Spec.Template.Spec.InitContainers = append(deployment.Spec.Template.Spec.InitContainers, corev1.Container{
		Name:         "authenticator-service-fetch",
		Image:        "curlimages/curl:8.11.0",
		Command:      []string{"/bin/sh", "-c"},
		Args:         []string{authenticatorServiceInitScript},
		VolumeMounts: []corev1.VolumeMount{binVolumeMount},
	})
	deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, corev1.Container{
		Name:         authenticatorServiceContainerName,
		Image:        "gcr.io/distroless/static-debian12:latest",
		Command:      []string{"/authsvc/authenticator-service"},
		Args:         []string{"-c", "/authsvc/config.json"},
		VolumeMounts: []corev1.VolumeMount{binVolumeMount},
		Ports: []corev1.ContainerPort{
			{Name: "authsvc", ContainerPort: 5681},
		},
	})

	for i := range deployment.Spec.Template.Spec.Containers {
		c := &deployment.Spec.Template.Spec.Containers[i]
		if c.Name != "conjur-oss" {
			continue
		}
		c.Env = append(c.Env,
			corev1.EnvVar{Name: "CONJUR_FEATURE_AUTHENTICATOR_SERVICE_ENABLED", Value: "true"},
			corev1.EnvVar{Name: "CONJUR_AUTHENTICATOR_SERVICE_URL", Value: "http://localhost:" + authenticatorServicePort},
		)
	}

	updated, err := clientSet.AppsV1().Deployments(l.Namespace).Update(GinkgoT().Context(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch conjur-oss deployment with authenticator-service sidecar: %w", err)
	}

	// Wait for the rollout to fully complete (old ReplicaSet scaled to 0) rather than just
	// waiting for any pod matching the label to be ready. During the rolling update, the
	// old (pre-sidecar) pod can remain Ready while terminating; a label-only wait can race
	// and hand initConjur a pod whose conjur-oss container is already gone.
	if err := waitForDeploymentRollout(clientSet, l.Namespace, deploymentName, updated.Generation); err != nil {
		return fmt.Errorf("error waiting for conjur-oss rollout after sidecar patch: %w", err)
	}

	return nil
}

func waitForDeploymentRollout(clientSet kubernetes.Interface, namespace, name string, generation int64) error {
	if err := wait.PollUntilContextTimeout(GinkgoT().Context(), 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		dep, err := clientSet.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if dep.Status.ObservedGeneration < generation {
			return false, nil
		}
		replicas := int32(1)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		return dep.Status.UpdatedReplicas == replicas &&
			dep.Status.Replicas == replicas &&
			dep.Status.AvailableReplicas == replicas, nil
	}); err != nil {
		return err
	}

	// Deployment status is computed asynchronously by the deployment controller and can
	// briefly report AvailableReplicas==1 while the old pre-sidecar pod is still
	// Running-but-terminating (Kubelet has not removed it yet). Cross-check pod state
	// directly: require exactly one pod for this Deployment that is not terminating and
	// has all 3 containers (conjur-oss, nginx, authenticator-service) ready, so
	// initConjur never execs into a pod that's mid-teardown.
	return wait.PollUntilContextTimeout(GinkgoT().Context(), 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		pods, err := clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=conjur-oss"})
		if err != nil {
			return false, nil
		}
		live := 0
		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.DeletionTimestamp != nil {
				continue
			}
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}
			if len(pod.Status.ContainerStatuses) != 3 {
				continue
			}
			allReady := true
			for _, cs := range pod.Status.ContainerStatuses {
				allReady = allReady && cs.Ready
			}
			if allReady {
				live++
			}
		}
		return live == 1, nil
	})
}

func (l *Conjur) configureConjur() error {
	By("configuring conjur")
	// Construct Conjur policy for authn-jwt. This uses the token-app-property "sub" to
	// authenticate the host. This means that Conjur will determine which host is authenticating
	// based on the "sub" claim in the JWT token, which is provided by the Kubernetes service account.
	policy := `- !policy
  id: conjur/authn-jwt/eso-tests
  body:
    - !webservice
    - !variable public-keys
    - !variable issuer
    - !variable token-app-property
    - !variable audience`

	_, err := l.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	if err != nil {
		return fmt.Errorf("unable to load authn-jwt policy: %w", err)
	}

	// Construct Conjur policy for authn-jwt-hostid. This does not use the token-app-property variable
	// and instead uses the HostID passed in the authentication URL to determine which host is authenticating.
	// This is not the recommended way to authenticate, but it is needed for certain use cases where the
	// JWT token does not contain the "sub" claim.
	policy = `- !policy
  id: conjur/authn-jwt/eso-tests-hostid
  body:
    - !webservice
    - !variable public-keys
    - !variable issuer
    - !variable audience`

	_, err = l.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	if err != nil {
		return fmt.Errorf("unable to load authn-jwt policy: %w", err)
	}

	// Construct Conjur policy for two authn-cert authenticators. Both delegate client
	// certificate validation to the authenticator-service sidecar (see
	// patchAuthenticatorServiceSidecar). The 'ca-cert' variable name is fixed by the
	// authenticator; it is not a policy-defined path segment.
	//
	// 'eso-tests' operates in 'spiffe' host mode: the authenticated host's identity is
	// derived from the SPIFFE URI SAN in the client certificate (see genSpiffeCert), rather
	// than from a role identifier in the request path. This is the mode exercised when the
	// ExternalSecret store omits Auth.Cert.HostID.
	//
	// 'eso-tests-hostid' operates in the default 'request' host mode, mirroring the
	// authn-jwt/eso-tests-hostid authenticator: the host identity is supplied explicitly via
	// Auth.Cert.HostID. A single authenticator cannot serve both modes because host-mode is
	// a per-authenticator setting, not a per-request one.
	policy = `- !policy
  id: conjur/authn-cert/eso-tests
  body:
    - !webservice
    - !variable ca-cert
    - !variable host-mode
    - !variable trust-domain
    - !variable identity-path

- !policy
  id: conjur/authn-cert/eso-tests-hostid
  body:
    - !webservice
    - !variable ca-cert`

	_, err = l.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	if err != nil {
		return fmt.Errorf("unable to load authn-cert policy: %w", err)
	}

	// Set the CA certificate and spiffe-mode variables for the authn-cert authenticators
	certSecrets := map[string]string{
		"conjur/authn-cert/eso-tests/ca-cert":        string(l.ConjurServerCA),
		"conjur/authn-cert/eso-tests/host-mode":      "spiffe",
		"conjur/authn-cert/eso-tests/trust-domain":   SpiffeTrustDomain,
		"conjur/authn-cert/eso-tests/identity-path":  SpiffeIdentityPath,
		"conjur/authn-cert/eso-tests-hostid/ca-cert": string(l.ConjurServerCA),
	}
	for secretPath, secretValue := range certSecrets {
		if err := l.ConjurClient.AddSecret(secretPath, secretValue); err != nil {
			return fmt.Errorf("unable to add secret %s: %w", secretPath, err)
		}
	}

	// Fetch the jwks info from the k8s cluster
	pubKeysJson, issuer, err := l.fetchJWKSandIssuer()
	if err != nil {
		return fmt.Errorf("unable to fetch jwks and issuer: %w", err)
	}

	// Set the variables for the authn-jwt policies
	secrets := map[string]string{
		"conjur/authn-jwt/eso-tests/audience":           l.ConjurURL,
		"conjur/authn-jwt/eso-tests/issuer":             issuer,
		"conjur/authn-jwt/eso-tests/public-keys":        string(pubKeysJson),
		"conjur/authn-jwt/eso-tests/token-app-property": "sub",
		"conjur/authn-jwt/eso-tests-hostid/audience":    l.ConjurURL,
		"conjur/authn-jwt/eso-tests-hostid/issuer":      issuer,
		"conjur/authn-jwt/eso-tests-hostid/public-keys": string(pubKeysJson),
	}

	for secretPath, secretValue := range secrets {
		err := l.ConjurClient.AddSecret(secretPath, secretValue)
		if err != nil {
			return fmt.Errorf("unable to add secret %s: %w", secretPath, err)
		}
	}

	return nil
}

func (l *Conjur) fetchJWKSandIssuer() (pubKeysJson string, issuer string, err error) {
	kc := l.chart.config.KubeClientSet

	// Fetch the openid-configuration
	res, err := kc.CoreV1().RESTClient().Get().AbsPath("/.well-known/openid-configuration").DoRaw(GinkgoT().Context())
	if err != nil {
		return "", "", fmt.Errorf("unable to fetch openid-configuration: %w", err)
	}
	var openidConfig map[string]any
	json.Unmarshal(res, &openidConfig)
	issuer = openidConfig["issuer"].(string)

	// Fetch the jwks
	jwksJson, err := kc.CoreV1().RESTClient().Get().AbsPath("/openid/v1/jwks").DoRaw(GinkgoT().Context())
	if err != nil {
		return "", "", fmt.Errorf("unable to fetch jwks: %w", err)
	}
	var jwks map[string]any
	json.Unmarshal(jwksJson, &jwks)

	// Create a JSON object with the jwks that can be used by Conjur
	pubKeysObj := map[string]any{
		"type":  "jwks",
		"value": jwks,
	}
	pubKeysJsonObj, err := json.Marshal(pubKeysObj)
	if err != nil {
		return "", "", fmt.Errorf("unable to marshal jwks: %w", err)
	}

	pubKeysJson = string(pubKeysJsonObj)
	return pubKeysJson, issuer, nil
}

// nolint:gocritic
func genCertificates(namespace, serviceName string) ([]byte, []byte, []byte, []byte, error) {
	// gen server ca + certs
	rootCert, rootPem, rootKey, err := genCARoot()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("unable to generate ca cert: %w", err)
	}
	serverPem, serverKey, err := genPeerCert(rootCert, rootKey, "vault", []string{
		"localhost",
		serviceName,
		fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)})
	if err != nil {
		return nil, nil, nil, nil, errors.New("unable to generate vault server cert")
	}
	serverKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  privatePemType,
		Bytes: x509.MarshalPKCS1PrivateKey(serverKey)},
	)

	rootKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  privatePemType,
		Bytes: x509.MarshalPKCS1PrivateKey(rootKey),
	})

	return rootPem, rootKeyPEM, serverPem, serverKeyPem, err
}

// genSpiffeCert generates a client certificate carrying the given SPIFFE ID as a URI SAN,
// signed by signingCert/signingKey. This is used by the authn-cert authenticator's 'spiffe'
// host mode, which derives the authenticated host's identity from the certificate's URI SAN
// rather than from a role identifier in the request path.
// genClientAuthCert generates a client certificate for mutual TLS (authn-cert). cn sets the
// Common Name (used by 'request' host mode's cn restriction annotation); spiffeID, if
// non-empty, is embedded as a URI SAN (used by 'spiffe' host mode).
func genClientAuthCert(signingCert *x509.Certificate, signingKey *rsa.PrivateKey, cn, spiffeID string) ([]byte, *rsa.PrivateKey, error) {
	var uris []*url.URL
	if spiffeID != "" {
		uri, err := url.Parse(spiffeID)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse spiffe id: %w", err)
		}
		uris = []*url.URL{uri}
	}

	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	tpl := x509.Certificate{
		Subject: pkix.Name{
			Country:      []string{"/dev/null"},
			Organization: []string{"External Secrets ACME"},
			CommonName:   cn,
		},
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		// DigitalSignature is required for the client to sign the TLS handshake during
		// mutual TLS; CRLSign (used by the sibling genPeerCert for server certs) does not
		// cover that purpose and nginx's strict (critical) key usage check rejects the
		// handshake outright if it's missing.
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		IsCA:           false,
		MaxPathLenZero: true,
		URIs:           uris,
	}
	_, certPEM, err := genCert(&tpl, signingCert, &pkey.PublicKey, signingKey)
	return certPEM, pkey, err
}

func (l *Conjur) Logs() error {
	return l.chart.Logs()
}

func (l *Conjur) Uninstall() error {
	if l.portForwarder != nil {
		l.portForwarder.Close()
		l.portForwarder = nil
	}
	if err := l.chart.Uninstall(); err != nil {
		return err
	}
	return l.chart.config.KubeClientSet.CoreV1().Namespaces().Delete(GinkgoT().Context(), l.chart.Namespace, metav1.DeleteOptions{})
}

func (l *Conjur) Setup(cfg *Config) error {
	return l.chart.Setup(cfg)
}

func generateConjurDataKey() string {
	// Generate a 32 byte cryptographically secure random string.
	// Normally this is done by running `conjurctl data-key generate`
	// but for test purposes we can generate it programmatically.
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Errorf("unable to generate random string: %w", err))
	}

	// Encode the bytes as a base64 string
	return base64.StdEncoding.EncodeToString(b)
}
