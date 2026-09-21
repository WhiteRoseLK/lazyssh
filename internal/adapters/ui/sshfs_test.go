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
	"strings"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/rivo/tview"
)

func TestBuildSSHFSCommand(t *testing.T) {
	server := domain.Server{
		Alias:         "devbox",
		Host:          "192.168.1.50",
		User:          "developer",
		Port:          2222,
		IdentityFiles: []string{"~/.ssh/id_ed25519"},
		ProxyJump:     "bastion.corp",
	}

	// Case 1: Full config with reconnect and read-only
	cmd := BuildSSHFSCommand(server, "/var/log", "/tmp/mnt", true, true, false)
	expectedParts := []string{
		"sshfs",
		"-p 2222",
		"-o IdentityFile=~/.ssh/id_ed25519",
		"-o ProxyJump=bastion.corp",
		"-o reconnect",
		"-o ro",
		"developer@192.168.1.50:/var/log",
		"/tmp/mnt",
	}
	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("BuildSSHFSCommand() missing part %q in: %s", part, cmd)
		}
	}

	// Case 2: Use SSH config alias
	aliasCmd := BuildSSHFSCommand(server, "/", "~/mounts/devbox", false, true, true)
	if !strings.Contains(aliasCmd, "devbox:/") {
		t.Errorf("expected alias target 'devbox:/', got %s", aliasCmd)
	}
	if !strings.Contains(aliasCmd, "-o reconnect") {
		t.Errorf("expected '-o reconnect', got %s", aliasCmd)
	}
	if strings.Contains(aliasCmd, "-p 2222") {
		t.Errorf("did not expect explicit port when useAlias=true, got %s", aliasCmd)
	}

	// Case 3: Empty remote path defaults to "/"
	defPathCmd := BuildSSHFSCommand(server, "", "", false, false, false)
	if !strings.Contains(defPathCmd, ":/") {
		t.Errorf("expected default root ':/', got %s", defPathCmd)
	}
	if !strings.Contains(defPathCmd, "~/mounts/devbox") {
		t.Errorf("expected default mount point '~/mounts/devbox', got %s", defPathCmd)
	}
}

func TestBuildSSHFSUnmountCommand(t *testing.T) {
	unmount := BuildSSHFSUnmountCommand("~/mounts/web")
	if !strings.Contains(unmount, "fusermount3 -u ~/mounts/web") || !strings.Contains(unmount, "umount ~/mounts/web") {
		t.Errorf("unexpected unmount command: %s", unmount)
	}

	defUnmount := BuildSSHFSUnmountCommand("")
	if !strings.Contains(defUnmount, "remote") {
		t.Errorf("expected default remote unmount, got %s", defUnmount)
	}
}

func TestSSHFSModal(t *testing.T) {
	app := tview.NewApplication()
	srv := domain.Server{
		Alias: "srv1",
		Host:  "10.0.0.1",
		User:  "root",
		Port:  22,
	}

	modal := NewSSHFSModal(app, srv)
	if modal == nil {
		t.Fatal("expected non-nil SSHFSModal")
	}

	var copiedCmd string
	modal.OnCopied(func(cmd string) {
		copiedCmd = cmd
	})

	var canceled bool
	modal.OnCancel(func() {
		canceled = true
	})

	if err := modal.Show(); err != nil {
		t.Fatalf("Show() failed: %v", err)
	}

	// Verify preview was populated
	preview := modal.previewText.GetText(false)
	if !strings.Contains(preview, "sshfs") {
		t.Errorf("expected preview to contain sshfs command, got %q", preview)
	}

	// Trigger callbacks
	if modal.onCopied != nil {
		modal.onCopied("test-cmd")
	}
	if copiedCmd != "test-cmd" {
		t.Errorf("expected copiedCmd='test-cmd', got %q", copiedCmd)
	}

	if modal.onCancel != nil {
		modal.onCancel()
	}
	if !canceled {
		t.Error("expected canceled=true")
	}
}
