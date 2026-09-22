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

func TestGuardAndRestoreIncludeDirectives(t *testing.T) {
	in := "Include terraform.d/*\n" +
		"  include  spaced.conf # trailing\n" +
		"Host a\n" +
		"  HostName x\n" +
		"# a plain comment\n"
	guarded := string(guardIncludeDirectives([]byte(in)))
	for _, line := range strings.Split(guarded, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(strings.ToLower(trimmed), "include") {
			t.Fatalf("active Include directive survived guarding: %q", line)
		}
	}
	if !strings.Contains(guarded, includeGuardMarker) {
		t.Fatalf("marker missing: %q", guarded)
	}
	if got := restoreIncludeDirectives(guarded); got != in {
		t.Fatalf("round-trip mismatch:\n got %q\nwant %q", got, in)
	}
}

// Regression: an `Include` glob that matches a directory must not break loading.
// The third-party parser used to call os.ReadFile on the directory and fail with
// "is a directory", leaving the TUI with zero targets.
func TestResolveIncludes_GlobMatchingDirectory(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf.d")
	if err := os.MkdirAll(filepath.Join(confDir, "keys"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	main := filepath.Join(dir, "config")
	if err := os.WriteFile(main, []byte("Include "+confDir+"/*\nHost local\n  HostName 127.0.0.1\n"), 0o600); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "extra"), []byte("Host included-host\n  HostName 10.0.0.1\n"), 0o600); err != nil {
		t.Fatalf("write include: %v", err)
	}

	logger := zap.NewNop().Sugar()
	r := &Repository{
		logger:          logger,
		configPath:      main,
		fileSystem:      DefaultFileSystem{},
		metadataManager: newMetadataManager(filepath.Join(dir, "metadata.json"), logger),
	}

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	found := false
	for _, s := range servers {
		if s.Alias == "included-host" {
			found = true
		}
	}
	if !found {
		t.Fatalf("included host not loaded, got %d servers: %+v", len(servers), servers)
	}
}

// The Include directive must survive an edit of the config file byte-for-byte.
func TestWriteRestoresIncludeDirective(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	inc := "/home/u/.ssh/extra"
	fs.write(main, "Include "+inc+"\nHost local\n  HostName 127.0.0.1\n")
	fs.write(inc, "Host existing\n  HostName 2.2.2.2\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	newSrv := domain.Server{Alias: "local", Host: "127.0.0.1", User: "root"}
	if err := r.UpdateServer(domain.Server{Alias: "local"}, newSrv); err != nil {
		t.Fatalf("update: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "Include "+inc) {
		t.Fatalf("Include directive lost on write: %q", content)
	}
	if strings.Contains(content, includeGuardMarker) {
		t.Fatalf("guard marker leaked into written config: %q", content)
	}
}
