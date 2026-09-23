// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

type mockPingService struct {
	mockServerService
	mu          sync.Mutex
	pingedHosts []string
}

func (m *mockPingService) Ping(server domain.Server) (bool, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingedHosts = append(m.pingedHosts, server.Alias)
	return true, 25 * time.Millisecond, nil
}

func (m *mockPingService) UpdateServerPing(alias, status string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.servers {
		if m.servers[i].Alias == alias {
			m.servers[i].PingStatus = status
			m.servers[i].PingLatency = latency
			break
		}
	}
}

func TestAutoPing_StartStop(t *testing.T) {
	logger := zap.NewNop().Sugar()
	svc := &mockPingService{}
	app := NewTUI(logger, svc, "v1.0.0", "abc1234").(*tui)
	app.statusBar = tview.NewTextView().SetDynamicColors(true)

	// Initially disabled
	if app.autoPingEnabled {
		t.Errorf("expected autoPingEnabled to be false initially")
	}
	initialStatus := app.defaultStatusText()
	if strings.Contains(initialStatus, "WATCH") {
		t.Errorf("expected status text without WATCH, got: %s", initialStatus)
	}

	// Start auto-ping
	app.startAutoPing()
	if !app.autoPingEnabled {
		t.Errorf("expected autoPingEnabled to be true after start")
	}
	if app.autoPingStop == nil {
		t.Errorf("expected autoPingStop channel to be initialized")
	}

	watchStatus := app.defaultStatusText()
	if !strings.Contains(watchStatus, "WATCH") {
		t.Errorf("expected status text with WATCH badge, got: %s", watchStatus)
	}

	// Stop auto-ping
	app.stopAutoPing()
	if app.autoPingEnabled {
		t.Errorf("expected autoPingEnabled to be false after stop")
	}
	if app.autoPingStop != nil {
		t.Errorf("expected autoPingStop to be nil after stop")
	}

	stoppedStatus := app.defaultStatusText()
	if strings.Contains(stoppedStatus, "WATCH") {
		t.Errorf("expected status text without WATCH after stop, got: %s", stoppedStatus)
	}
}

func TestAutoPing_Toggle(t *testing.T) {
	logger := zap.NewNop().Sugar()
	svc := &mockPingService{}
	app := NewTUI(logger, svc, "v1.0.0", "abc1234").(*tui)
	app.statusBar = tview.NewTextView().SetDynamicColors(true)

	// Toggle ON
	app.handleToggleAutoPing()
	if !app.autoPingEnabled {
		t.Errorf("expected autoPingEnabled=true after first toggle")
	}

	// Toggle OFF
	app.handleToggleAutoPing()
	if app.autoPingEnabled {
		t.Errorf("expected autoPingEnabled=false after second toggle")
	}
}

func TestAutoPing_ConfigInitialization(t *testing.T) {
	logger := zap.NewNop().Sugar()
	svc := &mockPingService{}
	app := NewTUI(logger, svc, "v1.0.0", "abc1234", Config{
		AutoPing:         true,
		AutoPingSet:      true,
		AutoPingInterval: 30,
	}).(*tui)
	defer app.stopAutoPing()

	if !app.autoPingEnabled {
		t.Errorf("expected autoPingEnabled=true from config")
	}
	if app.autoPingInterval != 30*time.Second {
		t.Errorf("expected autoPingInterval=30s, got %v", app.autoPingInterval)
	}
	if app.autoPingSecondsRemaining != 30 {
		t.Errorf("expected autoPingSecondsRemaining=30, got %d", app.autoPingSecondsRemaining)
	}
}

func TestAutoPing_CountdownStatusText(t *testing.T) {
	logger := zap.NewNop().Sugar()
	svc := &mockPingService{}
	app := NewTUI(logger, svc, "v1.0.0", "abc1234").(*tui)

	text := app.defaultStatusTextWithCountdown(17)
	if !strings.Contains(text, "WATCH 17s") {
		t.Errorf("expected text to contain 'WATCH 17s', got %s", text)
	}
}

func TestAutoPing_ExecuteBackgroundPingSweep(t *testing.T) {
	logger := zap.NewNop().Sugar()
	servers := []domain.Server{
		{Alias: "srv-1", Host: "10.0.0.1"},
		{Alias: "srv-2", Host: "10.0.0.2"},
		{Alias: "*.wildcard", Host: "*.corp", IsWildcard: true},
	}
	svc := &mockPingService{
		mockServerService: mockServerService{
			servers: servers,
		},
	}

	app := NewTUI(logger, svc, "v1.0.0", "abc1234").(*tui)
	app.serverList = NewServerList()
	app.serverList.UpdateServers(servers)

	app.executeBackgroundPingSweep()

	// Wait briefly for background ping goroutines
	time.Sleep(50 * time.Millisecond)

	svc.mu.Lock()
	pinged := make([]string, len(svc.pingedHosts))
	copy(pinged, svc.pingedHosts)
	svc.mu.Unlock()

	if len(pinged) != 2 { // wildcard server excluded
		t.Errorf("expected 2 pinged hosts, got %d: %v", len(pinged), pinged)
	}
}
