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

	// ClusterInfo provides information about the Teleport cluster.
	ClusterInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "cluster_info",
		Help:      "Information about the Teleport cluster. Value is always 1.",
	}, []string{"cluster_name"})

	// NodesTotal is the total number of nodes registered in Teleport.
	NodesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "nodes_total",
		Help:      "Total number of nodes registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// NodeInfo provides detailed information about each node.
	NodeInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "node_info",
		Help:      "Information about each node registered in Teleport. Value is always 1.",
	}, []string{"cluster_name", "node_name", "hostname", "address", "namespace", "subkind"})

	// KubeClustersTotal is the total number of Kubernetes clusters registered in Teleport.
	KubeClustersTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "kubernetes_clusters_total",
		Help:      "Total number of Kubernetes clusters registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// KubeClusterInfo provides detailed information about each Kubernetes cluster.
	KubeClusterInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "kubernetes_cluster_info",
		Help:      "Information about each Kubernetes cluster registered in Teleport. Value is always 1.",
	}, []string{"cluster_name", "kube_cluster_name"})

	// DatabasesTotal is the total number of databases registered in Teleport.
	DatabasesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "databases_total",
		Help:      "Total number of databases registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// DatabaseInfo provides detailed information about each database.
	DatabaseInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "database_info",
		Help:      "Information about each database registered in Teleport. Value is always 1.",
	}, []string{"cluster_name", "database_name", "protocol", "type"})

	// AppsTotal is the total number of applications registered in Teleport.
	AppsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "apps_total",
		Help:      "Total number of applications registered in the Teleport cluster.",
	}, []string{"cluster_name"})

	// AppInfo provides detailed information about each application.
	AppInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "app_info",
		Help:      "Information about each application registered in Teleport. Value is always 1.",
	}, []string{"cluster_name", "app_name", "public_addr"})

	// CollectDuration is the duration of the last metrics collection.
	CollectDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "collect_duration_seconds",
		Help:      "Duration of the last metrics collection in seconds.",
	})
)
