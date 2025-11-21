// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package kubernetes implements Kubernetes cluster targets
package kubernetes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/targets"
)

var mu sync.Mutex

// Provider implements the Kubernetes target provider.
type Provider struct{}

// ScanTarget wraps everything needed by scan/push logic for a Kubernetes cluster.
type ScanTarget struct {
	Name                    string
	Namespace               string
	RestConfig              *rest.Config
	ClusterClient           crclient.Client
	NamespaceInclude        []string
	NamespaceExclude        []string
	Selector                labels.Selector
	IncludeImagePullSecrets bool
	IncludeEnvFrom          bool
	IncludeEnvSecretKeyRefs bool
	IncludeVolumeSecrets    bool
	KubeClient              crclient.Client
}

const (
	errNotImplemented    = "not implemented"
	errPropertyMandatory = "property is mandatory"
)

// NewClient creates a new Kubernetes scan target client.
func (p *Provider) NewClient(
	ctx context.Context,
	mgrClient crclient.Client,
	target crclient.Object,
) (tgtv1alpha1.ScanTarget, error) {
	converted, ok := target.(*tgtv1alpha1.KubernetesCluster)
	if !ok {
		return nil, fmt.Errorf("target %q not found", target.GetObjectKind().GroupVersionKind().Kind)
	}

	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return newClient(ctx, converted, mgrClient, clientset.CoreV1())
}

// SecretStoreProvider implements the Kubernetes secret store provider.
type SecretStoreProvider struct {
}

// Capabilities returns the capabilities of the Kubernetes secret store provider.
func (p *SecretStoreProvider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreWriteOnly
}

// ValidateStore validates the Kubernetes secret store.
func (p *SecretStoreProvider) ValidateStore(_ esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

// NewClient creates a new Kubernetes secrets client.
func (p *SecretStoreProvider) NewClient(ctx context.Context, store esv1.GenericStore, mgrClient crclient.Client, _ string) (esv1.SecretsClient, error) {
	converted, ok := store.(*tgtv1alpha1.KubernetesCluster)
	if !ok {
		return nil, fmt.Errorf("store %q not found", store.GetObjectKind().GroupVersionKind().Kind)
	}

	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return newClient(ctx, converted, mgrClient, clientset.CoreV1())
}

// Lock locks the scan target.
func (s *ScanTarget) Lock() {
	mu.Lock()
}

// Unlock unlocks the scan target.
func (s *ScanTarget) Unlock() {
	mu.Unlock()
}

// ScanForSecrets scans for secrets in the Kubernetes cluster.
func (s *ScanTarget) ScanForSecrets(ctx context.Context, secrets []string, _ int) ([]scanv1alpha1.SecretInStoreRef, error) {
	referencedSecrets, err := s.collectReferencedSecrets(ctx)
	if err != nil {
		return nil, err
	}

	var secretsList corev1.SecretList
	if err := s.ClusterClient.List(ctx, &secretsList, &crclient.ListOptions{}); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	results := make([]scanv1alpha1.SecretInStoreRef, 0, 64)
	for i := range secretsList.Items {
		secret := &secretsList.Items[i]
		if !s.namespaceAllowed(secret.Namespace) {
			continue
		}
		key := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)
		if _, ok := referencedSecrets[key]; !ok {
			// Respect scan filters: only consider secrets actually bound by workloads
			continue
		}

		for dataKey, val := range secret.Data {
			if len(val) == 0 {
				continue
			}

			for _, sec := range secrets {
				isEqual := bytes.Equal(val, []byte(sec))
				if !isEqual {
					continue
				}

				results = append(results, scanv1alpha1.SecretInStoreRef{
					APIVersion: tgtv1alpha1.SchemeGroupVersion.String(),
					Kind:       tgtv1alpha1.KubernetesTargetKind,
					Name:       s.Name,
					RemoteRef: scanv1alpha1.RemoteRef{
						Key:      key, // "<namespace>/<secretName>"
						Property: dataKey,
					},
				})
			}
		}
	}

	return results, nil
}

