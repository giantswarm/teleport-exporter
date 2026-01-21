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
		log:             logr.Discard(),
		lastDbProtocols: make(map[string]struct{}),
		lastDbTypes:     make(map[string]struct{}),
	}
}

func TestCollector_UpdateNodeMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.NodesTotal.Reset()
	metrics.NodesIdentifiedTotal.Reset()
	metrics.NodesUnidentifiedTotal.Reset()

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

	// Verify identified count (both have cluster extracted from hostname)
	identifiedValue := testutil.ToFloat64(metrics.NodesIdentifiedTotal.WithLabelValues("test-cluster"))
	if identifiedValue != 2 {
		t.Errorf("expected NodesIdentifiedTotal to be 2, got %f", identifiedValue)
	}

	// Verify unidentified count
	unidentifiedValue := testutil.ToFloat64(metrics.NodesUnidentifiedTotal.WithLabelValues("test-cluster"))
	if unidentifiedValue != 0 {
		t.Errorf("expected NodesUnidentifiedTotal to be 0, got %f", unidentifiedValue)
	}
}

func TestCollector_UpdateNodeMetrics_IdentifiedVsUnidentified(t *testing.T) {
	// Reset metrics before test
	metrics.NodesTotal.Reset()
	metrics.NodesIdentifiedTotal.Reset()
	metrics.NodesUnidentifiedTotal.Reset()

	c := newTestCollector()

	// Mix of identified and unidentified nodes
	nodes := []teleport.NodeInfo{
		{Name: "node-1", Hostname: "host1.cluster-a.local", Labels: map[string]string{"giantswarm.io/cluster": "prod"}}, // identified via label
		{Name: "node-2", Hostname: "single-word-host"},                                                                  // unidentified (no dots, no labels)
		{Name: "node-3", Hostname: "host3.cluster-b.local"},                                                             // identified via hostname
	}

	c.updateNodeMetrics("test-cluster", nodes)

	// Verify counts
	totalValue := testutil.ToFloat64(metrics.NodesTotal.WithLabelValues("test-cluster"))
	if totalValue != 3 {
		t.Errorf("expected NodesTotal to be 3, got %f", totalValue)
	}

	identifiedValue := testutil.ToFloat64(metrics.NodesIdentifiedTotal.WithLabelValues("test-cluster"))
	if identifiedValue != 2 {
		t.Errorf("expected NodesIdentifiedTotal to be 2, got %f", identifiedValue)
	}

	unidentifiedValue := testutil.ToFloat64(metrics.NodesUnidentifiedTotal.WithLabelValues("test-cluster"))
	if unidentifiedValue != 1 {
		t.Errorf("expected NodesUnidentifiedTotal to be 1, got %f", unidentifiedValue)
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
	metrics.KubeClustersTotal.Reset()
	metrics.KubeManagementClustersTotal.Reset()
	metrics.KubeWorkloadClustersTotal.Reset()

	c := newTestCollector()

	// Mix of MC (no hyphen) and WC (has hyphen) clusters
	clusters := []teleport.KubeClusterInfo{
		{Name: "golem"},       // MC - no hyphen
		{Name: "guppy"},       // MC - no hyphen
		{Name: "golem-abc12"}, // WC - has hyphen
		{Name: "golem-xyz99"}, // WC - has hyphen
		{Name: "guppy-test1"}, // WC - has hyphen
	}

	c.updateKubeClusterMetrics("test-cluster", clusters)

	// Verify total count
	totalValue := testutil.ToFloat64(metrics.KubeClustersTotal.WithLabelValues("test-cluster"))
	if totalValue != 5 {
		t.Errorf("expected KubeClustersTotal to be 5, got %f", totalValue)
	}

	// Verify MC count (2 clusters without hyphen)
	mcValue := testutil.ToFloat64(metrics.KubeManagementClustersTotal.WithLabelValues("test-cluster"))
	if mcValue != 2 {
		t.Errorf("expected KubeManagementClustersTotal to be 2, got %f", mcValue)
	}

	// Verify WC count (3 clusters with hyphen)
	wcValue := testutil.ToFloat64(metrics.KubeWorkloadClustersTotal.WithLabelValues("test-cluster"))
	if wcValue != 3 {
		t.Errorf("expected KubeWorkloadClustersTotal to be 3, got %f", wcValue)
	}
}

func TestIsWorkloadCluster(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"golem", false},      // MC - no hyphen
		{"guppy", false},      // MC - no hyphen
		{"golem-abc12", true}, // WC - has hyphen
		{"my-cluster", true},  // WC - has hyphen
		{"a-b-c", true},       // WC - multiple hyphens
		{"", false},           // empty string
	}

	for _, tt := range tests {
		result := isWorkloadCluster(tt.name)
		if result != tt.expected {
			t.Errorf("isWorkloadCluster(%q) = %v, expected %v", tt.name, result, tt.expected)
		}
	}
}

