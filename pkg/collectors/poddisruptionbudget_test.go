/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package collectors

import (
	"testing"
	"time"

	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
)

type mockPodDisruptionBudgetStore struct {
	list func() (v1beta1.PodDisruptionBudgetList, error)
}

func (ns mockPodDisruptionBudgetStore) List() (v1beta1.PodDisruptionBudgetList, error) {
	return ns.list()
}

func TestPodDisruptionBudgetCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
	# HELP kube_poddisruptionbudget_created Unix creation timestamp
	# TYPE kube_poddisruptionbudget_created gauge
	# HELP kube_poddisruptionbudget_status_current_healthy Current number of healthy pods
	# TYPE kube_poddisruptionbudget_status_current_healthy gauge
	# HELP kube_poddisruptionbudget_status_desired_healthy Minimum desired number of healthy pods
	# TYPE kube_poddisruptionbudget_status_desired_healthy gauge
	# HELP kube_poddisruptionbudget_status_pod_disruptions_allowed Number of pod disruptions that are currently allowed
	# TYPE kube_poddisruptionbudget_status_pod_disruptions_allowed gauge
	# HELP kube_poddisruptionbudget_status_expected_pods Total number of pods counted by this disruption budget
	# TYPE kube_poddisruptionbudget_status_expected_pods gauge
	# HELP kube_poddisruptionbudget_status_observed_generation Most recent generation observed when updating this PDB status
	# TYPE kube_poddisruptionbudget_status_observed_generation gauge
	`
	cases := []struct {
		pdbs []v1beta1.PodDisruptionBudget
		want string
	}{
		{
			pdbs: []v1beta1.PodDisruptionBudget{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pdb1",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "ns1",
						Generation:        21,
					},
					Status: v1beta1.PodDisruptionBudgetStatus{
						CurrentHealthy:        12,
						DesiredHealthy:        10,
						PodDisruptionsAllowed: 2,
						ExpectedPods:          15,
						ObservedGeneration:    111,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:       "pdb2",
						Namespace:  "ns2",
						Generation: 14,
					},
					Status: v1beta1.PodDisruptionBudgetStatus{
						CurrentHealthy:        8,
						DesiredHealthy:        9,
						PodDisruptionsAllowed: 0,
						ExpectedPods:          10,
						ObservedGeneration:    1111,
					},
				},
			},
			want: metadata + `
				kube_poddisruptionbudget_created{namespace="ns1",poddisruptionbudget="pdb1"} 1.5e+09
				kube_poddisruptionbudget_status_current_healthy{namespace="ns1",poddisruptionbudget="pdb1"} 12
				kube_poddisruptionbudget_status_current_healthy{namespace="ns2",poddisruptionbudget="pdb2"} 8
				kube_poddisruptionbudget_status_desired_healthy{namespace="ns1",poddisruptionbudget="pdb1"} 10
				kube_poddisruptionbudget_status_desired_healthy{namespace="ns2",poddisruptionbudget="pdb2"} 9
				kube_poddisruptionbudget_status_pod_disruptions_allowed{namespace="ns1",poddisruptionbudget="pdb1"} 2
				kube_poddisruptionbudget_status_pod_disruptions_allowed{namespace="ns2",poddisruptionbudget="pdb2"} 0
				kube_poddisruptionbudget_status_expected_pods{namespace="ns1",poddisruptionbudget="pdb1"} 15
				kube_poddisruptionbudget_status_expected_pods{namespace="ns2",poddisruptionbudget="pdb2"} 10
				kube_poddisruptionbudget_status_observed_generation{namespace="ns1",poddisruptionbudget="pdb1"} 111
				kube_poddisruptionbudget_status_observed_generation{namespace="ns2",poddisruptionbudget="pdb2"} 1111
			`,
		},
	}
	for _, c := range cases {
		pdbc := &podDisruptionBudgetCollector{
			store: &mockPodDisruptionBudgetStore{
				list: func() (v1beta1.PodDisruptionBudgetList, error) {
					return v1beta1.PodDisruptionBudgetList{Items: c.pdbs}, nil
				},
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(pdbc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
