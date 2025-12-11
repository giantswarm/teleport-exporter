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

package collector

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	"github.com/giantswarm/teleport-exporter/internal/metrics"
	"github.com/giantswarm/teleport-exporter/internal/teleport"
)

// Config holds the configuration for the collector.
type Config struct {
	TeleportClient  *teleport.Client
	RefreshInterval time.Duration
	Log             logr.Logger
}

// Collector collects metrics from Teleport and exposes them to Prometheus.
type Collector struct {
	client          *teleport.Client
	refreshInterval time.Duration
	log             logr.Logger
}

// New creates a new Collector.
func New(cfg Config) *Collector {
	return &Collector{
		client:          cfg.TeleportClient,
		refreshInterval: cfg.RefreshInterval,
		log:             cfg.Log,
	}
}

// Run starts the collector loop.
func (c *Collector) Run(ctx context.Context) {
	c.log.Info("starting collector", "refreshInterval", c.refreshInterval)

	// Initial collection
	c.collect(ctx)

	ticker := time.NewTicker(c.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("stopping collector")
			return
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

func (c *Collector) collect(ctx context.Context) {
	c.log.V(1).Info("collecting metrics from Teleport")

	startTime := time.Now()

	// Get cluster name
	clusterName, err := c.client.GetClusterName(ctx)
	if err != nil {
		c.log.Error(err, "failed to get cluster name")
		metrics.TeleportUp.Set(0)
		return
	}

	metrics.TeleportUp.Set(1)
	metrics.ClusterInfo.WithLabelValues(clusterName).Set(1)

	// Collect nodes
	nodes, err := c.client.GetNodes(ctx)
	if err != nil {
		c.log.Error(err, "failed to get nodes")
	} else {
		c.updateNodeMetrics(clusterName, nodes)
	}

	// Collect Kubernetes clusters
	kubeClusters, err := c.client.GetKubeClusters(ctx)
	if err != nil {
		c.log.Error(err, "failed to get Kubernetes clusters")
	} else {
		c.updateKubeClusterMetrics(clusterName, kubeClusters)
	}

	// Collect databases
	databases, err := c.client.GetDatabases(ctx)
	if err != nil {
		c.log.Error(err, "failed to get databases")
	} else {
		c.updateDatabaseMetrics(clusterName, databases)
	}

	// Collect applications
	apps, err := c.client.GetApps(ctx)
	if err != nil {
		c.log.Error(err, "failed to get applications")
	} else {
		c.updateAppMetrics(clusterName, apps)
	}

	duration := time.Since(startTime)
	metrics.CollectDuration.Set(duration.Seconds())
	c.log.V(1).Info("metrics collection completed", "duration", duration)
}

func (c *Collector) updateNodeMetrics(clusterName string, nodes []teleport.NodeInfo) {
	// Reset the node info metric to clear stale entries
	metrics.NodeInfo.Reset()

	for _, node := range nodes {
		metrics.NodeInfo.WithLabelValues(
			clusterName,
			node.Name,
			node.Hostname,
			node.Address,
			node.Namespace,
			node.SubKind,
		).Set(1)
	}

	metrics.NodesTotal.WithLabelValues(clusterName).Set(float64(len(nodes)))
	c.log.V(1).Info("updated node metrics", "count", len(nodes))
}

func (c *Collector) updateKubeClusterMetrics(clusterName string, clusters []teleport.KubeClusterInfo) {
	// Reset the kube cluster info metric to clear stale entries
	metrics.KubeClusterInfo.Reset()

	for _, cluster := range clusters {
		metrics.KubeClusterInfo.WithLabelValues(
			clusterName,
			cluster.Name,
		).Set(1)
	}

	metrics.KubeClustersTotal.WithLabelValues(clusterName).Set(float64(len(clusters)))
	c.log.V(1).Info("updated Kubernetes cluster metrics", "count", len(clusters))
}

func (c *Collector) updateDatabaseMetrics(clusterName string, databases []teleport.DatabaseInfo) {
	// Reset the database info metric to clear stale entries
	metrics.DatabaseInfo.Reset()

	for _, db := range databases {
		metrics.DatabaseInfo.WithLabelValues(
			clusterName,
			db.Name,
			db.Protocol,
			db.Type,
		).Set(1)
	}

	metrics.DatabasesTotal.WithLabelValues(clusterName).Set(float64(len(databases)))
	c.log.V(1).Info("updated database metrics", "count", len(databases))
}

func (c *Collector) updateAppMetrics(clusterName string, apps []teleport.AppInfo) {
	// Reset the app info metric to clear stale entries
	metrics.AppInfo.Reset()

	for _, app := range apps {
		metrics.AppInfo.WithLabelValues(
			clusterName,
			app.Name,
			app.PublicAddr,
		).Set(1)
	}

	metrics.AppsTotal.WithLabelValues(clusterName).Set(float64(len(apps)))
	c.log.V(1).Info("updated application metrics", "count", len(apps))
}
