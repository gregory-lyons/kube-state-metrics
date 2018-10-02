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
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descPodDisruptionBudgetLabelsDefaultLabels = []string{"poddisruptionbudget", "namespace"}

	descPodDisruptionBudgetCreated = prometheus.NewDesc(
		"kube_poddisruptionbudget_created",
		"Unix creation timestamp",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)

	descPodDisruptionBudgetStatusCurrentHealthy = prometheus.NewDesc(
		"kube_poddisruptionbudget_status_current_healthy",
		"Current number of healthy pods",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusDesiredHealthy = prometheus.NewDesc(
		"kube_poddisruptionbudget_status_desired_healthy",
		"Minimum desired number of healthy pods",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusPodDisruptionsAllowed = prometheus.NewDesc(
		"kube_poddisruptionbudget_status_pod_disruptions_allowed",
		"Number of pod disruptions that are currently allowed",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusExpectedPods = prometheus.NewDesc(
		"kube_poddisruptionbudget_status_expected_pods",
		"Total number of pods counted by this disruption budget",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusObservedGeneration = prometheus.NewDesc(
		"kube_poddisruptionbudget_status_observed_generation",
		"Most recent generation observed when updating this PDB status",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
)

type PodDisruptionBudgetLister func() (v1beta1.PodDisruptionBudgetList, error)

func (l PodDisruptionBudgetLister) List() (v1beta1.PodDisruptionBudgetList, error) {
	return l()
}

func RegisterPodDisruptionBudgetCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Policy().V1beta1().PodDisruptionBudgets().Informer().(cache.SharedInformer))
	}

	podDisruptionBudgetLister := PodDisruptionBudgetLister(func() (podDisruptionBudgets v1beta1.PodDisruptionBudgetList, err error) {
		for _, pdbinf := range infs {
			for _, pdb := range pdbinf.GetStore().List() {
				podDisruptionBudgets.Items = append(podDisruptionBudgets.Items, *(pdb.(*v1beta1.PodDisruptionBudget)))
			}
		}
		return podDisruptionBudgets, nil
	})

	registry.MustRegister(&podDisruptionBudgetCollector{store: podDisruptionBudgetLister, opts: opts})
	infs.Run(context.Background().Done())
}

type podDisruptionBudgetStore interface {
	List() (v1beta1.PodDisruptionBudgetList, error)
}

// podDisruptionBudgetCollector collects metrics about all pod disruption budgets in the cluster.
type podDisruptionBudgetCollector struct {
	store podDisruptionBudgetStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (pdbc *podDisruptionBudgetCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPodDisruptionBudgetCreated
	ch <- descPodDisruptionBudgetStatusCurrentHealthy
	ch <- descPodDisruptionBudgetStatusDesiredHealthy
	ch <- descPodDisruptionBudgetStatusPodDisruptionsAllowed
	ch <- descPodDisruptionBudgetStatusExpectedPods
	ch <- descPodDisruptionBudgetStatusObservedGeneration
}

// Collect implements the prometheus.Collector interface.
func (pdbc *podDisruptionBudgetCollector) Collect(ch chan<- prometheus.Metric) {
	podDisruptionBudget, err := pdbc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "poddisruptionbudget"}).Inc()
		glog.Errorf("listing pod disruption budgets failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "poddisruptionbudget"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "poddisruptionbudget"}).Observe(float64(len(podDisruptionBudget.Items)))
	for _, pdb := range podDisruptionBudget.Items {
		pdbc.collectPodDisruptionBudget(ch, pdb)
	}

	glog.V(4).Infof("collected %d poddisruptionsbudgets", len(podDisruptionBudget.Items))
}

func (pdbc *podDisruptionBudgetCollector) collectPodDisruptionBudget(ch chan<- prometheus.Metric, pdb v1beta1.PodDisruptionBudget) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{pdb.Name, pdb.Namespace}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	if !pdb.CreationTimestamp.IsZero() {
		addGauge(descPodDisruptionBudgetCreated, float64(pdb.CreationTimestamp.Unix()))
	}
	addGauge(descPodDisruptionBudgetStatusCurrentHealthy, float64(pdb.Status.CurrentHealthy))
	addGauge(descPodDisruptionBudgetStatusDesiredHealthy, float64(pdb.Status.DesiredHealthy))
	addGauge(descPodDisruptionBudgetStatusPodDisruptionsAllowed, float64(pdb.Status.PodDisruptionsAllowed))
	addGauge(descPodDisruptionBudgetStatusExpectedPods, float64(pdb.Status.ExpectedPods))
	addGauge(descPodDisruptionBudgetStatusObservedGeneration, float64(pdb.Status.ObservedGeneration))
}
