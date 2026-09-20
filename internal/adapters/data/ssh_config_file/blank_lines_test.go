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

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

func TestUpdateServer_NoExtraBlankLinesOnCRLF(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	// Initial CRLF config file as typically found on Windows
	crlfContent := "Host example-host\r\n    HostName example-ip\r\n    Port 22\r\n    User root\r\n"
	fs.write(main, crlfContent)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	// Perform 5 successive edit/save cycles
	for i := 1; i <= 5; i++ {
		oldServer := domain.Server{
			Alias: "example-host",
			Host:  "example-ip",
			User:  "root",
			Port:  21 + i,
		}
		newServer := domain.Server{
			Alias: "example-host",
			Host:  "example-ip",
			User:  "root",
			Port:  22 + i,
		}
		if err := r.UpdateServer(oldServer, newServer); err != nil {
			t.Fatalf("cycle %d: UpdateServer failed: %v", i, err)
		}

		content := fs.read(main)
		// Check that no double newlines were inserted inside the host block
		trimmed := strings.TrimSpace(content)
		lines := strings.Split(trimmed, "\n")
		for lineIdx, line := range lines {
			if strings.TrimSpace(line) == "" {
				t.Fatalf("cycle %d: unexpected blank line found at line %d:\n%q", i, lineIdx, content)
			}
		}
	}
}

func TestUpdateServer_CleansLeadingBlankLine(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	// Config with extraneous blank line after Host
	dirtyContent := "Host example-host\n\n    HostName example-ip\n    Port 22\n    User root\n"
	fs.write(main, dirtyContent)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	oldServer := domain.Server{
		Alias: "example-host",
		Host:  "example-ip",
		User:  "root",
		Port:  22,
	}
	newServer := domain.Server{
		Alias: "example-host",
		Host:  "example-ip",
		User:  "root",
		Port:  2222,
	}
	if err := r.UpdateServer(oldServer, newServer); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	content := fs.read(main)
	if strings.Contains(content, "Host example-host\n\n") {
		t.Errorf("leading blank line was not cleaned up:\n%s", content)
	}
}

func TestUpdateServer_PreservesSingleBlankLineBetweenHosts(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	twoHosts := "Host host1\n    HostName 1.1.1.1\n    User u1\n\nHost host2\n    HostName 2.2.2.2\n    User u2\n"
	fs.write(main, twoHosts)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	old1 := domain.Server{Alias: "host1", Host: "1.1.1.1", User: "u1"}
	new1 := domain.Server{Alias: "host1", Host: "1.1.1.1", User: "u1", Port: 2201}
	if err := r.UpdateServer(old1, new1); err != nil {
		t.Fatalf("UpdateServer host1 failed: %v", err)
	}

	content := fs.read(main)
	// Must contain single blank line between host1 and host2
	if !strings.Contains(content, "\n\nHost host2") {
		t.Errorf("expected separation between hosts, got:\n%s", content)
	}
	// Must NOT contain 3 or more newlines
	if strings.Contains(content, "\n\n\n") {
		t.Errorf("found runaway newlines in output:\n%s", content)
	}
}
