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
	// TeleportUp indicates whether the exporter can successfully connect to Teleport.
	TeleportUp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "up",
		Help:      "Whether the exporter can successfully connect to Teleport (1 = connected, 0 = disconnected).",
	})

	// NodesTotal is the total number of nodes registered in Teleport.
	NodesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "nodes_total",
		Help:      "Total number of nodes registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// KubeClustersTotal is the total number of Kubernetes clusters registered in Teleport.
	KubeClustersTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "kubernetes_clusters_total",
		Help:      "Total number of Kubernetes clusters registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// DatabasesTotal is the total number of databases registered in Teleport.
	DatabasesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "databases_total",
		Help:      "Total number of databases registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// AppsTotal is the total number of applications registered in Teleport.
	AppsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "apps_total",
		Help:      "Total number of applications registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// ================================================================================
	// HIGH-CARDINALITY METRICS (local Prometheus only, drop before remote_write)
	// These metrics have `_local_` in the name to make filtering easy.
	// Use this regex to drop them: teleport_exporter_local_.*
	// ================================================================================

	// NodesByKubeClusterTotal shows the count of nodes per Kubernetes cluster.
	// HIGH-CARDINALITY: One series per Kubernetes cluster with SSH nodes.
	NodesByKubeClusterTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "local_nodes_by_kubernetes_cluster",
		Help:      "Total number of SSH nodes per Kubernetes cluster. HIGH-CARDINALITY: drop before remote_write.",
	}, []string{"cluster_name", "kube_cluster"})

	// KubeClusterInfo provides detailed information about each Kubernetes cluster.
	// HIGH-CARDINALITY: One series per Kubernetes cluster.
	KubeClusterInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "local_kubernetes_cluster_info",
		Help:      "Information about each Kubernetes cluster registered in Teleport. HIGH-CARDINALITY: drop before remote_write.",
	}, []string{"cluster_name", "kube_cluster_name"})

	// DatabaseInfo provides detailed information about each database.
	// MEDIUM-CARDINALITY: Aggregated by protocol/type combination.
	DatabaseInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "local_database_info",
		Help:      "Count of databases by protocol and type. MEDIUM-CARDINALITY: consider dropping before remote_write.",
	}, []string{"cluster_name", "protocol", "type"})

	// AppInfo provides detailed information about each application.
	// HIGH-CARDINALITY: One series per application.
	AppInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "local_app_info",
		Help:      "Information about each application registered in Teleport. HIGH-CARDINALITY: drop before remote_write.",
	}, []string{"cluster_name", "app_name"})

	// CollectDuration tracks the duration of the last metrics collection.
	CollectDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "collect_duration_seconds",
		Help:      "Duration of the last metrics collection in seconds.",
	})

	// CollectErrorsTotal is the total number of errors encountered during metrics collection.
	CollectErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "collect_errors_total",
		Help:      "Total number of errors encountered during metrics collection.",
	})

	// LastSuccessfulCollectTime is the timestamp of the last successful collection.
	LastSuccessfulCollectTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "last_successful_collect_timestamp_seconds",
		Help:      "Unix timestamp of the last successful metrics collection.",
	})
)
