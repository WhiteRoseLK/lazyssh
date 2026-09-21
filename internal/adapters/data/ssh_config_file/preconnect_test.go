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
	"testing"
)

func TestListServers_ReadsPreConnectFromConfigComments(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	configContent := `Host vpn-srv # pre-connect: /usr/local/bin/vpn-up.sh %h
    HostName 10.0.0.1
    User ubuntu

Host wol-srv
    # hook: wakeonlan 00:11:22:33:44:55
    HostName 10.0.0.2
    User admin
`
	fs.write(main, configContent)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}

	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	for _, s := range servers {
		switch s.Alias {
		case "vpn-srv":
			expected := "/usr/local/bin/vpn-up.sh %h"
			if s.PreConnectCommand != expected {
				t.Errorf("vpn-srv PreConnectCommand = %q, want %q", s.PreConnectCommand, expected)
			}
		case "wol-srv":
			expected := "wakeonlan 00:11:22:33:44:55"
			if s.PreConnectCommand != expected {
				t.Errorf("wol-srv PreConnectCommand = %q, want %q", s.PreConnectCommand, expected)
			}
		default:
			t.Fatalf("unexpected server: %s", s.Alias)
		}
	}
}

func TestRepository_PreConnectMetadataPersistence(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()
	main := "/home/u/.ssh/config"
	fs.write(main, "Host my-box\n    HostName 10.1.1.1\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil || len(servers) != 1 {
		t.Fatalf("expected 1 server, got %v (err=%v)", servers, err)
	}

	updated := servers[0]
	updated.PreConnectCommand = "vpn-start.sh %h"

	if err := r.UpdateServer(servers[0], updated); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	// Reload from repository and ensure PreConnectCommand is preserved from metadata
	reloaded, err := r.ListServers("")
	if err != nil || len(reloaded) != 1 {
		t.Fatalf("failed reloading servers: %v", err)
	}
	if reloaded[0].PreConnectCommand != "vpn-start.sh %h" {
		t.Errorf("expected PreConnectCommand 'vpn-start.sh %%h', got %q", reloaded[0].PreConnectCommand)
	}

	// Test Global Settings PreConnect
	if err := r.SavePreConnectCommand("global-auth.sh"); err != nil {
		t.Fatalf("SavePreConnectCommand failed: %v", err)
	}
	globalCmd, err := r.GetPreConnectCommand()
	if err != nil || globalCmd != "global-auth.sh" {
		t.Errorf("expected global PreConnectCommand 'global-auth.sh', got %q (err=%v)", globalCmd, err)
	}
}
