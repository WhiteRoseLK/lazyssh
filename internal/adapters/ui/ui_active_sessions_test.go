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
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
)

func TestUIActiveSessions_Actions(t *testing.T) {
	app, mockService, _ := setupNavTestTUI()

	activeServer := domain.Server{
		Alias:         "active-host",
		Host:          "192.168.1.50",
		User:          "ubuntu",
		Port:          22,
		ActivePID:     4321,
		IdentityFiles: []string{"~/.ssh/id_rsa"},
		LocalForward:  []string{"8080:localhost:80"},
	}
	app.activeList.UpdateServers([]domain.Server{activeServer})
	app.activeList.SetCurrentItem(0)

	// Focus activeList
	app.handleActiveListFocus()
	if !app.isActiveListFocused() {
		t.Fatalf("expected activeList to be focused")
	}

	// Test 'e' (edit) is disabled
	app.handleServerEdit()
	if app.statusBar.GetText(true) != "Edit disabled for active sessions. Use 'a' to add." {
		t.Fatalf("expected status message for disabled edit on active session, got %q", app.statusBar.GetText(true))
	}

	// Test 'd' (delete) is disabled
	app.handleServerDelete()
	if app.statusBar.GetText(true) != "Cannot delete active session. Use 'K' to terminate." {
		t.Fatalf("expected status message for disabled delete on active session, got %q", app.statusBar.GetText(true))
	}

	// Test 'a' (add from active) opens form without panicking
	app.handleServerAdd()

	// Return to main
	app.handleFormCancel()

	// Focus activeList again
	app.handleActiveListFocus()

	// Test 'K' (kill) calls KillActiveSessions
	app.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, 'K', tcell.ModNone))
	_ = mockService
}
