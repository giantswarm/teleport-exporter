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
	"math/rand"
	"sync"
	"time"

	"github.com/go-logr/logr"

	"github.com/giantswarm/teleport-exporter/internal/metrics"
	"github.com/giantswarm/teleport-exporter/internal/teleport"
)

const (
	// maxBackoffMultiplier is the maximum multiplier for exponential backoff
	maxBackoffMultiplier = 8
	// jitterFraction is the fraction of the interval to use for jitter (0.1 = 10%)
	jitterFraction = 0.1
)

// Config holds the configuration for the collector.
type Config struct {
	TeleportClient  *teleport.Client
	RefreshInterval time.Duration
	APITimeout      time.Duration
	Log             logr.Logger
}

// Collector collects metrics from Teleport and exposes them to Prometheus.
type Collector struct {
	client          *teleport.Client
	refreshInterval time.Duration
	log             logr.Logger

	// Tracking for smart metric cleanup (avoid Reset() gaps)
	mu                sync.RWMutex
	lastDbProtocols   map[string]struct{} // key: "protocol"
	lastDbTypes       map[string]struct{} // key: "type"
	lastClusterName   string
	consecutiveErrors int
}

// New creates a new Collector.
func New(cfg Config) *Collector {
	return &Collector{
		client:          cfg.TeleportClient,
		refreshInterval: cfg.RefreshInterval,
		log:             cfg.Log,
		lastDbProtocols: make(map[string]struct{}),
		lastDbTypes:     make(map[string]struct{}),
	}
}

// Run starts the collector loop with jitter and exponential backoff.
func (c *Collector) Run(ctx context.Context) {
	c.log.Info("starting collector", "refreshInterval", c.refreshInterval)

	// Initial collection with small random delay to avoid thundering herd on startup
	initialJitter := time.Duration(rand.Int63n(int64(c.refreshInterval / 4)))
	c.log.V(1).Info("waiting before initial collection", "jitter", initialJitter)

	select {
	case <-ctx.Done():
		return
	case <-time.After(initialJitter):
		c.collect(ctx)
	}

	for {
		// Calculate next interval with jitter and backoff
		interval := c.calculateNextInterval()

		select {
		case <-ctx.Done():
			c.log.Info("stopping collector")
			return
		case <-time.After(interval):
			c.collect(ctx)
		}
	}
}

// calculateNextInterval returns the next polling interval with jitter and backoff.
func (c *Collector) calculateNextInterval() time.Duration {
	c.mu.RLock()
	errors := c.consecutiveErrors
	c.mu.RUnlock()

	// Base interval
	interval := c.refreshInterval

	// Apply exponential backoff if we have consecutive errors
	if errors > 0 {
		multiplier := 1 << min(errors, maxBackoffMultiplier) // 2^errors, capped
		interval = time.Duration(multiplier) * c.refreshInterval
		c.log.V(1).Info("applying backoff", "consecutiveErrors", errors, "interval", interval)
	}

	// Add jitter (±10% of interval)
	jitter := time.Duration(float64(interval) * jitterFraction * (2*rand.Float64() - 1))
	interval += jitter

	return interval
}

