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

	"go.uber.org/zap"
)

func TestSettingsManager_AutoPing(t *testing.T) {
	logger := zap.NewNop().Sugar()

	t.Run("nil manager", func(t *testing.T) {
		var sm *settingsManager
		enabled, interval, err := sm.LoadAutoPing()
		if err == nil {
			t.Errorf("expected error from nil settingsManager, got nil")
		}
		if enabled || interval != 60 {
			t.Errorf("expected false, 60, got %v, %v", enabled, interval)
		}
		if err := sm.SaveAutoPing(true, 30); err == nil {
			t.Errorf("expected error saving to nil settingsManager")
		}
	})

	t.Run("non-existent file defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm := &settingsManager{
			filePath: filepath.Join(tmpDir, "settings.json"),
			logger:   logger,
		}

		enabled, interval, err := sm.LoadAutoPing()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enabled {
			t.Errorf("expected AutoPing enabled=false by default, got true")
		}
		if interval != 60 {
			t.Errorf("expected default interval=60, got %d", interval)
		}
	})

	t.Run("save and reload auto ping settings", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm := &settingsManager{
			filePath: filepath.Join(tmpDir, "settings.json"),
			logger:   logger,
		}

		if err := sm.SaveAutoPing(true, 45); err != nil {
			t.Fatalf("failed to save auto ping: %v", err)
		}

		enabled, interval, err := sm.LoadAutoPing()
		if err != nil {
			t.Fatalf("failed to reload auto ping: %v", err)
		}
		if !enabled {
			t.Errorf("expected enabled=true, got false")
		}
		if interval != 45 {
			t.Errorf("expected interval=45, got %d", interval)
		}

		// Disable without overriding positive interval
		if err := sm.SaveAutoPing(false, 0); err != nil {
			t.Fatalf("failed to save disabled auto ping: %v", err)
		}
		enabled, interval, err = sm.LoadAutoPing()
		if err != nil {
			t.Fatalf("failed to reload auto ping: %v", err)
		}
		if enabled {
			t.Errorf("expected enabled=false, got true")
		}
		if interval != 45 {
			t.Errorf("expected interval to remain 45, got %d", interval)
		}
	})
}
