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

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeConn struct {
	connected bool
	calls     int
}

func (f *fakeConn) IsConnected() bool {
	f.calls++
	return f.connected
}

type fakeCol struct {
	collected bool
}

func (f *fakeCol) HasCollected() bool { return f.collected }

func TestReadyHandler(t *testing.T) {
	tests := []struct {
		name          string
		collected     bool
		connected     bool
		wantStatus    int
		wantBody      string
		wantConnCalls int
	}{
		{"not collected yet", false, true, http.StatusServiceUnavailable, "initial collection not completed", 0},
		{"collected, disconnected", true, false, http.StatusServiceUnavailable, "not connected to Teleport", 1},
		{"collected, connected", true, true, http.StatusOK, "ok", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeConn{connected: tt.connected}
			col := &fakeCol{collected: tt.collected}
			rec := httptest.NewRecorder()
			readyHandler(conn, col)(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.wantBody)
			}
			if conn.calls != tt.wantConnCalls {
				t.Errorf("IsConnected calls = %d, want %d", conn.calls, tt.wantConnCalls)
			}
		})
	}
}

func TestHealthHandler_AlwaysOK(t *testing.T) {
	rec := httptest.NewRecorder()
	healthHandler(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
}