// ScanForConsumers scans for consumers of a secret in the Kubernetes cluster.
func (s *ScanTarget) ScanForConsumers(ctx context.Context, location scanv1alpha1.SecretInStoreRef, hash string) ([]scanv1alpha1.ConsumerFinding, error) {
	// Parse "<namespace>/<secret>"
	secretNamespace, secretName, err := parseNamespaceName(location.RemoteRef.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid secret location key %q: %w", location.RemoteRef.Key, err)
	}
	if !s.namespaceAllowed(secretNamespace) {
		return nil, nil
	}

	var pods corev1.PodList
	if err := s.ClusterClient.List(ctx, &pods, &crclient.ListOptions{
		Namespace:     secretNamespace,
		LabelSelector: s.SelectorOrEverything(),
	}); err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	// Group matched pods by top-level controller
	type agg struct {
		ref                workloadRef
		latestPodReadyTime metav1.Time
	}
	groups := map[string]*agg{}

	for i := range pods.Items {
		pod := &pods.Items[i]

		// Does this pod bind the secret according to toggles?
		ok := s.podBindsSecret(pod, secretName)
		if !ok {
			continue
		}

		// Find top-level controller (Deployment/STS/DS/CronJob/Job; else Pod)
		ref := s.topControllerRef(ctx, pod)

		key := fmt.Sprintf("%s.%s.%s.%s", strings.ToLower(ref.Group+"."+ref.Kind), ref.Namespace, ref.Name, ref.UID)
		if ref.UID == "" {
			key = fmt.Sprintf("%s.%s.%s", strings.ToLower(ref.Group+"."+ref.Kind), ref.Namespace, ref.Name)
		}

		group, ok := groups[key]
		if !ok {
			group = &agg{
				ref: ref,
			}
			groups[key] = group
		}

		if t := podReadyTime(pod); !t.IsZero() {
			if t.After(group.latestPodReadyTime.Time) {
				group.latestPodReadyTime = metav1.NewTime(t)
			}
		}
	}

	// Build ConsumerFindings (one per workload)
	out := make([]scanv1alpha1.ConsumerFinding, 0, len(groups))
	for _, g := range groups {
		id := stableID(s.Name, g.ref)
		display := fmt.Sprintf("%s/%s (%s)", g.ref.Namespace, g.ref.Name, g.ref.Kind)

		out = append(out, scanv1alpha1.ConsumerFinding{
			Type:        tgtv1alpha1.KubernetesTargetKind,
			ID:          id,
			DisplayName: display,
			Attributes: scanv1alpha1.ConsumerAttrs{
				K8sWorkload: &scanv1alpha1.K8sWorkloadSpec{
					ClusterName:     s.Name,
					Namespace:       g.ref.Namespace,
					WorkloadKind:    g.ref.Kind,
					WorkloadGroup:   g.ref.Group,
					WorkloadVersion: g.ref.Version,
					WorkloadName:    g.ref.Name,
					WorkloadUID:     g.ref.UID,
					Controller:      controllerString(g.ref),
				},
			},
			Location: location,
			ObservedIndex: scanv1alpha1.SecretUpdateRecord{
				Timestamp:  g.latestPodReadyTime,
				SecretHash: hash,
			},
		})
	}

	return out, nil
}

// Build a set of "<namespace>/<secretName>" that are referenced by Pods according to enabled scan toggles.
func (s *ScanTarget) collectReferencedSecrets(ctx context.Context) (map[string]struct{}, error) {
	referencedSecrets := make(map[string]struct{})

	var pods corev1.PodList
	if err := s.ClusterClient.List(ctx, &pods, &crclient.ListOptions{
		LabelSelector: s.SelectorOrEverything(),
	}); err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	add := func(namespace, name string) {
		if !s.namespaceAllowed(namespace) {
			return
		}
		referencedSecrets[fmt.Sprintf("%s/%s", namespace, name)] = struct{}{}
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if !s.namespaceAllowed(pod.Namespace) {
			continue
		}

		// imagePullSecrets on Pod
		if s.IncludeImagePullSecrets {
			for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
				add(pod.Namespace, imagePullSecret.Name)
			}
		}

		// volumes[].secret.secretName
		if s.IncludeVolumeSecrets {
			for _, volume := range pod.Spec.Volumes {
				if volume.Secret != nil {
					add(pod.Namespace, volume.Secret.SecretName)
				}
			}
		}

		// envFrom secretRef
		if s.IncludeEnvFrom {
			for _, container := range pod.Spec.Containers {
				for _, envFrom := range container.EnvFrom {
					if envFrom.SecretRef != nil && envFrom.SecretRef.Name != "" {
						add(pod.Namespace, envFrom.SecretRef.Name)
					}
				}
			}
			for _, container := range pod.Spec.InitContainers {
				for _, envFrom := range container.EnvFrom {
					if envFrom.SecretRef != nil && envFrom.SecretRef.Name != "" {
						add(pod.Namespace, envFrom.SecretRef.Name)
					}
				}
			}
		}

		// env[].valueFrom.secretKeyRef
		if s.IncludeEnvSecretKeyRefs {
			for _, container := range pod.Spec.Containers {
				for _, env := range container.Env {
					if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
						add(pod.Namespace, env.ValueFrom.SecretKeyRef.Name)
					}
				}
			}
			for _, container := range pod.Spec.InitContainers {
				for _, env := range container.Env {
					if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
						add(pod.Namespace, env.ValueFrom.SecretKeyRef.Name)
					}
				}
			}
		}
	}

	return referencedSecrets, nil
}

