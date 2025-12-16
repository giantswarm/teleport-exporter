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

package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	// Check that GoVersion is populated correctly
	if info.GoVersion != runtime.Version() {
		t.Errorf("expected GoVersion to be %s, got %s", runtime.Version(), info.GoVersion)
	}

	// Check that Platform is in correct format
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("expected Platform to be %s, got %s", expectedPlatform, info.Platform)
	}

	// Check that default values are set when not overridden by ldflags
	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.Commit == "" {
		t.Error("Commit should not be empty")
	}

	if info.BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}

func TestInfo_DefaultValues(t *testing.T) {
	// Test that the default values are sensible
	if Version != "dev" {
		// Only check this if ldflags weren't used
		if !strings.Contains(Version, ".") && Version != "dev" {
			t.Logf("Version has unexpected default value: %s", Version)
		}
	}

	if Commit == "" {
		t.Error("Commit should have a default value")
	}

	if BuildDate == "" {
		t.Error("BuildDate should have a default value")
	}
}
