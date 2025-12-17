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
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/giantswarm/teleport-exporter/internal/metrics"
	"github.com/giantswarm/teleport-exporter/internal/teleport"
)

// newTestCollector creates a Collector with initialized maps for testing.
func newTestCollector() *Collector {
	return &Collector{
		log:              logr.Discard(),
		lastNodes:        make(map[string]struct{}),
		lastKubeClusters: make(map[string]struct{}),
		lastDatabases:    make(map[string]struct{}),
		lastApps:         make(map[string]struct{}),
	}
}

func TestCollector_UpdateNodeMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.NodeInfo.Reset()
	metrics.NodesTotal.Reset()

	c := newTestCollector()

	nodes := []teleport.NodeInfo{
		{
			Name:      "node-1",
			Hostname:  "host1.example.com",
			Address:   "192.168.1.1:3022",
			Namespace: "default",
			SubKind:   "openssh",
		},
		{
			Name:      "node-2",
			Hostname:  "host2.example.com",
			Address:   "192.168.1.2:3022",
			Namespace: "default",
			SubKind:   "teleport",
		},
	}

	c.updateNodeMetrics("test-cluster", nodes)

	// Verify total count
	totalValue := testutil.ToFloat64(metrics.NodesTotal.WithLabelValues("test-cluster"))
	if totalValue != 2 {
		t.Errorf("expected NodesTotal to be 2, got %f", totalValue)
	}

	// Verify node info metrics exist
	nodeInfoValue := testutil.ToFloat64(metrics.NodeInfo.WithLabelValues(
		"test-cluster", "node-1", "host1.example.com", "192.168.1.1:3022", "default", "openssh",
	))
	if nodeInfoValue != 1 {
		t.Errorf("expected NodeInfo for node-1 to be 1, got %f", nodeInfoValue)
	}

	// Verify that tracking map is updated
	if len(c.lastNodes) != 2 {
		t.Errorf("expected lastNodes to have 2 entries, got %d", len(c.lastNodes))
	}
}

func TestCollector_UpdateNodeMetrics_RemovesStaleNodes(t *testing.T) {
	// Reset metrics before test
	metrics.NodeInfo.Reset()
	metrics.NodesTotal.Reset()

	c := newTestCollector()

	// First update with 2 nodes
	nodes := []teleport.NodeInfo{
		{Name: "node-1", Hostname: "host1", Address: "1.1.1.1", Namespace: "default", SubKind: "teleport"},
		{Name: "node-2", Hostname: "host2", Address: "2.2.2.2", Namespace: "default", SubKind: "teleport"},
	}
	c.updateNodeMetrics("test-cluster", nodes)

	// Verify both nodes exist
	if testutil.ToFloat64(metrics.NodesTotal.WithLabelValues("test-cluster")) != 2 {
		t.Error("expected 2 nodes after first update")
	}

	// Second update with only 1 node (node-2 removed)
	nodes = []teleport.NodeInfo{
		{Name: "node-1", Hostname: "host1", Address: "1.1.1.1", Namespace: "default", SubKind: "teleport"},
	}
	c.updateNodeMetrics("test-cluster", nodes)

	// Verify only 1 node exists
	totalValue := testutil.ToFloat64(metrics.NodesTotal.WithLabelValues("test-cluster"))
	if totalValue != 1 {
		t.Errorf("expected NodesTotal to be 1 after removal, got %f", totalValue)
	}

	// Verify tracking map is updated
	if len(c.lastNodes) != 1 {
		t.Errorf("expected lastNodes to have 1 entry after removal, got %d", len(c.lastNodes))
	}
}

func TestCollector_UpdateKubeClusterMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.KubeClusterInfo.Reset()
	metrics.KubeClustersTotal.Reset()

	c := newTestCollector()

	clusters := []teleport.KubeClusterInfo{
		{Name: "kube-cluster-1"},
		{Name: "kube-cluster-2"},
		{Name: "kube-cluster-3"},
	}

	c.updateKubeClusterMetrics("test-cluster", clusters)

	// Verify total count
	totalValue := testutil.ToFloat64(metrics.KubeClustersTotal.WithLabelValues("test-cluster"))
	if totalValue != 3 {
		t.Errorf("expected KubeClustersTotal to be 3, got %f", totalValue)
	}

	// Verify tracking map
	if len(c.lastKubeClusters) != 3 {
		t.Errorf("expected lastKubeClusters to have 3 entries, got %d", len(c.lastKubeClusters))
	}
}