// SelectorOrEverything ensures we always have a valid selector.
func (s *ScanTarget) SelectorOrEverything() labels.Selector {
	if s.Selector == nil {
		return labels.Everything()
	}
	return s.Selector
}

func (s *ScanTarget) namespaceAllowed(namespace string) bool {
	if len(s.NamespaceInclude) == 0 && len(s.NamespaceExclude) == 0 {
		return true
	}

	if len(s.NamespaceInclude) > 0 && !matchAnyPattern(namespace, s.NamespaceInclude) {
		return false
	}

	if len(s.NamespaceExclude) > 0 && matchAnyPattern(namespace, s.NamespaceExclude) {
		return false
	}
	return true
}

func (s *ScanTarget) podBindsSecret(pod *corev1.Pod, secretName string) bool {
	var imagePulls []string

	// 1) Pod.imagePullSecrets
	if s.IncludeImagePullSecrets {
		for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
			if imagePullSecret.Name == secretName {
				imagePulls = append(imagePulls, imagePullSecret.Name)
			}
		}
	}

	// 2) volumes[].secret
	volOK := false
	if s.IncludeVolumeSecrets {
		for _, volume := range pod.Spec.Volumes {
			if volume.Secret != nil && volume.Secret.SecretName == secretName {
				volOK = true
				break
			}
		}
	}

	// 3) envFrom
	envFromOK := false
	if s.IncludeEnvFrom {
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secretName {
					envFromOK = true
					break
				}
			}
		}
		if !envFromOK {
			for _, c := range pod.Spec.InitContainers {
				for _, ef := range c.EnvFrom {
					if ef.SecretRef != nil && ef.SecretRef.Name == secretName {
						envFromOK = true
						break
					}
				}
			}
		}
	}

	// 4) env[].valueFrom.secretKeyRef
	envKeyOK := false
	if s.IncludeEnvSecretKeyRefs {
		for _, container := range pod.Spec.Containers {
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secretName {
					envKeyOK = true
					break
				}
			}
		}
		if !envKeyOK {
			for _, container := range pod.Spec.InitContainers {
				for _, env := range container.Env {
					if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secretName {
						envKeyOK = true
						break
					}
				}
			}
		}
	}

	match := volOK || envFromOK || envKeyOK || (len(imagePulls) > 0)
	return match
}

