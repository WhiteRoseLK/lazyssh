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

package ssh_config_file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings_NonExistentFile(t *testing.T) {
	settings, err := LoadSettings("/nonexistent/path/metadata.json")
	if err != nil {
		t.Errorf("LoadSettings() with non-existent file should not error, got %v", err)
	}
	if settings.Theme != "" {
		t.Errorf("LoadSettings() with non-existent file should return empty theme, got %q", settings.Theme)
	}
}

func TestLoadSettings_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	if err := os.WriteFile(tmpFile, []byte{}, 0o600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	settings, err := LoadSettings(tmpFile)
	if err != nil {
		t.Errorf("LoadSettings() with empty file should not error, got %v", err)
	}
	if settings.Theme != "" {
		t.Errorf("LoadSettings() with empty file should return empty theme, got %q", settings.Theme)
	}
}

func TestLoadSettings_NewFormat(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	content := `{
		"settings": {"theme": "light"},
		"servers": {
			"srv1": {"tags": ["prod"]}
		}
	}`
	if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	settings, err := LoadSettings(tmpFile)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}
	if settings.Theme != "light" {
		t.Errorf("LoadSettings() theme = %q, want 'light'", settings.Theme)
	}
}

func TestLoadSettings_OldFormat(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	content := `{
		"srv1": {
			"tags": ["prod"],
			"last_seen": "2024-01-01T00:00:00Z"
		}
	}`
	if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	settings, err := LoadSettings(tmpFile)
	if err != nil {
		t.Fatalf("LoadSettings() with old format error: %v", err)
	}
	if settings.Theme != "" {
		t.Errorf("LoadSettings() with old format should return empty theme, got %q", settings.Theme)
	}

	mm := newMetadataManager(tmpFile, nil)
	servers, err := mm.loadAll()
	if err != nil {
		t.Fatalf("loadAll() error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if len(servers["srv1"].Tags) != 1 || servers["srv1"].Tags[0] != "prod" {
		t.Errorf("server tags mismatch: %v", servers["srv1"].Tags)
	}
}

func TestSaveSettings_PreservesServers(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	content := `{
		"servers": {
			"srv1": {"tags": ["web"]}
		}
	}`
	if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	mm := newMetadataManager(tmpFile, nil)
	if err := mm.SaveSettings(Settings{Theme: "dark"}); err != nil {
		t.Fatalf("SaveSettings() error: %v", err)
	}

	settings, err := mm.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error: %v", err)
	}
	if settings.Theme != "dark" {
		t.Errorf("theme = %q, want 'dark'", settings.Theme)
	}

	servers, err := mm.loadAll()
	if err != nil {
		t.Fatalf("loadAll() error: %v", err)
	}
	if len(servers) != 1 || len(servers["srv1"].Tags) != 1 || servers["srv1"].Tags[0] != "web" {
		t.Errorf("servers altered after SaveSettings: %v", servers)
	}
}
