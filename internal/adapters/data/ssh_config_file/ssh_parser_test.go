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
	"slices"
	"strings"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

func TestParseMultipleAliases_PreservesAll(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	configContent := `Host web1 web.prod.internal prod-web
    HostName 10.0.0.1
    User dev
    Port 2222
`
	fs.write(main, configContent)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	srv := servers[0]
	if srv.Alias != "web1" {
		t.Errorf("expected Alias 'web1', got %q", srv.Alias)
	}

	expectedAliases := []string{"web1", "web.prod.internal", "prod-web"}
	if !slices.Equal(srv.Aliases, expectedAliases) {
		t.Errorf("expected Aliases %v, got %v", expectedAliases, srv.Aliases)
	}

	if srv.Host != "10.0.0.1" || srv.User != "dev" || srv.Port != 2222 {
		t.Errorf("server fields mismatch: %+v", srv)
	}

	// Verify searching by any alias finds the server
	for _, aliasQuery := range []string{"web1", "web.prod.internal", "prod-web", "prod"} {
		results, err := r.ListServers(aliasQuery)
		if err != nil {
			t.Fatalf("ListServers(%q) failed: %v", aliasQuery, err)
		}
		if len(results) != 1 {
			t.Errorf("ListServers(%q) expected 1 match, got %d", aliasQuery, len(results))
		} else if results[0].Alias != "web1" {
			t.Errorf("ListServers(%q) expected Alias 'web1', got %q", aliasQuery, results[0].Alias)
		}
	}
}

func TestAddServer_WithMultipleAliases_OutputsHostLine(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, "")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	newServer := domain.Server{
		Alias:   "alias1",
		Aliases: []string{"alias1", "alias2", "alias3"},
		Host:    "192.168.1.100",
		User:    "admin",
		Port:    22,
	}

	if err := r.AddServer(newServer); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "Host alias1 alias2 alias3") {
		t.Errorf("expected config to contain 'Host alias1 alias2 alias3', got:\n%s", content)
	}

	// Reload and verify
	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if !slices.Equal(servers[0].Aliases, []string{"alias1", "alias2", "alias3"}) {
		t.Errorf("expected Aliases [alias1 alias2 alias3], got %v", servers[0].Aliases)
	}
}

func TestUpdateServer_WithMultipleAliases_OutputsHostLine(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	initialConfig := `Host alias1 alias2 alias3
    HostName 10.0.0.1
    User ubuntu
`
	fs.write(main, initialConfig)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	orig := servers[0]
	updated := orig
	updated.Port = 2222
	updated.User = "deploy"

	if err := r.UpdateServer(orig, updated); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "Host alias1 alias2 alias3") {
		t.Errorf("expected config to contain 'Host alias1 alias2 alias3', got:\n%s", content)
	}
	if !strings.Contains(content, "Port 2222") {
		t.Errorf("expected config to contain 'Port 2222', got:\n%s", content)
	}

	// Reload and verify
	reloaded, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers after update failed: %v", err)
	}
	if len(reloaded) != 1 {
		t.Fatalf("expected 1 server, got %d", len(reloaded))
	}
	if !slices.Equal(reloaded[0].Aliases, []string{"alias1", "alias2", "alias3"}) {
		t.Errorf("expected Aliases preserved as [alias1 alias2 alias3], got %v", reloaded[0].Aliases)
	}
	if reloaded[0].Port != 2222 || reloaded[0].User != "deploy" {
		t.Errorf("updated fields mismatch: %+v", reloaded[0])
	}
}

func TestUpdateServer_ModifyOneServer_DoesNotAlterOtherServerAliases(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	initialConfig := `Host alpha a1 a2
    HostName alpha.example.com
    User u1

Host beta b1 b2
    HostName beta.example.com
    User u2
`
	fs.write(main, initialConfig)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	var alphaSrv, betaSrv domain.Server
	for _, s := range servers {
		if s.Alias == "alpha" {
			alphaSrv = s
		} else if s.Alias == "beta" {
			betaSrv = s
		}
	}

	if !slices.Equal(alphaSrv.Aliases, []string{"alpha", "a1", "a2"}) {
		t.Fatalf("expected alpha aliases [alpha a1 a2], got %v", alphaSrv.Aliases)
	}
	if !slices.Equal(betaSrv.Aliases, []string{"beta", "b1", "b2"}) {
		t.Fatalf("expected beta aliases [beta b1 b2], got %v", betaSrv.Aliases)
	}

	// Mutate only alpha (e.g. change port and user)
	updatedAlpha := alphaSrv
	updatedAlpha.Port = 8022
	updatedAlpha.User = "superalpha"

	if err := r.UpdateServer(alphaSrv, updatedAlpha); err != nil {
		t.Fatalf("UpdateServer alpha failed: %v", err)
	}

	content := fs.read(main)
	// Check that beta's host line is completely intact
	if !strings.Contains(content, "Host beta b1 b2") {
		t.Errorf("expected 'Host beta b1 b2' to remain unchanged, got:\n%s", content)
	}
	if !strings.Contains(content, "Host alpha a1 a2") {
		t.Errorf("expected 'Host alpha a1 a2' to remain intact, got:\n%s", content)
	}

	// Reload all servers
	reloaded, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers after update failed: %v", err)
	}
	var reloadedBeta domain.Server
	for _, s := range reloaded {
		if s.Alias == "beta" {
			reloadedBeta = s
		}
	}
	if !slices.Equal(reloadedBeta.Aliases, []string{"beta", "b1", "b2"}) {
		t.Errorf("expected beta aliases still [beta b1 b2], got %v", reloadedBeta.Aliases)
	}
	if reloadedBeta.Host != "beta.example.com" || reloadedBeta.User != "u2" {
		t.Errorf("expected beta fields unchanged, got %+v", reloadedBeta)
	}
}

func TestUpdateServer_UpdateAliasesList(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	initialConfig := `Host web1 web2
    HostName 10.0.0.1
`
	fs.write(main, initialConfig)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}

	orig := servers[0]
	updated := orig
	updated.Aliases = []string{"web1", "web2", "web3", "web-prod"}

	if err := r.UpdateServer(orig, updated); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "Host web1 web2 web3 web-prod") {
		t.Errorf("expected 'Host web1 web2 web3 web-prod', got:\n%s", content)
	}

	reloaded, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if !slices.Equal(reloaded[0].Aliases, []string{"web1", "web2", "web3", "web-prod"}) {
		t.Errorf("expected Aliases updated, got %v", reloaded[0].Aliases)
	}
}