func (s *ScanTarget) topControllerRef(ctx context.Context, pod *corev1.Pod) workloadRef {
	// default: naked Pod
	ref := workloadRef{
		Group: "", Version: "v1", Kind: "Pod",
		Namespace: pod.Namespace, Name: pod.Name, UID: string(pod.UID),
	}
	owner := controllerOwner(pod.OwnerReferences)
	if owner == nil {
		return ref
	}

	switch owner.Kind {
	case "ReplicaSet":
		replicaSet := &appsv1.ReplicaSet{}
		if err := s.ClusterClient.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: owner.Name}, replicaSet); err == nil {
			up := controllerOwner(replicaSet.OwnerReferences)
			if up != nil && up.Kind == "Deployment" {
				return workloadRef{Group: "apps", Version: "v1", Kind: "Deployment", Namespace: replicaSet.Namespace, Name: up.Name}
			}
			return workloadRef{Group: "apps", Version: "v1", Kind: "ReplicaSet", Namespace: replicaSet.Namespace, Name: replicaSet.Name, UID: string(replicaSet.UID)}
		}
		return workloadRef{Group: "apps", Version: "v1", Kind: "ReplicaSet", Namespace: pod.Namespace, Name: owner.Name}

	case "StatefulSet":
		return workloadRef{Group: "apps", Version: "v1", Kind: "StatefulSet", Namespace: pod.Namespace, Name: owner.Name}

	case "DaemonSet":
		return workloadRef{Group: "apps", Version: "v1", Kind: "DaemonSet", Namespace: pod.Namespace, Name: owner.Name}

	case "Job":
		job := &batchv1.Job{}
		if err := s.ClusterClient.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: owner.Name}, job); err == nil {
			up := controllerOwner(job.OwnerReferences)
			if up != nil && up.Kind == "CronJob" {
				return workloadRef{Group: "batch", Version: "v1", Kind: "CronJob", Namespace: job.Namespace, Name: up.Name}
			}
			return workloadRef{Group: "batch", Version: "v1", Kind: "Job", Namespace: job.Namespace, Name: job.Name, UID: string(job.UID)}
		}
		return workloadRef{Group: "batch", Version: "v1", Kind: "Job", Namespace: pod.Namespace, Name: owner.Name}

	default:
		// Unknown owner; fall back to the owner kind/name
		return workloadRef{Group: "", Version: "", Kind: owner.Kind, Namespace: pod.Namespace, Name: owner.Name}
	}
}

func newClient(ctx context.Context, converted *tgtv1alpha1.KubernetesCluster, mgrClient crclient.Client, ctrlClientset typedcorev1.CoreV1Interface) (*ScanTarget, error) {
	cfg, err := buildRestConfig(
		ctx,
		mgrClient,
		ctrlClientset,
		converted.GetNamespace(),
		converted.Spec.Server,
		converted.Spec.Auth,
		converted.Spec.AuthRef,
	)
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}

	kube, err := crclient.New(cfg, crclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("create k8s client: %w", err)
	}

	selector := labels.Everything()
	if converted.Spec.Selector != nil {
		selector, err = metav1.LabelSelectorAsSelector(converted.Spec.Selector)
		if err != nil {
			return nil, fmt.Errorf("invalid selector: %w", err)
		}
	}

	var include, exclude []string
	if converted.Spec.Namespaces != nil {
		include = append(include, converted.Spec.Namespaces.Include...)
		exclude = append(exclude, converted.Spec.Namespaces.Exclude...)
	}

	includeEnvFrom := true
	includeEnvKeys := true
	includeVolumes := true
	includePull := false
	if converted.Spec.Scan != nil {
		includeEnvFrom = converted.Spec.Scan.IncludeEnvFrom
		includeEnvKeys = converted.Spec.Scan.IncludeEnvSecretKeyRefs
		includeVolumes = converted.Spec.Scan.IncludeVolumeSecrets
		includePull = converted.Spec.Scan.IncludeImagePullSecrets
	}

	return &ScanTarget{
		Name:                    converted.GetName(),
		Namespace:               converted.GetNamespace(),
		RestConfig:              cfg,
		ClusterClient:           kube,
		NamespaceInclude:        include,
		NamespaceExclude:        exclude,
		Selector:                selector,
		IncludeImagePullSecrets: includePull,
		IncludeEnvFrom:          includeEnvFrom,
		IncludeEnvSecretKeyRefs: includeEnvKeys,
		IncludeVolumeSecrets:    includeVolumes,
		KubeClient:              mgrClient,
	}, nil
}

func controllerOwner(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for _, r := range refs {
		if r.Controller != nil && *r.Controller {
			return &r
		}
	}
	return nil
}

func controllerString(w workloadRef) string {
	g := w.Group
	if g == "" {
		g = "core"
	}
	return fmt.Sprintf("%s.%s/%s", strings.ToLower(w.Kind), g, w.Name)
}

func stableID(name string, ref workloadRef) string {
	s := fmt.Sprintf("%s-%s-%s-%s", name, ref.Namespace, strings.ToLower(ref.Kind), ref.Name)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.Trim(s, "-")
	return targets.Hash([]byte(s))
}

