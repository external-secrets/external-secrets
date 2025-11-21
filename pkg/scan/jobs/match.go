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
	"github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Candidate represents a candidate match for a finding.
type Candidate struct {
	ID          string
	Name        string
	DisplayName string
	Inter       int
	Union       int
	Jaccard     float64
	CurrCount   int
}

// JaccardParams defines the thresholds used to decide if two sets of locations
// are considered similar enough to represent the same logical finding.
// A match is accepted if either:
//   - The Jaccard index is greater than or equal to MinJaccard, OR
//   - The raw intersection has at least MinIntersection elements and
//     covers at least half of the smaller set.
type JaccardParams struct {
	MinJaccard      float64 // minimum Jaccard index (0.0–1.0) to accept similarity
	MinIntersection int     // minimum number of common elements required for fallback rule
}

// Jaccard computes the Jaccard similarity between two sets.
func Jaccard(newSet, currentSet map[string]struct{}) (intersection, union int, j float64) {
	// Ensure we iterate over the larger set for efficiency
	if len(newSet) < len(currentSet) {
		newSet, currentSet = currentSet, newSet
	}
	intersection = 0
	for k := range newSet {
		if _, ok := currentSet[k]; ok {
			intersection++
		}
	}
	union = len(newSet) + len(currentSet) - intersection
	if union == 0 {
		return 0, 0, 1.0
	}
	return intersection, union, float64(intersection) / float64(union)
}

// LocationsToStringSet converts a slice of locations to a set of strings.
func LocationsToStringSet(locations []scanv1alpha1.SecretInStoreRef) map[string]struct{} {
	locationsSet := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		locationsSet[Sanitize(location)] = struct{}{}
	}
	return locationsSet
}

// AssignIDs assigns IDs to new findings based on similarity to current findings.
func AssignIDs(currentFindings, newFindings []v1alpha1.Finding, params JaccardParams) []v1alpha1.Finding {
	for i := range newFindings {
		newLocationsSet := LocationsToStringSet(newFindings[i].Status.Locations)
		best := Candidate{}
		for _, currentFinding := range currentFindings {
			currentLocationsSet := LocationsToStringSet(currentFinding.Status.Locations)
			intersection, union, jaccardIndex := Jaccard(newLocationsSet, currentLocationsSet)
			isSimilarEnough := jaccardIndex >= params.MinJaccard
			// Fallback for small sets: accept if there's a strong raw overlap,
			// meaning at least MinIntersection elements match AND the overlap
			// covers at least half of the smaller set.
			isSimilarEnough = isSimilarEnough || (intersection >= params.MinIntersection &&
				intersection*2 >= min(len(newLocationsSet), len(currentLocationsSet)))
			if !isSimilarEnough {
				continue
			}
			cand := Candidate{
				ID:          currentFinding.Spec.ID,
				Name:        currentFinding.Name,
				DisplayName: currentFinding.Spec.DisplayName,
				Inter:       intersection,
				Union:       union,
				Jaccard:     jaccardIndex,
				CurrCount:   len(currentLocationsSet),
			}
			if better(cand, best) {
				best = cand
			}
		}
		if best.ID != "" {
			newFindings[i].Spec.ID = best.ID
			newFindings[i].Spec.DisplayName = best.DisplayName
			newFindings[i].ObjectMeta = metav1.ObjectMeta{
				Name: best.Name,
			}
		} else {
			newUUID := uuid.NewString()
			newFindings[i].Spec.ID = newUUID
			newFindings[i].ObjectMeta = metav1.ObjectMeta{
				Name: newUUID,
			}
		}
	}
	return newFindings
}

// Checks if new candidate is better than current candidate.
func better(newCandidate, currCandidate Candidate) bool {
	if currCandidate.ID == "" {
		return true
	}
	if newCandidate.Jaccard != currCandidate.Jaccard {
		return newCandidate.Jaccard > currCandidate.Jaccard
	}
	if newCandidate.Inter != currCandidate.Inter {
		return newCandidate.Inter > currCandidate.Inter
	}
	// deterministic tie-breaker
	return newCandidate.ID < currCandidate.ID
}
