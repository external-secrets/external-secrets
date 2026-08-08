// /*
// Copyright © The ESO Authors
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

//go:build perf

package perf

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const createConcurrency = 64

// CreateNamespaces creates n namespaces with the given prefix, up to createConcurrency at a time.
// Returns the list of created namespace names.
func CreateNamespaces(ctx context.Context, c client.Client, n int, prefix string) ([]string, error) {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("%s-%05d", prefix, i)
	}

	sem := make(chan struct{}, createConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, name := range names {
		wg.Add(1)
		sem <- struct{}{}
		go func(nsName string) {
			defer wg.Done()
			defer func() { <-sem }()
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
			if err := c.Create(ctx, ns); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(name)
	}
	wg.Wait()
	return names, firstErr
}

// CreateSecretStores creates one SecretStore per namespace, all referencing providerSpec.
func CreateSecretStores(ctx context.Context, c client.Client, namespaces []string, providerSpec *esv1.SecretStoreProvider) error {
	sem := make(chan struct{}, createConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, ns := range namespaces {
		wg.Add(1)
		sem <- struct{}{}
		go func(namespace string) {
			defer wg.Done()
			defer func() { <-sem }()
			store := &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "perf-store",
					Namespace: namespace,
				},
				Spec: esv1.SecretStoreSpec{
					Provider: providerSpec,
				},
			}
			if err := c.Create(ctx, store); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(ns)
	}
	wg.Wait()
	return firstErr
}

// CreateExternalSecrets creates n ExternalSecrets in a single namespace, all referencing storeRef.
// Each ES requests one key ("perf-key") from the store, with the given refreshInterval.
func CreateExternalSecrets(ctx context.Context, c client.Client, namespace string, n int, storeRef esv1.SecretStoreRef, refreshInterval time.Duration) error {
	sem := make(chan struct{}, createConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("perf-es-%05d", idx),
					Namespace: namespace,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef:  storeRef,
					RefreshInterval: &metav1.Duration{Duration: refreshInterval},
					Target: esv1.ExternalSecretTarget{
						Name: fmt.Sprintf("perf-secret-%05d", idx),
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "perf-key",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "perf-remote-key",
							},
						},
					},
				},
			}
			if err := c.Create(ctx, es); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return firstErr
}

// WaitAllReady polls until n ExternalSecrets in namespace all have Ready=True, or timeout is reached.
// Returns the elapsed time from the first call until all are ready.
func WaitAllReady(ctx context.Context, c client.Client, namespace string, n int, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	deadline := start.Add(timeout)
	for time.Now().Before(deadline) {
		esList := &esv1.ExternalSecretList{}
		if err := c.List(ctx, esList, client.InNamespace(namespace)); err != nil {
			return 0, err
		}
		ready := 0
		for i := range esList.Items {
			if isESReady(&esList.Items[i]) {
				ready++
			}
		}
		if ready >= n {
			return time.Since(start), nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return 0, fmt.Errorf("timeout after %s waiting for %d ExternalSecrets to be Ready in %s", timeout, n, namespace)
}

// WaitAllStoresReady polls until every namespace in namespaces has its "perf-store" SecretStore
// at Ready=True, or timeout is reached. Uses a single global List for efficiency.
func WaitAllStoresReady(ctx context.Context, c client.Client, namespaces []string, timeout time.Duration) (time.Duration, error) {
	nsSet := make(map[string]struct{}, len(namespaces))
	for _, ns := range namespaces {
		nsSet[ns] = struct{}{}
	}
	expected := len(namespaces)

	start := time.Now()
	deadline := start.Add(timeout)
	for time.Now().Before(deadline) {
		storeList := &esv1.SecretStoreList{}
		if err := c.List(ctx, storeList); err != nil {
			return 0, err
		}
		ready := 0
		for i := range storeList.Items {
			s := &storeList.Items[i]
			if _, ok := nsSet[s.Namespace]; !ok {
				continue
			}
			if isStoreReady(s) {
				ready++
			}
		}
		if ready >= expected {
			return time.Since(start), nil
		}
		time.Sleep(time.Second)
	}
	return 0, fmt.Errorf("timeout after %s waiting for %d SecretStores to be Ready", timeout, expected)
}

// WaitAllReadyMultiNS polls until (len(namespaces) * perNS) ExternalSecrets across all listed
// namespaces reach Ready=True. Uses a global List for efficiency at scale.
func WaitAllReadyMultiNS(ctx context.Context, c client.Client, namespaces []string, perNS int, timeout time.Duration) (time.Duration, error) {
	nsSet := make(map[string]struct{}, len(namespaces))
	for _, ns := range namespaces {
		nsSet[ns] = struct{}{}
	}
	expected := len(namespaces) * perNS

	start := time.Now()
	deadline := start.Add(timeout)
	for time.Now().Before(deadline) {
		esList := &esv1.ExternalSecretList{}
		if err := c.List(ctx, esList); err != nil {
			return 0, err
		}
		ready := 0
		for i := range esList.Items {
			es := &esList.Items[i]
			if _, ok := nsSet[es.Namespace]; !ok {
				continue
			}
			if isESReady(es) {
				ready++
			}
		}
		if ready >= expected {
			return time.Since(start), nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return 0, fmt.Errorf("timeout after %s waiting for %d ExternalSecrets to be Ready across %d namespaces", timeout, expected, len(namespaces))
}

func isESReady(es *esv1.ExternalSecret) bool {
	cond := esv1.GetExternalSecretCondition(es.Status, esv1.ExternalSecretReady)
	return cond != nil && cond.Status == corev1.ConditionTrue
}

func isStoreReady(store *esv1.SecretStore) bool {
	for _, c := range store.Status.Conditions {
		if c.Type == esv1.SecretStoreReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