func TestCollector_UpdateDatabaseMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.DatabaseInfo.Reset()
	metrics.DatabasesTotal.Reset()

	c := newTestCollector()

	databases := []teleport.DatabaseInfo{
		{
			Name:     "postgres-db",
			Protocol: "postgres",
			Type:     "rds",
		},
		{
			Name:     "mysql-db",
			Protocol: "mysql",
			Type:     "self-hosted",
		},
	}

	c.updateDatabaseMetrics("test-cluster", databases)

	// Verify total count
	totalValue := testutil.ToFloat64(metrics.DatabasesTotal.WithLabelValues("test-cluster"))
	if totalValue != 2 {
		t.Errorf("expected DatabasesTotal to be 2, got %f", totalValue)
	}

	// Verify database info
	dbInfoValue := testutil.ToFloat64(metrics.DatabaseInfo.WithLabelValues(
		"test-cluster", "postgres-db", "postgres", "rds",
	))
	if dbInfoValue != 1 {
		t.Errorf("expected DatabaseInfo for postgres-db to be 1, got %f", dbInfoValue)
	}
}

func TestCollector_UpdateAppMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.AppInfo.Reset()
	metrics.AppsTotal.Reset()

	c := newTestCollector()

	apps := []teleport.AppInfo{
		{
			Name:       "grafana",
			PublicAddr: "grafana.example.com",
			URI:        "http://localhost:3000",
		},
	}

	c.updateAppMetrics("test-cluster", apps)

	// Verify total count
	totalValue := testutil.ToFloat64(metrics.AppsTotal.WithLabelValues("test-cluster"))
	if totalValue != 1 {
		t.Errorf("expected AppsTotal to be 1, got %f", totalValue)
	}
}

func TestCollector_New(t *testing.T) {
	cfg := Config{
		TeleportClient:  nil, // Would be set in real usage
		RefreshInterval: 60 * time.Second,
		Log:             logr.Discard(),
	}

	c := New(cfg)

	if c.refreshInterval != 60*time.Second {
		t.Errorf("expected refreshInterval to be 60s, got %v", c.refreshInterval)
	}

	// Verify maps are initialized
	if c.lastNodes == nil {
		t.Error("expected lastNodes to be initialized")
	}
	if c.lastKubeClusters == nil {
		t.Error("expected lastKubeClusters to be initialized")
	}
	if c.lastDatabases == nil {
		t.Error("expected lastDatabases to be initialized")
	}
	if c.lastApps == nil {
		t.Error("expected lastApps to be initialized")
	}
}

func TestCollector_BackoffCalculation(t *testing.T) {
	c := newTestCollector()
	c.refreshInterval = 60 * time.Second

	// With no errors, interval should be close to base (with some jitter)
	interval := c.calculateNextInterval()
	if interval < 50*time.Second || interval > 70*time.Second {
		t.Errorf("expected interval to be around 60s with no errors, got %v", interval)
	}

	// Simulate errors
	c.consecutiveErrors = 1
	interval = c.calculateNextInterval()
	// With 1 error, multiplier is 2, so ~120s
	if interval < 100*time.Second || interval > 140*time.Second {
		t.Errorf("expected interval to be around 120s with 1 error, got %v", interval)
	}

	c.consecutiveErrors = 3
	interval = c.calculateNextInterval()
	// With 3 errors, multiplier is 8, so ~480s (8 minutes)
	if interval < 400*time.Second || interval > 560*time.Second {
		t.Errorf("expected interval to be around 480s with 3 errors, got %v", interval)
	}
}

func TestCollector_ErrorTracking(t *testing.T) {
	c := newTestCollector()

	if c.consecutiveErrors != 0 {
		t.Error("expected initial consecutiveErrors to be 0")
	}

	c.incrementErrors()
	if c.consecutiveErrors != 1 {
		t.Errorf("expected consecutiveErrors to be 1, got %d", c.consecutiveErrors)
	}

	c.incrementErrors()
	if c.consecutiveErrors != 2 {
		t.Errorf("expected consecutiveErrors to be 2, got %d", c.consecutiveErrors)
	}

	c.resetErrors()
	if c.consecutiveErrors != 0 {
		t.Errorf("expected consecutiveErrors to be 0 after reset, got %d", c.consecutiveErrors)
	}
}

func TestSplitKey(t *testing.T) {
	tests := []struct {
		key           string
		expectedParts int
		expected      []string
	}{
		{"a|b|c", 3, []string{"a", "b", "c"}},
		{"cluster|node|host|addr|ns|kind", 6, []string{"cluster", "node", "host", "addr", "ns", "kind"}},
		{"single", 1, []string{"single"}},
		{"a|b", 2, []string{"a", "b"}},
	}

	for _, tt := range tests {
		parts := splitKey(tt.key, tt.expectedParts)
		if len(parts) != len(tt.expected) {
			t.Errorf("splitKey(%q): expected %d parts, got %d", tt.key, len(tt.expected), len(parts))
			continue
		}
		for i, part := range parts {
			if part != tt.expected[i] {
				t.Errorf("splitKey(%q): part %d: expected %q, got %q", tt.key, i, tt.expected[i], part)
			}
		}
	}
}
