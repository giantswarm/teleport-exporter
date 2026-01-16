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
	lastNodes         map[string]struct{} // key: "subkind"
	lastKubeClusters  map[string]struct{} // key: "kubeClusterName"
	lastDatabases     map[string]struct{} // key: "protocol|type"
	lastApps          map[string]struct{} // key: "appName"
	lastClusterName   string
	consecutiveErrors int
}

// New creates a new Collector.
func New(cfg Config) *Collector {
	return &Collector{
		client:           cfg.TeleportClient,
		refreshInterval:  cfg.RefreshInterval,
		log:              cfg.Log,
		lastNodes:        make(map[string]struct{}),
		lastKubeClusters: make(map[string]struct{}),
		lastDatabases:    make(map[string]struct{}),
		lastApps:         make(map[string]struct{}),
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
		c.incrementErrors()
		return
	}

	metrics.TeleportUp.Set(1)

	// Collect nodes - on error, keep previous metrics (don't clear them)
	nodes, err := c.client.GetNodes(ctx)
	if err != nil {
		c.log.Error(err, "failed to get nodes")
		hadErrors = true
	} else {
		c.updateNodeMetrics(clusterName, nodes)
	}

	// Collect Kubernetes clusters
	kubeClusters, err := c.client.GetKubeClusters(ctx)
	if err != nil {
		c.log.Error(err, "failed to get Kubernetes clusters")
		hadErrors = true
	} else {
		c.updateKubeClusterMetrics(clusterName, kubeClusters)
	}

	// Collect databases
	databases, err := c.client.GetDatabases(ctx)
	if err != nil {
		c.log.Error(err, "failed to get databases")
		hadErrors = true
	} else {
		c.updateDatabaseMetrics(clusterName, databases)
	}

	// Collect applications
	apps, err := c.client.GetApps(ctx)
	if err != nil {
		c.log.Error(err, "failed to get applications")
		hadErrors = true
	} else {
		c.updateAppMetrics(clusterName, apps)
	}

	duration := time.Since(startTime)
	metrics.CollectDuration.Set(duration.Seconds())

	if hadErrors {
		c.incrementErrors()
	} else {
		c.resetErrors()
		metrics.LastSuccessfulCollectTime.Set(float64(time.Now().Unix()))
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
	c.mu.Lock()
	defer c.mu.Unlock()

	// Count nodes by subkind
	subkindCounts := make(map[string]int)
	for _, node := range nodes {
		subkind := node.SubKind
		if subkind == "" {
			subkind = "default"
		}
		subkindCounts[subkind]++
	}

	// Build set of current subkinds and update metrics
	currentNodes := make(map[string]struct{}, len(subkindCounts))
	for subkind, count := range subkindCounts {
		currentNodes[subkind] = struct{}{}
		metrics.NodeInfo.WithLabelValues(clusterName, subkind).Set(float64(count))
	}

	// Delete metrics for subkinds that no longer exist
	for subkind := range c.lastNodes {
		if _, exists := currentNodes[subkind]; !exists {
			metrics.NodeInfo.DeleteLabelValues(c.lastClusterName, subkind)
		}
	}

	c.lastNodes = currentNodes
	c.lastClusterName = clusterName
	metrics.NodesTotal.WithLabelValues(clusterName).Set(float64(len(nodes)))
	c.log.V(1).Info("updated node metrics", "count", len(nodes))
}

func (c *Collector) updateKubeClusterMetrics(clusterName string, clusters []teleport.KubeClusterInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build set of current clusters
	currentClusters := make(map[string]struct{}, len(clusters))

	for _, cluster := range clusters {
		currentClusters[cluster.Name] = struct{}{}
		metrics.KubeClusterInfo.WithLabelValues(clusterName, cluster.Name).Set(1)
	}

	// Delete metrics for clusters that no longer exist
	for kubeClusterName := range c.lastKubeClusters {
		if _, exists := currentClusters[kubeClusterName]; !exists {
			metrics.KubeClusterInfo.DeleteLabelValues(c.lastClusterName, kubeClusterName)
		}
	}

	c.lastKubeClusters = currentClusters
	c.lastClusterName = clusterName
	metrics.KubeClustersTotal.WithLabelValues(clusterName).Set(float64(len(clusters)))
	c.log.V(1).Info("updated Kubernetes cluster metrics", "count", len(clusters))
}

func (c *Collector) updateDatabaseMetrics(clusterName string, databases []teleport.DatabaseInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Count databases by protocol/type combination
	dbCounts := make(map[string]int)
	for _, db := range databases {
		protocol := db.Protocol
		if protocol == "" {
			protocol = "unknown"
		}
		dbType := db.Type
		if dbType == "" {
			dbType = "unknown"
		}
		key := protocol + "|" + dbType
		dbCounts[key]++
	}

	// Build set of current combinations and update metrics
	currentDatabases := make(map[string]struct{}, len(dbCounts))
	for key, count := range dbCounts {
		currentDatabases[key] = struct{}{}
		parts := splitKey(key, 2)
		metrics.DatabaseInfo.WithLabelValues(clusterName, parts[0], parts[1]).Set(float64(count))
	}

	// Delete metrics for combinations that no longer exist
	for key := range c.lastDatabases {
		if _, exists := currentDatabases[key]; !exists {
			parts := splitKey(key, 2)
			if len(parts) == 2 {
				metrics.DatabaseInfo.DeleteLabelValues(c.lastClusterName, parts[0], parts[1])
			}
		}
	}

	c.lastDatabases = currentDatabases
	c.lastClusterName = clusterName
	metrics.DatabasesTotal.WithLabelValues(clusterName).Set(float64(len(databases)))
	c.log.V(1).Info("updated database metrics", "count", len(databases))
}

func (c *Collector) updateAppMetrics(clusterName string, apps []teleport.AppInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build set of current apps
	currentApps := make(map[string]struct{}, len(apps))

	for _, app := range apps {
		currentApps[app.Name] = struct{}{}
		metrics.AppInfo.WithLabelValues(clusterName, app.Name).Set(1)
	}

	// Delete metrics for apps that no longer exist
	for appName := range c.lastApps {
		if _, exists := currentApps[appName]; !exists {
			metrics.AppInfo.DeleteLabelValues(c.lastClusterName, appName)
		}
	}

	c.lastApps = currentApps
	c.lastClusterName = clusterName
	metrics.AppsTotal.WithLabelValues(clusterName).Set(float64(len(apps)))
	c.log.V(1).Info("updated application metrics", "count", len(apps))
}

// splitKey splits a pipe-delimited key into parts.
func splitKey(key string, expectedParts int) []string {
	parts := make([]string, 0, expectedParts)
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}
