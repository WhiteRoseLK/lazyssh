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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/WhiteRoseLK/neossh/internal/core/ports"
)

func TestRootCmd_ExitOnDisconnectFlags(t *testing.T) {
	tests := []struct {
		name                  string
		args                  []string
		expectedExitOnDisc    bool
		expectedSSHConfigFile string
	}{
		{
			name:                  "default is false and empty config",
			args:                  []string{},
			expectedExitOnDisc:    false,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag --exit-on-disconnect",
			args:                  []string{"--exit-on-disconnect"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag shorthand -x",
			args:                  []string{"-x"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag --auto-exit",
			args:                  []string{"--auto-exit"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag -x combined with --sshconfig",
			args:                  []string{"--sshconfig", "/custom/ssh/config", "-x"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "/custom/ssh/config",
		},
		{
			name:                  "flag --exit-on-disconnect combined with --sshconfig",
			args:                  []string{"--exit-on-disconnect", "--sshconfig", "/custom/ssh/config"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "/custom/ssh/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables before each test case
			exitOnDisconnect = false
			sshConfigFile = ""

			cmd := newRootCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("unexpected error parsing flags %v: %v", tt.args, err)
			}

			if exitOnDisconnect != tt.expectedExitOnDisc {
				t.Errorf("exitOnDisconnect = %v, want %v", exitOnDisconnect, tt.expectedExitOnDisc)
			}

			if sshConfigFile != tt.expectedSSHConfigFile {
				t.Errorf("sshConfigFile = %q, want %q", sshConfigFile, tt.expectedSSHConfigFile)
			}
		})
	}
}

func TestRootCmd_FlagsExist(t *testing.T) {
	cmd := newRootCmd()

	exitFlag := cmd.PersistentFlags().Lookup("exit-on-disconnect")
	if exitFlag == nil {
		t.Fatal("expected persistent flag --exit-on-disconnect to exist")
	}
	if exitFlag.Shorthand != "x" {
		t.Errorf("expected shorthand 'x', got %q", exitFlag.Shorthand)
	}

	roFlag := cmd.PersistentFlags().Lookup("ssh-config-readonly")
	if roFlag == nil {
		t.Fatal("expected persistent flag --ssh-config-readonly to exist")
	}

	roAlias := cmd.PersistentFlags().Lookup("readonly")
	if roAlias == nil {
		t.Fatal("expected persistent flag --readonly to exist")
	}
	if roAlias.Shorthand != "r" {
		t.Errorf("expected shorthand 'r', got %q", roAlias.Shorthand)
	}
}

func TestRootCmd_ReadOnlyFlags(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedReadOnly bool
	}{
		{
			name:             "default without readonly",
			args:             []string{},
			expectedReadOnly: false,
		},
		{
			name:             "flag --ssh-config-readonly",
			args:             []string{"--ssh-config-readonly"},
			expectedReadOnly: true,
		},
		{
			name:             "flag --readonly",
			args:             []string{"--readonly"},
			expectedReadOnly: true,
		},
		{
			name:             "flag -r",
			args:             []string{"-r"},
			expectedReadOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sshConfigReadonly = false
			cmd := newRootCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("unexpected error parsing flags %v: %v", tt.args, err)
			}
			if sshConfigReadonly != tt.expectedReadOnly {
				t.Errorf("sshConfigReadonly = %v, want %v", sshConfigReadonly, tt.expectedReadOnly)
			}
		})
	}
}

func TestRootCmd_FilterAndConnectFlags(t *testing.T) {
	cmd := newRootCmd()

	filterFlag := cmd.PersistentFlags().Lookup("filter")
	if filterFlag == nil {
		t.Fatal("expected persistent flag --filter to exist")
	}
	if filterFlag.Shorthand != "f" {
		t.Errorf("expected shorthand 'f', got %q", filterFlag.Shorthand)
	}

	connectFlag := cmd.PersistentFlags().Lookup("connect")
	if connectFlag == nil {
		t.Fatal("expected persistent flag --connect to exist")
	}
	if connectFlag.Shorthand != "c" {
		t.Errorf("expected shorthand 'c', got %q", connectFlag.Shorthand)
	}
}