func (c *Collector) collect(ctx context.Context) {
	c.log.V(1).Info("collecting metrics from Teleport")

	startTime := time.Now()
	var hadErrors bool

	// Get cluster name
	clusterName, err := c.client.GetClusterName(ctx)
	if err != nil {
		c.log.Error(err, "failed to get cluster name")
		metrics.TeleportUp.Set(0)
		// Use last known cluster name for error metrics, or "unknown" if not set
		errorClusterName := c.lastClusterName
		if errorClusterName == "" {
			errorClusterName = "unknown"
		}
		metrics.CollectErrorsTotal.WithLabelValues(errorClusterName).Inc()
		c.incrementErrors()
		return
	}

	metrics.TeleportUp.Set(1)
	c.mu.Lock()
	c.lastClusterName = clusterName
	c.mu.Unlock()

	// Collect nodes - on error, keep previous metrics (don't clear them)
	nodes, err := c.client.GetNodes(ctx)
	if err != nil {
		c.log.Error(err, "failed to get nodes")
		metrics.CollectErrorsTotal.WithLabelValues(clusterName).Inc()
		hadErrors = true
	} else {
		c.updateNodeMetrics(clusterName, nodes)
	}

	// Collect Kubernetes clusters
	kubeClusters, err := c.client.GetKubeClusters(ctx)
	if err != nil {
		c.log.Error(err, "failed to get Kubernetes clusters")
		metrics.CollectErrorsTotal.WithLabelValues(clusterName).Inc()
		hadErrors = true
	} else {
		c.updateKubeClusterMetrics(clusterName, kubeClusters)
	}

	// Collect databases
	databases, err := c.client.GetDatabases(ctx)
	if err != nil {
		c.log.Error(err, "failed to get databases")
		metrics.CollectErrorsTotal.WithLabelValues(clusterName).Inc()
		hadErrors = true
	} else {
		c.updateDatabaseMetrics(clusterName, databases)
	}

	// Collect applications
	apps, err := c.client.GetApps(ctx)
	if err != nil {
		c.log.Error(err, "failed to get applications")
		metrics.CollectErrorsTotal.WithLabelValues(clusterName).Inc()
		hadErrors = true
	} else {
		c.updateAppMetrics(clusterName, apps)
	}

	duration := time.Since(startTime)
	metrics.CollectDuration.WithLabelValues(clusterName).Set(duration.Seconds())

	if hadErrors {
		c.incrementErrors()
	} else {
		c.resetErrors()
		metrics.LastSuccessfulCollectTime.WithLabelValues(clusterName).Set(float64(time.Now().Unix()))
	}

	c.log.V(1).Info("metrics collection completed", "duration", duration, "hadErrors", hadErrors)
}

// incrementErrors increases the consecutive error count for backoff calculation.
func (c *Collector) incrementErrors() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consecutiveErrors++
}

// resetErrors resets the consecutive error count after a successful collection.
func (c *Collector) resetErrors() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consecutiveErrors = 0
}

func (c *Collector) updateNodeMetrics(clusterName string, nodes []teleport.NodeInfo) {
	// Count identified vs unidentified nodes
	identifiedCount := 0
	unidentifiedCount := 0

	for _, node := range nodes {
		kubeCluster := extractKubeCluster(node)
		if kubeCluster == "unknown" {
			unidentifiedCount++
		} else {
			identifiedCount++
		}
	}

	// Update low-cardinality metrics
	metrics.NodesTotal.WithLabelValues(clusterName).Set(float64(len(nodes)))
	metrics.NodesIdentifiedTotal.WithLabelValues(clusterName).Set(float64(identifiedCount))
	metrics.NodesUnidentifiedTotal.WithLabelValues(clusterName).Set(float64(unidentifiedCount))

	c.log.V(1).Info("updated node metrics", "count", len(nodes), "identified", identifiedCount, "unidentified", unidentifiedCount)
}

// extractKubeCluster extracts the Kubernetes cluster name from node labels or hostname.
// It looks for common label patterns used by Giant Swarm and other Kubernetes deployments.
func extractKubeCluster(node teleport.NodeInfo) string {
	// Check for common cluster labels (in order of preference)
	clusterLabels := []string{
		"giantswarm.io/cluster",
		"cluster",
		"kubernetes-cluster",
		"kube-cluster",
		"teleport.dev/kubernetes-cluster",
	}

	for _, labelKey := range clusterLabels {
		if cluster, ok := node.Labels[labelKey]; ok && cluster != "" {
			return cluster
		}
	}

	// Try to extract from hostname pattern (e.g., "node-1.clustername" or "clustername-node-1")
	hostname := node.Hostname
	if hostname != "" {
		// Check for pattern like "xxx.clustername.xxx" (domain-style)
		if parts := splitByDot(hostname); len(parts) >= 2 {
			// Return the second part which is often the cluster name
			// e.g., "ip-10-0-0-1.us-west-2.compute.internal" -> "us-west-2"
			// e.g., "node-1.mycluster.local" -> "mycluster"
			return parts[1]
		}
	}

	return "unknown"
}

