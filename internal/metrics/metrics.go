/*
Copyright 2024.

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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "teleport_exporter"
)

var (
	// --- Connection Status ---

	// TeleportUp indicates whether the exporter can successfully connect to Teleport.
	TeleportUp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "up",
		Help:      "Whether the exporter can successfully connect to Teleport (1 = connected, 0 = disconnected).",
	})

	// --- SSH Nodes ---

	// NodesTotal is the total number of SSH nodes registered in Teleport.
	NodesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "nodes_total",
		Help:      "Total number of SSH nodes registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// NodesIdentifiedTotal is the count of nodes where we could identify the Kubernetes cluster.
	NodesIdentifiedTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "nodes_identified_total",
		Help:      "Number of SSH nodes with identified Kubernetes cluster (via labels or hostname).",
	}, []string{"cluster_name"})

	// NodesUnidentifiedTotal is the count of nodes where we couldn't identify the Kubernetes cluster.
	NodesUnidentifiedTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "nodes_unidentified_total",
		Help:      "Number of SSH nodes with unknown Kubernetes cluster.",
	}, []string{"cluster_name"})

	// --- Kubernetes Clusters ---

	// KubeClustersTotal is the total number of Kubernetes clusters registered in Teleport.
	KubeClustersTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "kubernetes_clusters_total",
		Help:      "Total number of Kubernetes clusters registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// KubeManagementClustersTotal is the count of management clusters (no hyphen in name).
	KubeManagementClustersTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "kubernetes_management_clusters_total",
		Help:      "Number of management clusters (cluster names without hyphen).",
	}, []string{"cluster_name"})

	// KubeWorkloadClustersTotal is the count of workload clusters (has hyphen in name).
	KubeWorkloadClustersTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "kubernetes_workload_clusters_total",
		Help:      "Number of workload clusters (cluster names with hyphen).",
	}, []string{"cluster_name"})

	// --- Databases ---

	// DatabasesTotal is the total number of databases registered in Teleport.
	DatabasesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "databases_total",
		Help:      "Total number of databases registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// DatabasesByProtocolTotal shows database count per protocol.
	DatabasesByProtocolTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "databases_by_protocol_total",
		Help:      "Number of databases by protocol (postgres, mysql, mongodb, etc.).",
	}, []string{"cluster_name", "protocol"})

	// DatabasesByTypeTotal shows database count per type.
	DatabasesByTypeTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "databases_by_type_total",
		Help:      "Number of databases by type (rds, self-hosted, cloud-sql, etc.).",
	}, []string{"cluster_name", "type"})

	// --- Applications ---

	// AppsTotal is the total number of applications registered in Teleport.
	AppsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "apps_total",
		Help:      "Total number of applications registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// --- Exporter Health ---

	// CollectDuration tracks the duration of the last metrics collection.
	CollectDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "collect_duration_seconds",
		Help:      "Duration of the last metrics collection in seconds.",
	}, []string{"cluster_name"})

	// CollectErrorsTotal is the total number of errors encountered during metrics collection.
	CollectErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "collect_errors_total",
		Help:      "Total number of errors encountered during metrics collection.",
	}, []string{"cluster_name"})

	// LastSuccessfulCollectTime is the timestamp of the last successful collection.
	LastSuccessfulCollectTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "last_successful_collect_timestamp_seconds",
		Help:      "Unix timestamp of the last successful metrics collection.",
	}, []string{"cluster_name"})
)
