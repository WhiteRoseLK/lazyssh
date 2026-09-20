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
	"strings"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"go.uber.org/zap"
)

func TestAddServer_ConvertsAbsoluteIdentityFileToTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		t.Skip("skipping test: UserHomeDir unavailable")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	metaPath := filepath.Join(dir, "metadata.json")
	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metaPath)
	absKey := filepath.Join(homeDir, ".ssh", "id_ed25519")

	server := domain.Server{
		Alias:              "test-server",
		Host:               "192.168.1.50",
		User:               "deploy",
		Port:               22,
		IdentityFiles:      []string{absKey},
		UserKnownHostsFile: filepath.Join(homeDir, ".ssh", "known_hosts"),
	}

	if err := repo.AddServer(server); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "IdentityFile ~/.ssh/id_ed25519") {
		t.Errorf("expected config to contain 'IdentityFile ~/.ssh/id_ed25519', got:\n%s", strContent)
	}
	if strings.Contains(strContent, absKey) {
		t.Errorf("config should not contain absolute key path %s, got:\n%s", absKey, strContent)
	}
	if !strings.Contains(strContent, "UserKnownHostsFile ~/.ssh/known_hosts") {
		t.Errorf("expected config to contain 'UserKnownHostsFile ~/.ssh/known_hosts', got:\n%s", strContent)
	}
}

func TestUpdateServer_ConvertsAbsoluteIdentityFileToTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		t.Skip("skipping test: UserHomeDir unavailable")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	metaPath := filepath.Join(dir, "metadata.json")
	initialConfig := `Host myserver
    HostName 10.0.0.1
    User admin
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metaPath)
	oldServer := domain.Server{
		Alias: "myserver",
		Host:  "10.0.0.1",
		User:  "admin",
	}

	absKey := filepath.Join(homeDir, ".ssh", "id_custom")
	newServer := oldServer
	newServer.IdentityFiles = []string{absKey}

	if err := repo.UpdateServer(oldServer, newServer); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "IdentityFile ~/.ssh/id_custom") {
		t.Errorf("expected config to contain 'IdentityFile ~/.ssh/id_custom', got:\n%s", strContent)
	}
	if strings.Contains(strContent, absKey) {
		t.Errorf("config should not contain absolute key path %s, got:\n%s", absKey, strContent)
	}
}

func TestListServers_MapsAbsoluteIdentityFileToTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		t.Skip("skipping test: UserHomeDir unavailable")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	metaPath := filepath.Join(dir, "metadata.json")
	absKey := filepath.Join(homeDir, ".ssh", "id_rsa")
	rawConfig := "Host portable\n    HostName 10.0.0.2\n    IdentityFile " + absKey + "\n"

	if err := os.WriteFile(configPath, []byte(rawConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metaPath)
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	if len(servers[0].IdentityFiles) != 1 || servers[0].IdentityFiles[0] != "~/.ssh/id_rsa" {
		t.Errorf("expected IdentityFile '~/.ssh/id_rsa', got %v", servers[0].IdentityFiles)
	}
}