// splitByDot splits a string by dots and returns the parts.
func splitByDot(s string) []string {
	if s == "" {
		return []string{""}
	}
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func (c *Collector) updateKubeClusterMetrics(clusterName string, clusters []teleport.KubeClusterInfo) {
	// Count MC vs WC clusters
	managementCount := 0
	workloadCount := 0

	for _, cluster := range clusters {
		// Classify as MC (no hyphen) or WC (has hyphen)
		if isWorkloadCluster(cluster.Name) {
			workloadCount++
		} else {
			managementCount++
		}
	}

	// Update low-cardinality metrics
	metrics.KubeClustersTotal.WithLabelValues(clusterName).Set(float64(len(clusters)))
	metrics.KubeManagementClustersTotal.WithLabelValues(clusterName).Set(float64(managementCount))
	metrics.KubeWorkloadClustersTotal.WithLabelValues(clusterName).Set(float64(workloadCount))

	c.log.V(1).Info("updated Kubernetes cluster metrics", "total", len(clusters), "mc", managementCount, "wc", workloadCount)
}

// isWorkloadCluster returns true if the cluster name contains a hyphen (WC naming convention).
func isWorkloadCluster(name string) bool {
	for i := 0; i < len(name); i++ {
		if name[i] == '-' {
			return true
		}
	}
	return false
}

func (c *Collector) updateDatabaseMetrics(clusterName string, databases []teleport.DatabaseInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Count databases by protocol and type
	protocolCounts := make(map[string]int)
	typeCounts := make(map[string]int)

	for _, db := range databases {
		protocol := db.Protocol
		if protocol == "" {
			protocol = "unknown"
		}
		dbType := db.Type
		if dbType == "" {
			dbType = "unknown"
		}
		protocolCounts[protocol]++
		typeCounts[dbType]++
	}

	// Update by-protocol metrics
	currentProtocols := make(map[string]struct{}, len(protocolCounts))
	for protocol, count := range protocolCounts {
		currentProtocols[protocol] = struct{}{}
		metrics.DatabasesByProtocolTotal.WithLabelValues(clusterName, protocol).Set(float64(count))
	}
	for protocol := range c.lastDbProtocols {
		if _, exists := currentProtocols[protocol]; !exists {
			metrics.DatabasesByProtocolTotal.DeleteLabelValues(c.lastClusterName, protocol)
		}
	}

	// Update by-type metrics
	currentTypes := make(map[string]struct{}, len(typeCounts))
	for dbType, count := range typeCounts {
		currentTypes[dbType] = struct{}{}
		metrics.DatabasesByTypeTotal.WithLabelValues(clusterName, dbType).Set(float64(count))
	}
	for dbType := range c.lastDbTypes {
		if _, exists := currentTypes[dbType]; !exists {
			metrics.DatabasesByTypeTotal.DeleteLabelValues(c.lastClusterName, dbType)
		}
	}

	c.lastDbProtocols = currentProtocols
	c.lastDbTypes = currentTypes
	c.lastClusterName = clusterName

	// Update total
	metrics.DatabasesTotal.WithLabelValues(clusterName).Set(float64(len(databases)))
	c.log.V(1).Info("updated database metrics", "count", len(databases), "protocols", len(protocolCounts), "types", len(typeCounts))
}

func (c *Collector) updateAppMetrics(clusterName string, apps []teleport.AppInfo) {
	metrics.AppsTotal.WithLabelValues(clusterName).Set(float64(len(apps)))
	c.log.V(1).Info("updated application metrics", "count", len(apps))
}
