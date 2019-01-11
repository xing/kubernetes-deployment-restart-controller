package controller

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ResourceVersionsTotal exposes the total number of resource versions observed
	ResourceVersionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "deployment_restart_controller",
		Name:      "resource_versions_total",
		Help:      "The total number of distinct resource versions observed.",
	}, []string{})

	// ConfigsTotal exposes the total number of tracked configs
	ConfigsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "deployment_restart_controller",
		Name:      "configs_total",
		Help:      "The total number of tracked configs.",
	}, []string{})

	// DeploymentsTotal exposes the total number of tracked deployments
	DeploymentsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "deployment_restart_controller",
		Name:      "deployments_total",
		Help:      "The total number of tracked deployments.",
	}, []string{})

	// DeploymentAnnotationUpdatesTotal exposes the total number of deployment annotation updates
	DeploymentAnnotationUpdatesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "deployment_restart_controller",
		Name:      "deployment_annotation_updates_total",
		Help:      "The total number of deployment annotation updates.",
	}, []string{})

	// DeploymentRestartsTotal exposes the total number of deployment restarts triggered
	DeploymentRestartsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "deployment_restart_controller",
		Name:      "deployment_restarts_total",
		Help:      "The total number of deployment restarts triggered.",
	}, []string{})

	// ChangesProcessedTotal exposes the total number of resource changes processed
	ChangesProcessedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "deployment_restart_controller",
		Name:      "changes_processed_total",
		Help:      "The total number of resource changes processed.",
	}, []string{})

	// ChangesWaitingTotal exposes the total number of changes waiting to be processed
	ChangesWaitingTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "deployment_restart_controller",
		Name:      "changes_waiting_total",
		Help:      "The total number of changes waiting to be processed.",
	}, []string{})
)

func init() {
	counters := []*prometheus.CounterVec{
		ResourceVersionsTotal,
		DeploymentAnnotationUpdatesTotal,
		DeploymentRestartsTotal,
		ChangesProcessedTotal,
	}

	gauges := []*prometheus.GaugeVec{
		ConfigsTotal,
		DeploymentsTotal,
		ChangesWaitingTotal,
	}

	// Unincremented counters and unset gauges do not show up in /metrics and produce
	// incomplete data in the metrics data store.

	for _, v := range counters {
		prometheus.MustRegister(v)
		v.WithLabelValues().Add(0)
	}

	for _, v := range gauges {
		prometheus.MustRegister(v)
		v.WithLabelValues().Set(0)
	}
}
