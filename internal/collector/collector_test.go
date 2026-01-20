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
		log:                    logr.Discard(),
		lastNodesByKubeCluster: make(map[string]struct{}),
		lastKubeClusters:       make(map[string]struct{}),
		lastDatabases:          make(map[string]struct{}),
		lastApps:               make(map[string]struct{}),
	}
}

func TestCollector_UpdateNodeMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.NodesByKubeClusterTotal.Reset()

	c := newTestCollector()

	nodes := []teleport.NodeInfo{
		{
			Name:      "node-1",
			Hostname:  "host1.mycluster.example.com",
			Address:   "192.168.1.1:3022",
			Namespace: "default",
			SubKind:   "teleport",
		},
		{
			Name:      "node-2",
			Hostname:  "host2.mycluster.example.com",
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

	// Verify nodes by kubernetes cluster
	// kube_cluster is extracted from hostname: host1.mycluster.example.com -> "mycluster"
	kubeClusterValue := testutil.ToFloat64(metrics.NodesByKubeClusterTotal.WithLabelValues("test-cluster", "mycluster"))
	if kubeClusterValue != 2 {
		t.Errorf("expected NodesByKubeClusterTotal for mycluster to be 2, got %f", kubeClusterValue)
	}

	// Verify that lastNodesByKubeCluster has 1 entry
	if len(c.lastNodesByKubeCluster) != 1 {
		t.Errorf("expected lastNodesByKubeCluster to have 1 entry, got %d", len(c.lastNodesByKubeCluster))
	}
}

func TestCollector_UpdateNodeMetrics_RemovesStaleKubeClusters(t *testing.T) {
	// Reset metrics before test
	metrics.NodesByKubeClusterTotal.Reset()

	c := newTestCollector()

	// First update with nodes in 2 different kube clusters
	nodes := []teleport.NodeInfo{
		{Name: "node-1", Hostname: "host1.cluster-a.local", Address: "1.1.1.1", Namespace: "default", SubKind: "teleport"},
		{Name: "node-2", Hostname: "host2.cluster-b.local", Address: "2.2.2.2", Namespace: "default", SubKind: "teleport"},
	}
	c.updateNodeMetrics("test-cluster", nodes)

	// Verify 2 kube clusters tracked
	if len(c.lastNodesByKubeCluster) != 2 {
		t.Errorf("expected 2 kube clusters after first update, got %d", len(c.lastNodesByKubeCluster))
	}

	// Second update with only cluster-a (cluster-b node removed)
	nodes = []teleport.NodeInfo{
		{Name: "node-1", Hostname: "host1.cluster-a.local", Address: "1.1.1.1", Namespace: "default", SubKind: "teleport"},
	}
	c.updateNodeMetrics("test-cluster", nodes)

	// Verify only 1 kube cluster tracked
	if len(c.lastNodesByKubeCluster) != 1 {
		t.Errorf("expected 1 kube cluster after removal, got %d", len(c.lastNodesByKubeCluster))
	}
}

func TestCollector_UpdateNodeMetrics_WithLabels(t *testing.T) {
	// Reset metrics before test
	metrics.NodesByKubeClusterTotal.Reset()

	c := newTestCollector()

	// Test nodes with cluster labels (labels take precedence over hostname)
	nodes := []teleport.NodeInfo{
		{
			Name:      "node-1",
			Hostname:  "host1.default.local",
			Address:   "1.1.1.1",
			Namespace: "default",
			SubKind:   "teleport",
			Labels:    map[string]string{"giantswarm.io/cluster": "prod-cluster"},
		},
		{
			Name:      "node-2",
			Hostname:  "host2.default.local",
			Address:   "2.2.2.2",
			Namespace: "default",
			SubKind:   "teleport",
			Labels:    map[string]string{"giantswarm.io/cluster": "prod-cluster"},
		},
		{
			Name:      "node-3",
			Hostname:  "host3.default.local",
			Address:   "3.3.3.3",
			Namespace: "default",
			SubKind:   "teleport",
			Labels:    map[string]string{"giantswarm.io/cluster": "dev-cluster"},
		},
	}

	c.updateNodeMetrics("test-cluster", nodes)

	// Verify nodes by kubernetes cluster (should use labels)
	prodValue := testutil.ToFloat64(metrics.NodesByKubeClusterTotal.WithLabelValues("test-cluster", "prod-cluster"))
	if prodValue != 2 {
		t.Errorf("expected NodesByKubeClusterTotal for prod-cluster to be 2, got %f", prodValue)
	}

	devValue := testutil.ToFloat64(metrics.NodesByKubeClusterTotal.WithLabelValues("test-cluster", "dev-cluster"))
	if devValue != 1 {
		t.Errorf("expected NodesByKubeClusterTotal for dev-cluster to be 1, got %f", devValue)
	}

	// Verify 2 kube clusters tracked
	if len(c.lastNodesByKubeCluster) != 2 {
		t.Errorf("expected lastNodesByKubeCluster to have 2 entries, got %d", len(c.lastNodesByKubeCluster))
	}
}

func TestExtractKubeCluster(t *testing.T) {
	tests := []struct {
		name     string
		node     teleport.NodeInfo
		expected string
	}{
		{
			name: "from giantswarm.io/cluster label",
			node: teleport.NodeInfo{
				Hostname: "host1.local",
				Labels:   map[string]string{"giantswarm.io/cluster": "my-cluster"},
			},
			expected: "my-cluster",
		},
		{
			name: "from cluster label",
			node: teleport.NodeInfo{
				Hostname: "host1.local",
				Labels:   map[string]string{"cluster": "another-cluster"},
			},
			expected: "another-cluster",
		},
		{
			name: "from hostname pattern",
			node: teleport.NodeInfo{
				Hostname: "ip-10-0-1-5.us-west-2.compute.internal",
				Labels:   map[string]string{},
			},
			expected: "us-west-2",
		},
		{
			name: "from simple hostname",
			node: teleport.NodeInfo{
				Hostname: "node-1.mycluster.local",
				Labels:   map[string]string{},
			},
			expected: "mycluster",
		},
		{
			name: "unknown when no pattern matches",
			node: teleport.NodeInfo{
				Hostname: "single-word-host",
				Labels:   map[string]string{},
			},
			expected: "unknown",
		},
		{
			name: "label takes precedence over hostname",
			node: teleport.NodeInfo{
				Hostname: "node-1.hostname-cluster.local",
				Labels:   map[string]string{"giantswarm.io/cluster": "label-cluster"},
			},
			expected: "label-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractKubeCluster(tt.node)
			if result != tt.expected {
				t.Errorf("extractKubeCluster() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestSplitByDot(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a.b.c", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"host.example.com", []string{"host", "example", "com"}},
		{"", []string{""}},
	}

	for _, tt := range tests {
		result := splitByDot(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitByDot(%q): expected %d parts, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for i, part := range result {
			if part != tt.expected[i] {
				t.Errorf("splitByDot(%q): part %d: expected %q, got %q", tt.input, i, tt.expected[i], part)
			}
		}
	}
}

func TestCollector_UpdateKubeClusterMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.KubeClusterInfo.Reset()

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

	// Verify individual cluster info (now with cluster_name)
	clusterInfoValue := testutil.ToFloat64(metrics.KubeClusterInfo.WithLabelValues("test-cluster", "kube-cluster-1"))
	if clusterInfoValue != 1 {
		t.Errorf("expected KubeClusterInfo for kube-cluster-1 to be 1, got %f", clusterInfoValue)
	}

	// Verify tracking map
	if len(c.lastKubeClusters) != 3 {
		t.Errorf("expected lastKubeClusters to have 3 entries, got %d", len(c.lastKubeClusters))
	}
}

func TestCollector_UpdateDatabaseMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.DatabaseInfo.Reset()

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

	// Verify database info by protocol/type (now aggregated with cluster_name)
	dbInfoValue := testutil.ToFloat64(metrics.DatabaseInfo.WithLabelValues("test-cluster", "postgres", "rds"))
	if dbInfoValue != 1 {
		t.Errorf("expected DatabaseInfo for postgres/rds to be 1, got %f", dbInfoValue)
	}
}

func TestCollector_UpdateAppMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.AppInfo.Reset()

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

	// Verify app info (now with cluster_name, app_name)
	appInfoValue := testutil.ToFloat64(metrics.AppInfo.WithLabelValues("test-cluster", "grafana"))
	if appInfoValue != 1 {
		t.Errorf("expected AppInfo for grafana to be 1, got %f", appInfoValue)
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
	if c.lastNodesByKubeCluster == nil {
		t.Error("expected lastNodesByKubeCluster to be initialized")
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
