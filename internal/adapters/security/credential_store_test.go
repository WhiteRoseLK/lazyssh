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

package security

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
	"go.uber.org/zap"
)

func TestCredentialStore_MockKeyring(t *testing.T) {
	keyring.MockInit()

	tmpDir := t.TempDir()
	store := NewCredentialStore(zap.NewNop().Sugar(), tmpDir)

	// 1. Get non-existent password
	pwd, err := store.GetPassword("server-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pwd != "" {
		t.Fatalf("expected empty password, got %q", pwd)
	}

	// 2. Set password
	if err := store.SetPassword("server-1", "supersecret123"); err != nil {
		t.Fatalf("failed to set password: %v", err)
	}

	// 3. Get password
	pwd, err = store.GetPassword("server-1")
	if err != nil {
		t.Fatalf("failed to get password: %v", err)
	}
	if pwd != "supersecret123" {
		t.Fatalf("expected 'supersecret123', got %q", pwd)
	}

	// 4. Delete password
	if err := store.DeletePassword("server-1"); err != nil {
		t.Fatalf("failed to delete password: %v", err)
	}

	pwd, err = store.GetPassword("server-1")
	if err != nil || pwd != "" {
		t.Fatalf("expected empty password after deletion, got %q (err=%v)", pwd, err)
	}
}

func TestCredentialStore_EncryptedVaultFallback(t *testing.T) {
	tmpDir := t.TempDir()
	store := &KeyringCredentialStore{
		logger:    zap.NewNop().Sugar(),
		configDir: tmpDir,
	}

	alias := "prod-bastion"
	secret := "P@ssw0rdSecure!"

	// 1. Set password directly into encrypted vault
	if err := store.setVaultPassword(alias, secret); err != nil {
		t.Fatalf("failed to set vault password: %v", err)
	}

	// 2. Check vault file permissions (0600)
	vaultFile := filepath.Join(tmpDir, vaultFileName)
	fi, err := os.Stat(vaultFile)
	if err != nil {
		t.Fatalf("vault file does not exist: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("expected vault file perm 0600, got %o", fi.Mode().Perm())
	}

	// 3. Ensure ciphertext does NOT contain plain secret
	rawBytes, err := os.ReadFile(vaultFile)
	if err != nil {
		t.Fatalf("failed reading vault file: %v", err)
	}
	if bytes.Contains(rawBytes, []byte(secret)) {
		t.Fatalf("SECURITY VIOLATION: plain secret found in vault ciphertext!")
	}

	// 4. Retrieve password
	retrieved, err := store.getVaultPassword(alias)
	if err != nil {
		t.Fatalf("failed getting vault password: %v", err)
	}
	if retrieved != secret {
		t.Fatalf("expected retrieved secret %q, got %q", secret, retrieved)
	}

	// 5. Delete password from vault
	if err := store.deleteVaultPassword(alias); err != nil {
		t.Fatalf("failed deleting vault password: %v", err)
	}
	deleted, err := store.getVaultPassword(alias)
	if err != nil || deleted != "" {
		t.Fatalf("expected empty password after deletion, got %q (err=%v)", deleted, err)
	}
}
