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

	"go.uber.org/zap"
)

func TestRepository_SetHidden(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	metaPath := filepath.Join(tmpDir, "metadata.json")

	initialConfig := "Host jump-box\n  HostName 10.0.0.1\n  User root\n\nHost web-prod\n  HostName 10.0.0.2\n  User admin\n"
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	logger := zap.NewNop().Sugar()
	repo := NewRepository(logger, configPath, metaPath)

	// Initially neither should be hidden
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	for _, s := range servers {
		if s.Hidden {
			t.Errorf("expected server %s not to be hidden initially", s.Alias)
		}
	}

	// Mark jump-box as hidden
	if err := repo.SetHidden("jump-box", true); err != nil {
		t.Fatalf("failed to set jump-box hidden: %v", err)
	}

	servers, err = repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range servers {
		if s.Alias == "jump-box" && !s.Hidden {
			t.Errorf("expected jump-box to be hidden")
		}
		if s.Alias == "web-prod" && s.Hidden {
			t.Errorf("expected web-prod not to be hidden")
		}
	}

	// Unhide jump-box
	if err := repo.SetHidden("jump-box", false); err != nil {
		t.Fatalf("failed to unhide jump-box: %v", err)
	}

	servers, err = repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range servers {
		if s.Alias == "jump-box" && s.Hidden {
			t.Errorf("expected jump-box to be unhidden")
		}
	}
}

func TestRepository_HiddenTagMapping(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	metaPath := filepath.Join(tmpDir, "metadata.json")

	// Host with # tags: hidden comment
	initialConfig := "Host internal-proxy # tags: hidden\n  HostName 10.1.1.1\n"
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	logger := zap.NewNop().Sugar()
	repo := NewRepository(logger, configPath, metaPath)

	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if !servers[0].Hidden {
		t.Errorf("expected internal-proxy with 'hidden' tag to be mapped as Hidden")
	}
}