func buildRestConfig(
	ctx context.Context,
	mgrClient crclient.Client,
	ctrlClientset typedcorev1.CoreV1Interface,
	namespace string,
	server tgtv1alpha1.KubernetesServer,
	auth *tgtv1alpha1.KubernetesAuth,
	authRef *esmeta.SecretKeySelector,
) (*rest.Config, error) {
	if authRef == nil && auth == nil && server.URL == "" {
		return rest.InClusterConfig()
	}

	if authRef != nil {
		cfg, err := fetchSecretKey(ctx, mgrClient, namespace, *authRef)
		if err != nil {
			return nil, err
		}

		return clientcmd.RESTConfigFromKubeConfig(cfg)
	}

	if auth == nil {
		return nil, errors.New("no auth provider given")
	}

	if server.URL == "" {
		return nil, errors.New("no server URL provided")
	}

	cfg := &rest.Config{
		Host: server.URL,
	}

	ca, err := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
		CABundle:   server.CABundle,
		CAProvider: server.CAProvider,
		StoreKind:  resolvers.EmptyStoreKind,
		Namespace:  namespace,
		Client:     mgrClient,
	})
	if err != nil {
		return nil, err
	}

	cfg.TLSClientConfig = rest.TLSClientConfig{
		Insecure: false,
		CAData:   ca,
	}

	switch {
	case auth.Token != nil:
		token, err := fetchSecretKey(ctx, mgrClient, namespace, auth.Token.BearerToken)
		if err != nil {
			return nil, fmt.Errorf("could not fetch Auth.Token.BearerToken: %w", err)
		}

		cfg.BearerToken = string(token)
	case auth.ServiceAccount != nil:
		token, err := serviceAccountToken(ctx, ctrlClientset, namespace, auth.ServiceAccount)
		if err != nil {
			return nil, fmt.Errorf("could not fetch Auth.ServiceAccount: %w", err)
		}

		cfg.BearerToken = string(token)
	case auth.Cert != nil:
		key, cert, err := getClientKeyAndCert(ctx, mgrClient, namespace, auth.Cert)
		if err != nil {
			return nil, fmt.Errorf("could not fetch client key and cert: %w", err)
		}

		cfg.TLSClientConfig.KeyData = key
		cfg.TLSClientConfig.CertData = cert
	default:
		return nil, errors.New("no auth provider given")
	}

	return cfg, nil
}

func getClientKeyAndCert(ctx context.Context, mgrClient crclient.Client, namespace string, authCert *tgtv1alpha1.CertAuth) ([]byte, []byte, error) {
	var err error
	cert, err := fetchSecretKey(ctx, mgrClient, namespace, authCert.ClientCert)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch client certificate: %w", err)
	}
	key, err := fetchSecretKey(ctx, mgrClient, namespace, authCert.ClientKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch client key: %w", err)
	}
	return key, cert, nil
}

func serviceAccountToken(ctx context.Context, ctrlClientset typedcorev1.CoreV1Interface, namespace string, serviceAccountRef *esmeta.ServiceAccountSelector) ([]byte, error) {
	expirationSeconds := int64(3600)
	tr, err := ctrlClientset.ServiceAccounts(namespace).CreateToken(ctx, serviceAccountRef.Name, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         serviceAccountRef.Audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot create service account token: %w", err)
	}
	return []byte(tr.Status.Token), nil
}

func fetchSecretKey(ctx context.Context, mgrClient crclient.Client, namespace string, ref esmeta.SecretKeySelector) ([]byte, error) {
	secret, err := resolvers.SecretKeyRef(
		ctx,
		mgrClient,
		resolvers.EmptyStoreKind,
		namespace,
		&ref,
	)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}

func matchAnyPattern(namespace string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == "*" {
			return true
		}
		ok, err := path.Match(pattern, namespace)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func parseNamespaceName(key string) (string, string, error) {
	parts := strings.Split(key, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("want \"namespace/name\"")
	}
	return parts[0], parts[1], nil
}

func podReadyTime(pod *corev1.Pod) time.Time {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return c.LastTransitionTime.UTC()
		}
	}
	return metav1.Now().UTC()
}

// JobNotReadyErr indicates that a job is not ready yet.
type JobNotReadyErr struct{}

func (e JobNotReadyErr) Error() string {
	return "job not ready"
}
