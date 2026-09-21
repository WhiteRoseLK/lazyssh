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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestBuildSCPCommand_Basic(t *testing.T) {
	server := domain.Server{
		Alias: "prod-server",
		Host:  "192.168.1.100",
		User:  "admin",
		Port:  22,
	}

	upload := BuildSCPUploadCommand(server, "local.txt", "/remote/path/file.txt", false, false)
	expectedUpload := "scp local.txt admin@192.168.1.100:/remote/path/file.txt"
	if upload != expectedUpload {
		t.Errorf("BuildSCPUploadCommand() = %q, want %q", upload, expectedUpload)
	}

	download := BuildSCPDownloadCommand(server, "/remote/path/file.txt", "local.txt", false, false)
	expectedDownload := "scp admin@192.168.1.100:/remote/path/file.txt local.txt"
	if download != expectedDownload {
		t.Errorf("BuildSCPDownloadCommand() = %q, want %q", download, expectedDownload)
	}
}

func TestBuildSCPCommand_CustomPortAndKey(t *testing.T) {
	server := domain.Server{
		Alias:         "custom-server",
		Host:          "example.org",
		User:          "deploy",
		Port:          2222,
		IdentityFiles: []string{"/home/user/.ssh/id_ed25519"},
		ProxyJump:     "jump.example.org",
	}

	cmd := BuildSCPUploadCommand(server, "archive.tar.gz", "~/backup/", true, false)
	expectedParts := []string{
		"scp",
		"-r",
		"-P 2222",
		"-i /home/user/.ssh/id_ed25519",
		"-J jump.example.org",
		"archive.tar.gz",
		"deploy@example.org:~/backup/",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain %q, got: %q", part, cmd)
		}
	}
}

func TestBuildSCPCommand_AliasMode(t *testing.T) {
	server := domain.Server{
		Alias:         "myalias",
		Host:          "10.0.0.5",
		User:          "root",
		Port:          2200,
		IdentityFiles: []string{"/key"},
	}

	upload := BuildSCPUploadCommand(server, "file.txt", "~/remote.txt", false, true)
	if upload != "scp file.txt myalias:~/remote.txt" {
		t.Errorf("BuildSCPUploadCommand(useAlias=true) = %q, want %q", upload, "scp file.txt myalias:~/remote.txt")
	}

	download := BuildSCPDownloadCommand(server, "~/remote.txt", "file.txt", false, true)
	if download != "scp myalias:~/remote.txt file.txt" {
		t.Errorf("BuildSCPDownloadCommand(useAlias=true) = %q, want %q", download, "scp myalias:~/remote.txt file.txt")
	}
}

func TestBuildSCPCommand_QuotedPaths(t *testing.T) {
	server := domain.Server{
		Alias: "box",
		Host:  "box.local",
		User:  "user",
	}

	cmd := BuildSCPUploadCommand(server, "local file with spaces.txt", "remote dir with spaces/", false, false)
	if !strings.Contains(cmd, "\"local file with spaces.txt\"") {
		t.Errorf("expected local path to be quoted, got: %q", cmd)
	}
	if !strings.Contains(cmd, "user@box.local:\"remote dir with spaces/\"") {
		t.Errorf("expected remote target to be quoted, got: %q", cmd)
	}
}

func TestSCPModal_ShowAndCallbacks(t *testing.T) {
	app := tview.NewApplication()
	server := domain.Server{
		Alias: "test-server",
		Host:  "127.0.0.1",
		User:  "tester",
		Port:  2222,
	}

	modal := NewSCPModal(app, server)
	var copiedCmd string
	var canceled bool

	modal.OnCopied(func(cmd string) {
		copiedCmd = cmd
	}).OnCancel(func() {
		canceled = true
	})

	err := modal.Show()
	if err != nil {
		t.Fatalf("unexpected error from Show(): %v", err)
	}

	// Verify initial preview command
	initialCmd := modal.currentCommand()
	if !strings.Contains(initialCmd, "tester@127.0.0.1:~/") {
		t.Errorf("expected initial command to contain target, got: %q", initialCmd)
	}

	// Test copy trigger
	modal.copyAndClose()
	if copiedCmd == "" {
		t.Error("expected onCopied callback to be invoked with command")
	}

	// Test cancel trigger
	modal.cancel()
	if !canceled {
		t.Error("expected onCancel callback to be invoked")
	}
}

func TestSCPModal_EscapeCancels(t *testing.T) {
	app := tview.NewApplication()
	server := domain.Server{
		Alias: "test-server",
		Host:  "127.0.0.1",
	}

	modal := NewSCPModal(app, server)
	var canceled bool
	modal.OnCancel(func() {
		canceled = true
	})
	_ = modal.Show()

	handler := modal.form.GetInputCapture()
	if handler != nil {
		event := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
		handler(event)
		if !canceled {
			t.Error("expected Escape key to trigger cancel callback")
		}
	}
}