func TestCollector_UpdateDatabaseMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.DatabasesTotal.Reset()
	metrics.DatabasesByProtocolTotal.Reset()
	metrics.DatabasesByTypeTotal.Reset()

	c := newTestCollector()

	databases := []teleport.DatabaseInfo{
		{Name: "postgres-db-1", Protocol: "postgres", Type: "rds"},
		{Name: "postgres-db-2", Protocol: "postgres", Type: "rds"},
		{Name: "mysql-db", Protocol: "mysql", Type: "self-hosted"},
		{Name: "mongo-db", Protocol: "mongodb", Type: "self-hosted"},
	}

	c.updateDatabaseMetrics("test-cluster", databases)

	// Verify total count
	totalValue := testutil.ToFloat64(metrics.DatabasesTotal.WithLabelValues("test-cluster"))
	if totalValue != 4 {
		t.Errorf("expected DatabasesTotal to be 4, got %f", totalValue)
	}

	// Verify by-protocol counts
	postgresCount := testutil.ToFloat64(metrics.DatabasesByProtocolTotal.WithLabelValues("test-cluster", "postgres"))
	if postgresCount != 2 {
		t.Errorf("expected DatabasesByProtocolTotal for postgres to be 2, got %f", postgresCount)
	}
	mysqlCount := testutil.ToFloat64(metrics.DatabasesByProtocolTotal.WithLabelValues("test-cluster", "mysql"))
	if mysqlCount != 1 {
		t.Errorf("expected DatabasesByProtocolTotal for mysql to be 1, got %f", mysqlCount)
	}

	// Verify by-type counts
	rdsCount := testutil.ToFloat64(metrics.DatabasesByTypeTotal.WithLabelValues("test-cluster", "rds"))
	if rdsCount != 2 {
		t.Errorf("expected DatabasesByTypeTotal for rds to be 2, got %f", rdsCount)
	}
	selfHostedCount := testutil.ToFloat64(metrics.DatabasesByTypeTotal.WithLabelValues("test-cluster", "self-hosted"))
	if selfHostedCount != 2 {
		t.Errorf("expected DatabasesByTypeTotal for self-hosted to be 2, got %f", selfHostedCount)
	}

	// Verify tracking maps
	if len(c.lastDbProtocols) != 3 {
		t.Errorf("expected lastDbProtocols to have 3 entries, got %d", len(c.lastDbProtocols))
	}
	if len(c.lastDbTypes) != 2 {
		t.Errorf("expected lastDbTypes to have 2 entries, got %d", len(c.lastDbTypes))
	}
}

func TestCollector_UpdateAppMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.AppsTotal.Reset()

	c := newTestCollector()

	apps := []teleport.AppInfo{
		{
			Name:       "grafana",
			PublicAddr: "grafana.example.com",
			URI:        "http://localhost:3000",
		},
		{
			Name:       "prometheus",
			PublicAddr: "prometheus.example.com",
			URI:        "http://localhost:9090",
		},
	}

	c.updateAppMetrics("test-cluster", apps)

	// Verify total count
	totalValue := testutil.ToFloat64(metrics.AppsTotal.WithLabelValues("test-cluster"))
	if totalValue != 2 {
		t.Errorf("expected AppsTotal to be 2, got %f", totalValue)
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
	if c.lastDbProtocols == nil {
		t.Error("expected lastDbProtocols to be initialized")
	}
	if c.lastDbTypes == nil {
		t.Error("expected lastDbTypes to be initialized")
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
