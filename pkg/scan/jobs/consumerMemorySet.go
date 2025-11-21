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

package job

import (
	"fmt"
	"strings"
	"sync"

	"github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConsumerKey uniquely identifies a consumer.
type ConsumerKey struct {
	TargetNS string
	Target   string
	Type     string
	ID       string
}

// Accumulator for a single consumer.
type consumerAccum struct {
	spec   v1alpha1.ConsumerSpec
	status v1alpha1.ConsumerStatus
}

// ConsumerMemorySet merges many Provider findings into per-consumer accumulators.
type ConsumerMemorySet struct {
	mu     sync.RWMutex
	accums map[ConsumerKey]*consumerAccum
}

// NewConsumerMemorySet creates a new consumer memory set.
func NewConsumerMemorySet() *ConsumerMemorySet {
	return &ConsumerMemorySet{
		accums: make(map[ConsumerKey]*consumerAccum),
	}
}

// Add adds a consumer finding to the memory set.
func (cs *ConsumerMemorySet) Add(target v1alpha1.TargetReference, f scanv1alpha1.ConsumerFinding) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	key := ConsumerKey{
		TargetNS: target.Namespace,
		Target:   target.Name,
		Type:     f.Type,
		ID:       f.ID,
	}
	acc, ok := cs.accums[key]
	if !ok {
		acc = &consumerAccum{
			spec: v1alpha1.ConsumerSpec{
				Target:      target,
				Type:        f.Type,
				ID:          f.ID,
				DisplayName: f.DisplayName,
				Attributes:  f.Attributes,
			},
			status: v1alpha1.ConsumerStatus{},
		}
		acc.status.ObservedIndex = make(map[string]scanv1alpha1.SecretUpdateRecord)
		cs.accums[key] = acc
	}

	already := false
	for _, loc := range acc.status.Locations {
		if EqualLocations(loc, f.Location) {
			already = true
			break
		}
	}
	if !already {
		acc.status.Locations = append(acc.status.Locations, f.Location)
	}

	observedIndexKey := f.Location.RemoteRef.Key
	if strings.TrimSpace(f.Location.RemoteRef.Property) != "" {
		observedIndexKey = fmt.Sprintf("%s.%s", f.Location.RemoteRef.Key, f.Location.RemoteRef.Property)
	}

	currentObservedIndex, ok := acc.status.ObservedIndex[observedIndexKey]
	if !ok || f.ObservedIndex.Timestamp.Time.After(currentObservedIndex.Timestamp.Time) {
		acc.status.ObservedIndex[observedIndexKey] = f.ObservedIndex
	}
}

// List returns all consumers in the memory set.
func (cs *ConsumerMemorySet) List() []v1alpha1.Consumer {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	out := make([]v1alpha1.Consumer, 0, len(cs.accums))
	for _, acc := range cs.accums {
		SortLocations(acc.status.Locations)
		out = append(out, v1alpha1.Consumer{
			ObjectMeta: metav1.ObjectMeta{
				Name: acc.spec.ID,
			},
			Spec:   acc.spec,
			Status: acc.status,
		})
	}
	return out
}
