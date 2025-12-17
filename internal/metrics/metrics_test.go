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
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestTeleportUp(t *testing.T) {
	// Test that TeleportUp can be set
	TeleportUp.Set(1)
	value := testutil.ToFloat64(TeleportUp)
	if value != 1 {
		t.Errorf("expected TeleportUp to be 1, got %f", value)
	}

	TeleportUp.Set(0)
	value = testutil.ToFloat64(TeleportUp)
	if value != 0 {
		t.Errorf("expected TeleportUp to be 0, got %f", value)
	}
}

func TestClusterInfo(t *testing.T) {
	ClusterInfo.Reset()

	ClusterInfo.WithLabelValues("test-cluster").Set(1)
	value := testutil.ToFloat64(ClusterInfo.WithLabelValues("test-cluster"))
	if value != 1 {
		t.Errorf("expected ClusterInfo to be 1, got %f", value)
	}
}

func TestCollectErrorsTotal(t *testing.T) {
	// Test that error counter increments correctly
	initialValue := testutil.ToFloat64(CollectErrorsTotal.WithLabelValues("nodes"))

	CollectErrorsTotal.WithLabelValues("nodes").Inc()

	newValue := testutil.ToFloat64(CollectErrorsTotal.WithLabelValues("nodes"))
	if newValue != initialValue+1 {
		t.Errorf("expected CollectErrorsTotal to increment by 1, got %f", newValue-initialValue)
	}
}

func TestCollectDurationHistogram(t *testing.T) {
	// Test that histogram can observe values
	CollectDurationHistogram.Observe(0.5)
	CollectDurationHistogram.Observe(1.5)
	CollectDurationHistogram.Observe(5.0)

	// The histogram should have recorded these observations
	// We can't easily test the exact values, but we can verify it doesn't panic
}

func TestBuildInfo(t *testing.T) {
	BuildInfo.Reset()

	BuildInfo.WithLabelValues("1.0.0", "abc123", "2024-01-01", "go1.21.0").Set(1)
	value := testutil.ToFloat64(BuildInfo.WithLabelValues("1.0.0", "abc123", "2024-01-01", "go1.21.0"))
	if value != 1 {
		t.Errorf("expected BuildInfo to be 1, got %f", value)
	}
}

func TestLastSuccessfulCollectTime(t *testing.T) {
	testTimestamp := float64(1704067200) // 2024-01-01 00:00:00 UTC

	LastSuccessfulCollectTime.Set(testTimestamp)
	value := testutil.ToFloat64(LastSuccessfulCollectTime)
	if value != testTimestamp {
		t.Errorf("expected LastSuccessfulCollectTime to be %f, got %f", testTimestamp, value)
	}
}

func TestNodesTotal(t *testing.T) {
	NodesTotal.Reset()

	NodesTotal.WithLabelValues("cluster-1").Set(10)
	NodesTotal.WithLabelValues("cluster-2").Set(20)

	value1 := testutil.ToFloat64(NodesTotal.WithLabelValues("cluster-1"))
	if value1 != 10 {
		t.Errorf("expected NodesTotal for cluster-1 to be 10, got %f", value1)
	}

	value2 := testutil.ToFloat64(NodesTotal.WithLabelValues("cluster-2"))
	if value2 != 20 {
		t.Errorf("expected NodesTotal for cluster-2 to be 20, got %f", value2)
	}
}
