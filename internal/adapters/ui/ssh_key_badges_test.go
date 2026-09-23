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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

func TestExpandIdentityFile(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde slash prefix",
			input:    "~/.ssh/id_ed25519",
			expected: filepath.Join(home, ".ssh/id_ed25519"),
		},
		{
			name:     "tilde only",
			input:    "~",
			expected: home,
		},
		{
			name:     "absolute path unchanged",
			input:    "/etc/ssh/keys/id_rsa",
			expected: "/etc/ssh/keys/id_rsa",
		},
		{
			name:     "relative path unchanged",
			input:    "keys/id_rsa",
			expected: "keys/id_rsa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandIdentityFile(tt.input)
			if result != tt.expected {
				t.Errorf("expandIdentityFile(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatKeyTypeBadge(t *testing.T) {
	sd := NewServerDetails()

	tests := []struct {
		name           string
		key            *domain.SSHKey
		expectContains []string
		expectAbsent   []string
	}{
		{
			name: "ed25519 key",
			key: &domain.SSHKey{
				Type: "ed25519",
				Size: 256,
			},
			expectContains: []string{"ED25519-256"},
			expectAbsent:   []string{"FIDO2"},
		},
		{
			name: "rsa 4096 key",
			key: &domain.SSHKey{
				Type: "rsa",
				Size: 4096,
			},
			expectContains: []string{"RSA-4096"},
			expectAbsent:   []string{"FIDO2"},
		},
		{
			name: "ecdsa key no size",
			key: &domain.SSHKey{
				Type: "ecdsa",
			},
			expectContains: []string{"ECDSA"},
			expectAbsent:   []string{"FIDO2"},
		},
		{
			name: "ed25519 FIDO2 key",
			key: &domain.SSHKey{
				Type:    "ed25519",
				Size:    256,
				IsFIDO2: true,
			},
			expectContains: []string{"ED25519-256", "FIDO2/Security Key"},
		},
		{
			name: "ecdsa FIDO2 key",
			key: &domain.SSHKey{
				Type:    "ecdsa",
				Size:    256,
				IsFIDO2: true,
			},
			expectContains: []string{"ECDSA-256", "FIDO2/Security Key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sd.formatKeyTypeBadge(tt.key)
			for _, substr := range tt.expectContains {
				if !strings.Contains(result, substr) {
					t.Errorf("formatKeyTypeBadge() = %q, want it to contain %q", result, substr)
				}
			}
			for _, substr := range tt.expectAbsent {
				if strings.Contains(result, substr) {
					t.Errorf("formatKeyTypeBadge() = %q, want it NOT to contain %q", result, substr)
				}
			}
		})
	}
}

func TestRenderKeyWithBadges(t *testing.T) {
	sd := NewServerDetails()

	t.Run("no identity files", func(t *testing.T) {
		server := domain.Server{}
		result := sd.renderKeyWithBadges(server)
		if !strings.Contains(result, "-") {
			t.Errorf("renderKeyWithBadges() = %q, expected dim dash for no keys", result)
		}
	})

	t.Run("identity file without git service", func(t *testing.T) {
		// Create a temporary key file to simulate an existing key
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "id_test")
		if err := os.WriteFile(keyPath, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}

		server := domain.Server{
			IdentityFiles: []string{keyPath},
		}
		result := sd.renderKeyWithBadges(server)
		// No gitService so no badge, but path should appear
		if !strings.Contains(result, keyPath) {
			t.Errorf("renderKeyWithBadges() = %q, expected key path", result)
		}
		// File exists so no missing warning
		if strings.Contains(result, "missing") {
			t.Errorf("renderKeyWithBadges() = %q, should not contain 'missing' for existing file", result)
		}
	})

	t.Run("identity file missing on disk", func(t *testing.T) {
		server := domain.Server{
			IdentityFiles: []string{"/nonexistent/path/id_rsa"},
		}
		result := sd.renderKeyWithBadges(server)
		if !strings.Contains(result, "missing") {
			t.Errorf("renderKeyWithBadges() = %q, expected 'missing' for nonexistent file", result)
		}
	})

	t.Run("with git service and matching key", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "id_ed25519")

		mockGit := &mockGitServiceForUI{
			keys: []domain.SSHKey{
				{
					Path:       keyPath,
					Type:       "ed25519",
					Size:       256,
					FileExists: true,
				},
			},
		}
		mockRepo := &mockServerRepoForUI{}

		sdWithGit := NewServerDetails()
		sdWithGit.SetGitService(mockGit, mockRepo)

		server := domain.Server{
			IdentityFiles: []string{keyPath},
		}
		result := sdWithGit.renderKeyWithBadges(server)
		if !strings.Contains(result, "ED25519-256") {
			t.Errorf("renderKeyWithBadges() = %q, expected 'ED25519-256' badge", result)
		}
	})

	t.Run("with FIDO2 key", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "id_ed25519_sk")

		mockGit := &mockGitServiceForUI{
			keys: []domain.SSHKey{
				{
					Path:       keyPath,
					Type:       "ed25519",
					Size:       256,
					IsFIDO2:    true,
					FileExists: true,
				},
			},
		}
		mockRepo := &mockServerRepoForUI{}

		sdWithGit := NewServerDetails()
		sdWithGit.SetGitService(mockGit, mockRepo)

		server := domain.Server{
			IdentityFiles: []string{keyPath},
		}
		result := sdWithGit.renderKeyWithBadges(server)
		if !strings.Contains(result, "ED25519-256") {
			t.Errorf("renderKeyWithBadges() = %q, expected 'ED25519-256' badge", result)
		}
		if !strings.Contains(result, "FIDO2") {
			t.Errorf("renderKeyWithBadges() = %q, expected 'FIDO2' badge", result)
		}
	})

	t.Run("key file missing with git service", func(t *testing.T) {
		keyPath := "/nonexistent/id_ed25519"

		mockGit := &mockGitServiceForUI{
			keys: []domain.SSHKey{
				{
					Path:       keyPath,
					Type:       "ed25519",
					Size:       256,
					FileExists: false,
				},
			},
		}
		mockRepo := &mockServerRepoForUI{}

		sdWithGit := NewServerDetails()
		sdWithGit.SetGitService(mockGit, mockRepo)

		server := domain.Server{
			IdentityFiles: []string{keyPath},
		}
		result := sdWithGit.renderKeyWithBadges(server)
		if !strings.Contains(result, "missing") {
			t.Errorf("renderKeyWithBadges() = %q, expected 'missing' badge for missing file", result)
		}
	})
}