func TestRootCmd_FilterParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedFilter  string
		expectedConnect bool
	}{
		{
			name:            "default no filter",
			args:            []string{},
			expectedFilter:  "",
			expectedConnect: false,
		},
		{
			name:            "flag --filter",
			args:            []string{"--filter", "prod-web"},
			expectedFilter:  "prod-web",
			expectedConnect: false,
		},
		{
			name:            "flag -f",
			args:            []string{"-f", "staging"},
			expectedFilter:  "staging",
			expectedConnect: false,
		},
		{
			name:            "flag --connect",
			args:            []string{"--connect"},
			expectedFilter:  "",
			expectedConnect: true,
		},
		{
			name:            "flag -c with -f",
			args:            []string{"-c", "-f", "my-server"},
			expectedFilter:  "my-server",
			expectedConnect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterQuery = ""
			connectDirectly = false
			cmd := newRootCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("unexpected error parsing flags %v: %v", tt.args, err)
			}
			if filterQuery != tt.expectedFilter {
				t.Errorf("filterQuery = %q, want %q", filterQuery, tt.expectedFilter)
			}
			if connectDirectly != tt.expectedConnect {
				t.Errorf("connectDirectly = %v, want %v", connectDirectly, tt.expectedConnect)
			}
		})
	}
}

type mockDirectConnectService struct {
	ports.ServerService
	servers    []domain.Server
	sshCalled  string
	defaultKey string
}

func (m *mockDirectConnectService) ListServers(query string) ([]domain.Server, error) {
	var res []domain.Server
	for _, s := range m.servers {
		if strings.Contains(s.Alias, query) {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *mockDirectConnectService) SSH(alias string) error {
	m.sshCalled = alias
	return nil
}

func (m *mockDirectConnectService) GetDefaultIdentityKey() (string, error) {
	return m.defaultKey, nil
}

func (m *mockDirectConnectService) SaveDefaultIdentityKey(key string) error {
	m.defaultKey = key
	return nil
}

func TestHandleDirectConnect(t *testing.T) {
	svc := &mockDirectConnectService{
		servers: []domain.Server{
			{Alias: "web-prod", Host: "10.0.0.1"},
			{Alias: "web-staging", Host: "10.0.0.2"},
			{Alias: "*.corp", Host: "*.corp", IsWildcard: true},
		},
	}

	// Empty filter
	connected, err := handleDirectConnect("", svc)
	if err == nil || connected {
		t.Errorf("expected error for empty filter, got connected=%v, err=%v", connected, err)
	}

	// Non-existent server
	connected, err = handleDirectConnect("database", svc)
	if err == nil || connected {
		t.Errorf("expected error for non-existent server, got connected=%v, err=%v", connected, err)
	}

	// Exact match
	connected, err = handleDirectConnect("web-prod", svc)
	if err != nil || !connected {
		t.Fatalf("expected successful connect to web-prod, got connected=%v, err=%v", connected, err)
	}
	if svc.sshCalled != "web-prod" {
		t.Errorf("expected SSH called with 'web-prod', got %q", svc.sshCalled)
	}

	// Wildcard pattern match
	connected, err = handleDirectConnect("*.corp", svc)
	if err == nil || connected {
		t.Errorf("expected error when connecting to wildcard pattern block, got connected=%v, err=%v", connected, err)
	}

	// Ambiguous matches (multiple results, no exact match) - should not error, but return false to launch TUI
	svc.sshCalled = ""
	connected, err = handleDirectConnect("web", svc)
	if err != nil {
		t.Fatalf("expected nil error for ambiguous match, got %v", err)
	}
	if connected {
		t.Errorf("expected connected=false for ambiguous match, got true")
	}
}

func TestResolveSSHConfigFile_Default(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	path, cleanup, err := resolveSSHConfigFile(home, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	expected := filepath.Join(home, ".ssh", "config")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestEnsureMetadataFile(t *testing.T) {
	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		tmpDir := t.TempDir()
		metaFile, err := ensureMetadataFile(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Join(tmpDir, ".neossh", "metadata.json")
		if metaFile != expected {
			t.Errorf("expected %q, got %q", expected, metaFile)
		}
	})

	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		xdgDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgDir)
		metaFile, err := ensureMetadataFile(t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Join(xdgDir, "neossh", "metadata.json")
		if metaFile != expected {
			t.Errorf("expected %q, got %q", expected, metaFile)
		}
	})
}

func TestRootCmd_ImportKnownHostsFlags(t *testing.T) {
	cmd := newRootCmd()

	importFlag := cmd.PersistentFlags().Lookup("import-known-hosts")
	if importFlag == nil {
		t.Fatal("expected persistent flag --import-known-hosts to exist")
	}

	khFlag := cmd.PersistentFlags().Lookup("known-hosts")
	if khFlag == nil {
		t.Fatal("expected persistent flag --known-hosts to exist")
	}
}

func TestRootCmd_ImportKnownHostsParsing(t *testing.T) {
	tests := []struct {
		name                   string
		args                   []string
		expectedImport         bool
		expectedKnownHostsPath string
	}{
		{
			name:                   "default without import flag",
			args:                   []string{},
			expectedImport:         false,
			expectedKnownHostsPath: "",
		},
		{
			name:                   "flag --import-known-hosts",
			args:                   []string{"--import-known-hosts"},
			expectedImport:         true,
			expectedKnownHostsPath: "",
		},
		{
			name:                   "flag --import-known-hosts with --known-hosts",
			args:                   []string{"--import-known-hosts", "--known-hosts", "/custom/known_hosts"},
			expectedImport:         true,
			expectedKnownHostsPath: "/custom/known_hosts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importKnownHosts = false
			knownHostsFile = ""
			cmd := newRootCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("unexpected error parsing flags %v: %v", tt.args, err)
			}
			if importKnownHosts != tt.expectedImport {
				t.Errorf("importKnownHosts = %v, want %v", importKnownHosts, tt.expectedImport)
			}
			if knownHostsFile != tt.expectedKnownHostsPath {
				t.Errorf("knownHostsFile = %q, want %q", knownHostsFile, tt.expectedKnownHostsPath)
			}
		})
	}
}

