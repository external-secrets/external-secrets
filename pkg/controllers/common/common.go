/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"context"
	"time"

	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// BuildManagedSecretClient creates a new client that only sees secrets with the "managed" label.
func BuildManagedSecretClient(mgr ctrl.Manager, namespace string) (client.Client, error) {
	// secrets we manage will have the `reconcile.external-secrets.io/managed=true` label
	managedLabelReq, _ := labels.NewRequirement(esv1beta1.LabelManaged, selection.Equals, []string{esv1beta1.LabelManagedValue})
	managedLabelSelector := labels.NewSelector().Add(*managedLabelReq)

	// create a new cache with a label selector for managed secrets
	// NOTE: this means that the cache/client will be unable to see secrets without the "managed" label
	secretCacheOpts := cache.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Secret{}: {
				Label: managedLabelSelector,
			},
		},
		// this requires us to explicitly start an informer for each object type
		// and helps avoid people mistakenly using the secret client for other resources
		ReaderFailOnMissingInformer: true,
	}
	if namespace != "" {
		secretCacheOpts.DefaultNamespaces = map[string]cache.Config{
			namespace: {},
		}
	}

	secretCache, err := cache.New(mgr.GetConfig(), secretCacheOpts)
	if err != nil {
		return nil, err
	}

	// start an informer for secrets
	// this is required because we set ReaderFailOnMissingInformer to true
	_, err = secretCache.GetInformer(context.Background(), &corev1.Secret{})
	if err != nil {
		return nil, err
	}

	// add the secret cache to the manager, so that it starts at the same time
	err = mgr.Add(secretCache)
	if err != nil {
		return nil, err
	}

	// create a new client that uses the secret cache
	secretClient, err := client.New(mgr.GetConfig(), client.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
		Cache: &client.CacheOptions{
			Reader: secretCache,
		},
	})
	if err != nil {
		return nil, err
	}

	return secretClient, nil
}

// BuildRateLimiter creates a new rate limiter for our controllers.
// NOTE: we dont use `DefaultTypedControllerRateLimiter` because it retries very aggressively, starting at 5ms!
func BuildRateLimiter() workqueue.TypedRateLimiter[reconcile.Request] {
	// exponential backoff rate limiter
	//  - this handles per-item rate limiting for ~failures~
	//  - it uses an exponential backoff strategy were: delay = baseDelay * 2^failures
	//  - graph visualization: https://www.desmos.com/calculator/fexlpdmiti
	failureBaseDelay := 1 * time.Second
	failureMaxDelay := 7 * time.Minute
	failureRateLimiter := workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](failureBaseDelay, failureMaxDelay)

	// overall rate limiter
	//  - this handles overall rate limiting, ignoring individual items and only considering the overall rate
	//  - it implements a "token bucket" of size totalMaxBurst that is initially full,
	//    and which is refilled at rate totalEventsPerSecond tokens per second.
	totalEventsPerSecond := 10
	totalMaxBurst := 100
	totalRateLimiter := &workqueue.TypedBucketRateLimiter[reconcile.Request]{
		Limiter: rate.NewLimiter(rate.Limit(totalEventsPerSecond), totalMaxBurst),
	}

	// return the worst-case (longest) of the rate limiters for a given item
	return workqueue.NewTypedMaxOfRateLimiter[reconcile.Request](failureRateLimiter, totalRateLimiter)
}
