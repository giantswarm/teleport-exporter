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
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/giantswarm/teleport-exporter/internal/metrics"
	"github.com/giantswarm/teleport-exporter/internal/teleport"
)

// MockTeleportClient is a mock implementation for testing
type MockTeleportClient struct {
	ClusterName  string
	ClusterErr   error
	Nodes        []teleport.NodeInfo
	NodesErr     error
	KubeClusters []teleport.KubeClusterInfo
	KubeErr      error
	Databases    []teleport.DatabaseInfo
	DatabasesErr error
	Apps         []teleport.AppInfo
	AppsErr      error
}

func (m *MockTeleportClient) GetClusterName(ctx context.Context) (string, error) {
	return m.ClusterName, m.ClusterErr
}

func (m *MockTeleportClient) GetNodes(ctx context.Context) ([]teleport.NodeInfo, error) {
	return m.Nodes, m.NodesErr
}

func (m *MockTeleportClient) GetKubeClusters(ctx context.Context) ([]teleport.KubeClusterInfo, error) {
	return m.KubeClusters, m.KubeErr
}

func (m *MockTeleportClient) GetDatabases(ctx context.Context) ([]teleport.DatabaseInfo, error) {
	return m.Databases, m.DatabasesErr
}

func (m *MockTeleportClient) GetApps(ctx context.Context) ([]teleport.AppInfo, error) {
	return m.Apps, m.AppsErr
}

func TestCollector_UpdateNodeMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.NodeInfo.Reset()
	metrics.NodesTotal.Reset()

	c := &Collector{
		log: logr.Discard(),
	}

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
}

func TestCollector_UpdateKubeClusterMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.KubeClusterInfo.Reset()
	metrics.KubeClustersTotal.Reset()

	c := &Collector{
		log: logr.Discard(),
	}

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
}

func TestCollector_UpdateDatabaseMetrics(t *testing.T) {
	// Reset metrics before test
	metrics.DatabaseInfo.Reset()
	metrics.DatabasesTotal.Reset()

	c := &Collector{
		log: logr.Discard(),
	}

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

	c := &Collector{
		log: logr.Discard(),
	}

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
		RefreshInterval: 30 * time.Second,
		Log:             logr.Discard(),
	}

	c := New(cfg)

	if c.refreshInterval != 30*time.Second {
		t.Errorf("expected refreshInterval to be 30s, got %v", c.refreshInterval)
	}
}