func TestRootCmd_ImportKnownHosts_ReadOnlyError(t *testing.T) {
	importKnownHosts = false
	knownHostsFile = ""
	sshConfigReadonly = false
	sshConfigFile = ""

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--import-known-hosts", "-r"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error running --import-known-hosts in readonly mode, got nil")
	}
	if !strings.Contains(err.Error(), "cannot import known_hosts in read-only mode") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRootCmd_ImportKnownHosts_Execution(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfgPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(cfgPath, []byte("Host existing\n    HostName 10.0.0.1\n"), 0o600); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	khPath := filepath.Join(tmpDir, "known_hosts")
	khContent := "10.0.0.1 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6\n" +
		"discovered-server.org ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3\n" +
		"[discovered-port.org]:2222 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6\n"
	if err := os.WriteFile(khPath, []byte(khContent), 0o600); err != nil {
		t.Fatalf("failed to write known_hosts: %v", err)
	}

	importKnownHosts = false
	knownHostsFile = ""
	sshConfigReadonly = false
	sshConfigFile = ""

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"--sshconfig", cfgPath,
		"--import-known-hosts",
		"--known-hosts", khPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing import command: %v", err)
	}

	updatedConfig, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read updated config: %v", err)
	}
	cfgStr := string(updatedConfig)

	if !strings.Contains(cfgStr, "Host discovered-server.org") {
		t.Errorf("expected config to contain 'Host discovered-server.org', got:\n%s", cfgStr)
	}
	if !strings.Contains(cfgStr, "Host discovered-port.org-2222") {
		t.Errorf("expected config to contain 'Host discovered-port.org-2222', got:\n%s", cfgStr)
	}
	if !strings.Contains(cfgStr, "Port 2222") {
		t.Errorf("expected config to contain 'Port 2222', got:\n%s", cfgStr)
	}
}

func TestRootCmd_ShowHiddenFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "default is false",
			args:     []string{},
			expected: false,
		},
		{
			name:     "flag --show-hidden",
			args:     []string{"--show-hidden"},
			expected: true,
		},
		{
			name:     "flag shorthand -H",
			args:     []string{"-H"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			showHidden = false
			cmd := newRootCmd()
			cmd.SetArgs(tt.args)
			_ = cmd.ParseFlags(tt.args)

			val, err := cmd.Flags().GetBool("show-hidden")
			if err != nil {
				t.Fatalf("failed to get show-hidden flag: %v", err)
			}
			if val != tt.expected {
				t.Errorf("expected show-hidden=%v, got %v", tt.expected, val)
			}
		})
	}
}

