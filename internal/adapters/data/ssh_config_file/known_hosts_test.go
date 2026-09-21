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
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestParseKnownHosts_PlainHostAndIP(t *testing.T) {
	input := `
# Example known_hosts file
example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
192.168.1.10 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6 comment
10.0.0.1 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wZiT/auDWJCvuuymqFMZFUDW36JudbA=
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	expected := []struct {
		alias string
		host  string
		port  int
	}{
		{"example.com", "example.com", 22},
		{"192.168.1.10", "192.168.1.10", 22},
		{"10.0.0.1", "10.0.0.1", 22},
	}

	for i, exp := range expected {
		if servers[i].Alias != exp.alias {
			t.Errorf("[%d] Alias = %q, want %q", i, servers[i].Alias, exp.alias)
		}
		if servers[i].Host != exp.host {
			t.Errorf("[%d] Host = %q, want %q", i, servers[i].Host, exp.host)
		}
		if servers[i].Port != exp.port {
			t.Errorf("[%d] Port = %d, want %d", i, servers[i].Port, exp.port)
		}
	}
}

func TestParseKnownHosts_PortsAndBrackets(t *testing.T) {
	input := `
[git.example.com]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
[192.168.1.50]:8022 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
[bracket-default.org] ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
plain-colon.org:2200 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 4 {
		t.Fatalf("expected 4 servers, got %d", len(servers))
	}

	expected := []struct {
		alias string
		host  string
		port  int
	}{
		{"git.example.com-2222", "git.example.com", 2222},
		{"192.168.1.50-8022", "192.168.1.50", 8022},
		{"bracket-default.org", "bracket-default.org", 22},
		{"plain-colon.org-2200", "plain-colon.org", 2200},
	}

	for i, exp := range expected {
		if servers[i].Alias != exp.alias {
			t.Errorf("[%d] Alias = %q, want %q", i, servers[i].Alias, exp.alias)
		}
		if servers[i].Host != exp.host {
			t.Errorf("[%d] Host = %q, want %q", i, servers[i].Host, exp.host)
		}
		if servers[i].Port != exp.port {
			t.Errorf("[%d] Port = %d, want %d", i, servers[i].Port, exp.port)
		}
	}
}

func TestParseKnownHosts_IPv6(t *testing.T) {
	input := `
[2001:db8::1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
[::1]:22 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
fe80::1 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	if servers[0].Host != "2001:db8::1" || servers[0].Port != 2222 {
		t.Errorf("server 0: unexpected host %q or port %d", servers[0].Host, servers[0].Port)
	}
	if servers[0].Alias != "ipv6-2001-db8--1-2222" {
		t.Errorf("server 0: unexpected alias %q", servers[0].Alias)
	}

	if servers[1].Host != "::1" || servers[1].Port != 22 {
		t.Errorf("server 1: unexpected host %q or port %d", servers[1].Host, servers[1].Port)
	}
	if servers[1].Alias != "ipv6---1" {
		t.Errorf("server 1: unexpected alias %q", servers[1].Alias)
	}

	if servers[2].Host != "fe80::1" || servers[2].Port != 22 {
		t.Errorf("server 2: unexpected host %q or port %d", servers[2].Host, servers[2].Port)
	}
	if servers[2].Alias != "ipv6-fe80--1" {
		t.Errorf("server 2: unexpected alias %q", servers[2].Alias)
	}
}

func TestParseKnownHosts_HashedEntries_Skipped(t *testing.T) {
	input := `
|1|F1E1b7Ds9ng/SQn95LVB3FvBveA=|P23DxY32TG0yvK4DO9FpR3c6w+8= ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wZiT/auDWJCvuuymqFMZFUDW36JudbA=
|1|otherhash|secondhash ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
valid-host.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected only 1 server (hashed entries skipped), got %d", len(servers))
	}

	if servers[0].Host != "valid-host.com" {
		t.Errorf("expected valid-host.com, got %q", servers[0].Host)
	}
}

func TestParseKnownHosts_CommaSeparatedMultipleHosts(t *testing.T) {
	input := `
router.lan,192.168.1.1,[router-alt.lan]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 3 {
		t.Fatalf("expected 3 servers from comma-separated line, got %d", len(servers))
	}

	if servers[0].Host != "router.lan" || servers[0].Port != 22 {
		t.Errorf("servers[0] = %+v", servers[0])
	}
	if servers[1].Host != "192.168.1.1" || servers[1].Port != 22 {
		t.Errorf("servers[1] = %+v", servers[1])
	}
	if servers[2].Host != "router-alt.lan" || servers[2].Port != 2222 {
		t.Errorf("servers[2] = %+v", servers[2])
	}
}

func TestParseKnownHosts_Markers(t *testing.T) {
	input := `
@revoked revoked-host.example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
@cert-authority *.corp.example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
@cert-authority ca.internal.net ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected 1 server (revoked skipped, wildcard skipped), got %d", len(servers))
	}

	if servers[0].Host != "ca.internal.net" {
		t.Errorf("expected ca.internal.net, got %q", servers[0].Host)
	}
}

