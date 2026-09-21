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
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

func TestTUI_FilterServersForDisplay(t *testing.T) {
	logger := zap.NewNop().Sugar()
	servers := []domain.Server{
		{Alias: "srv-visible", Host: "10.0.0.1", Hidden: false},
		{Alias: "srv-hidden", Host: "10.0.0.2", Hidden: true},
	}
	svc := &mockServerService{servers: servers}

	// Case 1: Default showHidden = false
	appDef := NewTUI(logger, svc, "v1.0.0", "abc1234", Config{ShowHidden: false}).(*tui)
	displayed := appDef.filterServersForDisplay(servers)
	if len(displayed) != 1 || displayed[0].Alias != "srv-visible" {
		t.Fatalf("expected only srv-visible when showHidden=false, got %v", displayed)
	}

	// Case 2: showHidden = true
	appShow := NewTUI(logger, svc, "v1.0.0", "abc1234", Config{ShowHidden: true}).(*tui)
	displayed = appShow.filterServersForDisplay(servers)
	if len(displayed) != 2 {
		t.Fatalf("expected all 2 servers when showHidden=true, got %d", len(displayed))
	}
}

func TestTUI_ToggleShowHiddenKey(t *testing.T) {
	logger := zap.NewNop().Sugar()
	servers := []domain.Server{
		{Alias: "srv1", Host: "10.0.0.1", Hidden: false},
		{Alias: "srv2", Host: "10.0.0.2", Hidden: true},
	}
	svc := &mockServerService{servers: servers}

	appInstance := tview.NewApplication()
	uiApp := NewTUI(logger, svc, "v1.0.0", "abc1234", Config{ShowHidden: false}).(*tui)
	uiApp.app = appInstance

	uiApp.buildComponents()
	uiApp.buildLayout()
	uiApp.bindEvents()
	uiApp.loadInitialData()

	if uiApp.serverList.GetItemCount() != 1 {
		t.Fatalf("expected 1 item initially in list, got %d", uiApp.serverList.GetItemCount())
	}

	// Press 'H' to reveal hidden servers
	eventH := tcell.NewEventKey(tcell.KeyRune, 'H', tcell.ModNone)
	res := uiApp.handleGlobalKeys(eventH)
	if res != nil {
		t.Fatalf("expected 'H' event to be consumed, got %v", res)
	}

	if !uiApp.ShowHidden() {
		t.Errorf("expected ShowHidden to be true after pressing 'H'")
	}
	if uiApp.serverList.GetItemCount() != 2 {
		t.Errorf("expected 2 items in list after revealing, got %d", uiApp.serverList.GetItemCount())
	}
	if !strings.Contains(uiApp.serverList.GetTitle(), "[SHOW HIDDEN]") {
		t.Errorf("expected title to contain [SHOW HIDDEN], got %q", uiApp.serverList.GetTitle())
	}

	// Press 'H' again to hide
	_ = uiApp.handleGlobalKeys(eventH)
	if uiApp.ShowHidden() {
		t.Errorf("expected ShowHidden to be false after pressing 'H' again")
	}
	if uiApp.serverList.GetItemCount() != 1 {
		t.Errorf("expected 1 item in list after toggling off, got %d", uiApp.serverList.GetItemCount())
	}
}

func TestTUI_ToggleServerHiddenKey(t *testing.T) {
	logger := zap.NewNop().Sugar()
	servers := []domain.Server{
		{Alias: "srv1", Host: "10.0.0.1", Hidden: false},
	}
	svc := &mockServerService{servers: servers}

	appInstance := tview.NewApplication()
	uiApp := NewTUI(logger, svc, "v1.0.0", "abc1234", Config{ShowHidden: false}).(*tui)
	uiApp.app = appInstance

	uiApp.buildComponents()
	uiApp.buildLayout()
	uiApp.bindEvents()
	uiApp.loadInitialData()

	// Press 'm' to mark srv1 as hidden
	eventM := tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModNone)
	res := uiApp.handleGlobalKeys(eventM)
	if res != nil {
		t.Fatalf("expected 'm' event to be consumed, got %v", res)
	}

	if !svc.servers[0].Hidden {
		t.Fatalf("expected srv1 to be marked hidden in service")
	}

	// Since showHidden is false, srv1 should now disappear from view
	if uiApp.serverList.GetItemCount() != 0 {
		t.Errorf("expected 0 items in list after hiding selected server, got %d", uiApp.serverList.GetItemCount())
	}
}

func TestTUI_ReadOnlyBlocksHideKey(t *testing.T) {
	logger := zap.NewNop().Sugar()
	servers := []domain.Server{
		{Alias: "srv1", Host: "10.0.0.1", Hidden: false},
	}
	svc := &mockServerService{servers: servers}

	appInstance := tview.NewApplication()
	uiApp := NewTUI(logger, svc, "v1.0.0", "abc1234", Config{ReadOnly: true}).(*tui)
	uiApp.app = appInstance

	uiApp.buildComponents()
	uiApp.buildLayout()
	uiApp.bindEvents()
	uiApp.loadInitialData()

	// Press 'm' in read-only mode should be intercepted by read-only modal
	eventM := tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModNone)
	res := uiApp.handleGlobalKeys(eventM)
	if res != nil {
		t.Fatalf("expected 'm' event to be consumed by readonly check, got %v", res)
	}

	// srv1 should still not be hidden
	if svc.servers[0].Hidden {
		t.Fatalf("srv1 should not be marked hidden in read-only mode")
	}
}

func TestFormatServerLine_HiddenBadge(t *testing.T) {
	srv := domain.Server{
		Alias:  "hidden-srv",
		Host:   "1.2.3.4",
		Hidden: true,
	}

	primary, _ := formatServerLine(srv, 15, 80)
	if !strings.Contains(primary, "[hidden]") {
		t.Errorf("expected primary text to contain '[hidden]' badge, got %q", primary)
	}
}
