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
	"path/filepath"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"go.uber.org/zap"
)

func TestServerForm_DefaultIdentityKeyPrefilled(t *testing.T) {
	defaultKey := "~/.ssh/id_ed25519"
	form := NewServerForm(ServerFormAdd, nil).
		SetDefaultIdentityKey(defaultKey)

	defaults := form.getDefaultValues()
	if defaults.Key != defaultKey {
		t.Errorf("expected Key=%q in default values, got %q", defaultKey, defaults.Key)
	}

	// In edit mode with empty key, original server's empty key is preserved
	editServer := &domain.Server{Alias: "srv1", Host: "10.0.0.1"}
	editForm := NewServerForm(ServerFormEdit, editServer).
		SetDefaultIdentityKey(defaultKey)

	editDefaults := editForm.getDefaultValues()
	if editDefaults.Key != "" {
		t.Errorf("expected Key to be empty in edit mode, got %q", editDefaults.Key)
	}
}

func TestSettingsManager_DefaultIdentityKey(t *testing.T) {
	tempDir := t.TempDir()
	mgr := &settingsManager{
		logger:   zap.NewNop().Sugar(),
		filePath: filepath.Join(tempDir, "settings.json"),
	}

	// Initially empty
	key, err := mgr.LoadDefaultIdentityKey()
	if err != nil {
		t.Fatalf("unexpected error loading key: %v", err)
	}
	if key != "" {
		t.Errorf("expected empty key initially, got %q", key)
	}

	// Save and reload
	savedKey := "/custom/ssh/id_rsa"
	if err := mgr.SaveDefaultIdentityKey(savedKey); err != nil {
		t.Fatalf("unexpected error saving key: %v", err)
	}

	loadedKey, err := mgr.LoadDefaultIdentityKey()
	if err != nil {
		t.Fatalf("unexpected error reloading key: %v", err)
	}
	if loadedKey != savedKey {
		t.Errorf("expected %q, got %q", savedKey, loadedKey)
	}
}

func TestTUI_GetDefaultIdentityKey(t *testing.T) {
	// Case 1: defaultIdentityKey flag on tui
	t1 := &tui{
		defaultIdentityKey: "/path/from/flag",
	}
	if key := t1.getDefaultIdentityKey(); key != "/path/from/flag" {
		t.Errorf("expected '/path/from/flag', got %q", key)
	}

	// Case 2: fallback to settings manager
	tempDir := t.TempDir()
	mgr := &settingsManager{
		logger:   zap.NewNop().Sugar(),
		filePath: filepath.Join(tempDir, "settings.json"),
	}
	_ = mgr.SaveDefaultIdentityKey("/path/from/settings")
	t2 := &tui{
		settings: mgr,
	}
	if key := t2.getDefaultIdentityKey(); key != "/path/from/settings" {
		t.Errorf("expected '/path/from/settings', got %q", key)
	}
}