func TestParseKnownHosts_MalformedAndInvalid(t *testing.T) {
	input := `
# Invalid and malformed entries
[unclosed-bracket:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
[host]:notaport ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
[host]:999999 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
[host]:0 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
[host]:22trailingjunk ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
invalid..host ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
-leadingdash.host ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
trailingdot.host. ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
host with spaces ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
onlyonetoken
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 0 {
		t.Fatalf("expected 0 servers from malformed inputs, got %d", len(servers))
	}
}

func TestParseKnownHosts_DeduplicatesAcrossLines(t *testing.T) {
	input := `
github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wZiT/auDWJCvuuymqFMZFUDW36JudbA=
github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
[github.com]:22 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
`
	servers, err := ParseKnownHosts(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected exactly 1 server after deduplicating multiple keys for same host, got %d", len(servers))
	}

	if servers[0].Host != "github.com" || servers[0].Port != 22 {
		t.Errorf("unexpected server: %+v", servers[0])
	}
}

func TestRepository_DiscoverKnownHosts_DeduplicatesExistingConfig(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	cfgPath := filepath.Join(fs.tempDir, "config")
	metaPath := filepath.Join(fs.tempDir, "meta.json")
	khPath := filepath.Join(fs.tempDir, "known_hosts")

	cfgContent := `
Host web-server
    HostName 192.168.1.100
    Port 22

Host github.com
    HostName github.com
    User git

Host *.wildcard
    HostName %h.corp
`
	fs.write(cfgPath, cfgContent)

	khContent := `
github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
192.168.1.100 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
[192.168.1.100]:2222 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
brand-new.org ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLzapDVsvYX+28VUvxGhRNO3
|1|hash|hash ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
`
	fs.write(khPath, khContent)

	logger := zap.NewNop().Sugar()
	repo := NewRepositoryWithFS(logger, cfgPath, metaPath, fs)

	candidates, result, err := repo.DiscoverKnownHosts(khPath)
	if err != nil {
		t.Fatalf("DiscoverKnownHosts failed: %v", err)
	}

	// Total unique valid in known_hosts: github.com, 192.168.1.100, [192.168.1.100]:2222, brand-new.org = 4
	// Existing in config: github.com (alias/host match), 192.168.1.100 (HostName match on web-server) = 2 skipped
	// Candidates to import: [192.168.1.100]:2222 and brand-new.org = 2
	if result.Discovered != 4 {
		t.Errorf("result.Discovered = %d, want 4", result.Discovered)
	}
	if result.Skipped != 2 {
		t.Errorf("result.Skipped = %d, want 2", result.Skipped)
	}
	if len(candidates) != 2 {
		t.Fatalf("len(candidates) = %d, want 2", len(candidates))
	}

	expectedHosts := map[string]int{
		"192.168.1.100": 2222,
		"brand-new.org": 22,
	}

	for _, cand := range candidates {
		expPort, ok := expectedHosts[cand.Host]
		if !ok {
			t.Errorf("unexpected candidate host: %q", cand.Host)
			continue
		}
		if cand.Port != expPort {
			t.Errorf("candidate %q port = %d, want %d", cand.Host, cand.Port, expPort)
		}
	}
}

func TestRepository_ImportKnownHosts_Success(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	cfgPath := filepath.Join(fs.tempDir, "config")
	metaPath := filepath.Join(fs.tempDir, "meta.json")
	khPath := filepath.Join(fs.tempDir, "known_hosts")

	fs.write(cfgPath, "Host existing\n    HostName 10.0.0.1\n")
	khContent := `
10.0.0.1 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
10.0.0.2 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
[10.0.0.3]:2200 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQW6
`
	fs.write(khPath, khContent)

	logger := zap.NewNop().Sugar()
	repo := NewRepositoryWithFS(logger, cfgPath, metaPath, fs)

	res, err := repo.ImportKnownHosts(khPath)
	if err != nil {
		t.Fatalf("ImportKnownHosts failed: %v", err)
	}

	if res.Discovered != 3 {
		t.Errorf("res.Discovered = %d, want 3", res.Discovered)
	}
	if res.Skipped != 1 {
		t.Errorf("res.Skipped = %d, want 1", res.Skipped)
	}
	if res.Imported != 2 {
		t.Errorf("res.Imported = %d, want 2", res.Imported)
	}

	// Verify imported hosts exist in repository
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 3 {
		t.Fatalf("expected 3 total servers, got %d", len(servers))
	}

	// Re-importing should import 0 and skip all 3
	res2, err := repo.ImportKnownHosts(khPath)
	if err != nil {
		t.Fatalf("second ImportKnownHosts failed: %v", err)
	}
	if res2.Imported != 0 {
		t.Errorf("res2.Imported = %d, want 0", res2.Imported)
	}
	if res2.Skipped != 3 {
		t.Errorf("res2.Skipped = %d, want 3", res2.Skipped)
	}
}

func TestRepository_DiscoverKnownHosts_FileNotFound(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	cfgPath := filepath.Join(fs.tempDir, "config")
	metaPath := filepath.Join(fs.tempDir, "meta.json")
	fs.write(cfgPath, "")

	logger := zap.NewNop().Sugar()
	repo := NewRepositoryWithFS(logger, cfgPath, metaPath, fs)

	_, _, err := repo.DiscoverKnownHosts(filepath.Join(fs.tempDir, "nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent known_hosts file, got nil")
	}
	if !strings.Contains(err.Error(), "known_hosts file not found") {
		t.Errorf("expected 'known_hosts file not found' in error, got %q", err.Error())
	}
}