func TestRootCmd_ThemeFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "default is empty",
			args:     []string{},
			expected: "",
		},
		{
			name:     "flag --theme light",
			args:     []string{"--theme", "light"},
			expected: "light",
		},
		{
			name:     "flag shorthand -t system",
			args:     []string{"-t", "system"},
			expected: "system",
		},
		{
			name:     "flag --theme dark",
			args:     []string{"--theme", "dark"},
			expected: "dark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			themeFlag = ""
			cmd := newRootCmd()
			cmd.SetArgs(tt.args)
			_ = cmd.ParseFlags(tt.args)

			val, err := cmd.Flags().GetString("theme")
			if err != nil {
				t.Fatalf("failed to get theme flag: %v", err)
			}
			if val != tt.expected {
				t.Errorf("expected theme=%q, got %q", tt.expected, val)
			}
		})
	}
}

func TestRootCmd_SCPFlag(t *testing.T) {
	scpFlag = ""
	cmd := newRootCmd()
	err := cmd.ParseFlags([]string{"--scp", "myserver"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scpFlag != "myserver" {
		t.Errorf("expected scpFlag=%q, got %q", "myserver", scpFlag)
	}
}

func TestHandleSCPFlag(t *testing.T) {
	svc := &mockDirectConnectService{
		servers: []domain.Server{
			{
				Alias: "web-prod",
				Host:  "10.0.0.1",
				User:  "ubuntu",
				Port:  2202,
			},
		},
	}

	// Test found
	err := handleSCPFlag(svc, "web-prod")
	if err != nil {
		t.Errorf("expected no error for valid server alias, got %v", err)
	}

	// Test not found
	err = handleSCPFlag(svc, "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestRootCmd_PreConnectFlag(t *testing.T) {
	preConnectFlag = ""
	cmd := newRootCmd()
	err := cmd.ParseFlags([]string{"--pre-connect", "vpn-up.sh %h"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preConnectFlag != "vpn-up.sh %h" {
		t.Errorf("expected preConnectFlag=%q, got %q", "vpn-up.sh %h", preConnectFlag)
	}
}

func TestRootCmd_DefaultKeyFlag(t *testing.T) {
	defaultKeyFlag = ""
	cmd := newRootCmd()
	err := cmd.ParseFlags([]string{"--default-key", "~/.ssh/id_ed25519"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultKeyFlag != "~/.ssh/id_ed25519" {
		t.Errorf("expected defaultKeyFlag=%q, got %q", "~/.ssh/id_ed25519", defaultKeyFlag)
	}
}

func TestHandleDefaultKeyFlag(t *testing.T) {
	svc := &mockDirectConnectService{
		defaultKey: "~/.ssh/id_rsa",
	}

	// 1. Flag not changed -> handled=false
	cmd := newRootCmd()
	handled, err := handleDefaultKeyFlag(cmd, svc, false, "")
	if err != nil || handled {
		t.Errorf("expected handled=false, got handled=%v, err=%v", handled, err)
	}

	// 2. Flag changed, empty key -> displays current key
	cmd = newRootCmd()
	_ = cmd.ParseFlags([]string{"--default-key", ""})
	handled, err = handleDefaultKeyFlag(cmd, svc, false, "")
	if err != nil || !handled {
		t.Errorf("expected handled=true with nil error, got handled=%v, err=%v", handled, err)
	}

	// 3. Flag changed with key in read-only mode -> error
	cmd = newRootCmd()
	_ = cmd.ParseFlags([]string{"--default-key", "~/.ssh/new_key"})
	handled, err = handleDefaultKeyFlag(cmd, svc, true, "~/.ssh/new_key")
	if !handled || err == nil || !strings.Contains(err.Error(), "read-only mode") {
		t.Errorf("expected read-only error, got handled=%v, err=%v", handled, err)
	}

	// 4. Flag changed with key in normal mode -> saves key
	cmd = newRootCmd()
	_ = cmd.ParseFlags([]string{"--default-key", "~/.ssh/new_key"})
	handled, err = handleDefaultKeyFlag(cmd, svc, false, "~/.ssh/new_key")
	if err != nil || !handled {
		t.Fatalf("expected handled=true and nil err, got handled=%v, err=%v", handled, err)
	}
	if svc.defaultKey != "~/.ssh/new_key" {
		t.Errorf("expected defaultKey=%q, got %q", "~/.ssh/new_key", svc.defaultKey)
	}
}
