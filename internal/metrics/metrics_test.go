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

func TestNodesTotal(t *testing.T) {
	NodesTotal.Reset()

	NodesTotal.WithLabelValues("test-cluster").Set(10)
	value := testutil.ToFloat64(NodesTotal.WithLabelValues("test-cluster"))
	if value != 10 {
		t.Errorf("expected NodesTotal to be 10, got %f", value)
	}
}

func TestCollectErrorsTotal(t *testing.T) {
	// Test that error counter increments correctly
	initialValue := testutil.ToFloat64(CollectErrorsTotal)

	CollectErrorsTotal.Inc()

	newValue := testutil.ToFloat64(CollectErrorsTotal)
	if newValue != initialValue+1 {
		t.Errorf("expected CollectErrorsTotal to increment by 1, got %f", newValue-initialValue)
	}
}

func TestCollectDuration(t *testing.T) {
	// Test that gauge can be set
	CollectDuration.Set(0.5)
	value := testutil.ToFloat64(CollectDuration)
	if value != 0.5 {
		t.Errorf("expected CollectDuration to be 0.5, got %f", value)
	}

	CollectDuration.Set(1.5)
	value = testutil.ToFloat64(CollectDuration)
	if value != 1.5 {
		t.Errorf("expected CollectDuration to be 1.5, got %f", value)
	}
}

func TestKubeClustersTotal(t *testing.T) {
	KubeClustersTotal.Reset()

	KubeClustersTotal.WithLabelValues("test-cluster").Set(5)
	value := testutil.ToFloat64(KubeClustersTotal.WithLabelValues("test-cluster"))
	if value != 5 {
		t.Errorf("expected KubeClustersTotal to be 5, got %f", value)
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

func TestDatabasesTotal(t *testing.T) {
	DatabasesTotal.Reset()

	DatabasesTotal.WithLabelValues("test-cluster").Set(3)
	value := testutil.ToFloat64(DatabasesTotal.WithLabelValues("test-cluster"))
	if value != 3 {
		t.Errorf("expected DatabasesTotal to be 3, got %f", value)
	}
}

func TestAppsTotal(t *testing.T) {
	AppsTotal.Reset()

	AppsTotal.WithLabelValues("test-cluster").Set(7)
	value := testutil.ToFloat64(AppsTotal.WithLabelValues("test-cluster"))
	if value != 7 {
		t.Errorf("expected AppsTotal to be 7, got %f", value)
	}
}
