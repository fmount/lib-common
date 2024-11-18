/*
Copyright 2020 Red Hat

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

package affinity

import (
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// DistributePods - returns rule to ensure that two replicas of the same selector
// should not run if possible on the same worker node
// Deprecated: This function is still maintained to ensure backward compatibility
// with operators that need to bump lib-common without requiring to update the
// signature of the DistributePods function, which is present in all StatefulSets/
// Deployment definitions.
func DistributePods(
	selectorKey string,
	selectorValues []string,
	topologyKey string,
) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			// This rule ensures that two replicas of the same selector
			// should not run if possible on the same worker node
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      selectorKey,
									Operator: metav1.LabelSelectorOpIn,
									Values:   selectorValues,
								},
							},
						},
						// usually corev1.LabelHostname "kubernetes.io/hostname"
						// https://github.com/kubernetes/api/blob/master/core/v1/well_known_labels.go#L20
						TopologyKey: topologyKey,
					},
					Weight: 100,
				},
			},
		},
	}
}

// DistributePodsWithOverrides - returns rule to ensure that two replicas of
// the same selector should not run if possible on the same worker node
func DistributePodsWithOverrides(
	selectorKey string,
	selectorValues []string,
	topologyKey string,
	overrides *OverrideSpec,
) (*corev1.Affinity, error) {
	defaultAffinity := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			// This rule ensures that two replicas of the same selector
			// should not run if possible on the same worker node
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      selectorKey,
									Operator: metav1.LabelSelectorOpIn,
									Values:   selectorValues,
								},
							},
						},
						// usually corev1.LabelHostname "kubernetes.io/hostname"
						// https://github.com/kubernetes/api/blob/master/core/v1/well_known_labels.go#L20
						TopologyKey: topologyKey,
					},
					Weight: 100,
				},
			},
		},
	}
	// patch the default affinity Object with the data passed as input
	if overrides != nil {
		patchedAffinity, err := toCoreAffinity(defaultAffinity, overrides)
		return patchedAffinity, err
	}
	return defaultAffinity, nil
}

func toCoreAffinity(
	affinity *v1.Affinity,
	override *OverrideSpec,
) (*v1.Affinity, error) {

	aff := &v1.Affinity{
		PodAntiAffinity: affinity.PodAntiAffinity,
		PodAffinity:     affinity.PodAffinity,
	}
	if override != nil {
		origAffinit, err := json.Marshal(affinity)
		if err != nil {
			return aff, fmt.Errorf("error marshalling Affinity Spec: %w", err)
		}
		patch, err := json.Marshal(override)
		if err != nil {
			return aff, fmt.Errorf("error marshalling Affinity Spec: %w", err)
		}

		patchedJSON, err := strategicpatch.StrategicMergePatch(origAffinit, patch, v1.Affinity{})
		if err != nil {
			return aff, fmt.Errorf("error patching Affinity Spec: %w", err)
		}

		patchedSpec := v1.Affinity{}
		err = json.Unmarshal(patchedJSON, &patchedSpec)
		if err != nil {
			return aff, fmt.Errorf("error unmarshalling patched Service Spec: %w", err)
		}
		aff = &patchedSpec
	}
	return aff, nil
}
