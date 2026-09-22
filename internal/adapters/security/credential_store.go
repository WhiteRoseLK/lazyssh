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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/WhiteRoseLK/neossh/internal/core/ports"
	"github.com/zalando/go-keyring"
	"go.uber.org/zap"
)

const (
	keyringServiceName = "neossh"
	vaultFileName      = "vault.enc"
	vaultKeyFileName   = ".vault_key"
)

// KeyringCredentialStore securely stores credentials in the OS keyring,
// with an encrypted AES-256-GCM local vault fallback for headless or restricted environments.
type KeyringCredentialStore struct {
	logger    *zap.SugaredLogger
	configDir string
	mu        sync.RWMutex
}

// NewCredentialStore creates a new secure credential store.
func NewCredentialStore(logger *zap.SugaredLogger, configDir string) ports.CredentialStore {
	return &KeyringCredentialStore{
		logger:    logger,
		configDir: configDir,
	}
}

// GetPassword retrieves the password for a server alias.
func (s *KeyringCredentialStore) GetPassword(alias string) (string, error) {
	if alias == "" {
		return "", nil
	}

	// 1. Try OS Keyring
	pwd, err := keyring.Get(keyringServiceName, alias)
	if err == nil {
		return pwd, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		// Keyring is functional, but password does not exist
		return "", nil
	}

	// 2. Keyring failed (e.g. headless environment / no D-Bus), check encrypted fallback vault
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getVaultPassword(alias)
}

// SetPassword securely stores a password for a server alias.
func (s *KeyringCredentialStore) SetPassword(alias, password string) error {
	if alias == "" {
		return fmt.Errorf("server alias cannot be empty")
	}

	if password == "" {
		return s.DeletePassword(alias)
	}

	// 1. Try OS Keyring
	err := keyring.Set(keyringServiceName, alias, password)
	if err == nil {
		return nil
	}

	// 2. Keyring unavailable, save to encrypted fallback vault
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setVaultPassword(alias, password)
}

// DeletePassword removes a stored password for a server alias.
func (s *KeyringCredentialStore) DeletePassword(alias string) error {
	if alias == "" {
		return nil
	}

	var keyringErr error
	if err := keyring.Delete(keyringServiceName, alias); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		keyringErr = err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	vaultErr := s.deleteVaultPassword(alias)

	if keyringErr != nil && vaultErr != nil {
		return fmt.Errorf("failed to delete password: %w", errors.Join(keyringErr, vaultErr))
	}
	return nil
}

// --- Encrypted Local Vault (AES-256-GCM) Fallback ---

func (s *KeyringCredentialStore) getOrCreateKey() ([]byte, error) {
	keyPath := filepath.Clean(filepath.Join(s.configDir, vaultKeyFileName))
	if data, err := os.ReadFile(keyPath); err == nil && len(data) == 32 {
		return data, nil
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate vault key: %w", err)
	}

	if err := os.MkdirAll(s.configDir, 0o700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("write vault key: %w", err)
	}
	return key, nil
}

func (s *KeyringCredentialStore) loadVault() (map[string]string, error) {
	vaultPath := filepath.Clean(filepath.Join(s.configDir, vaultFileName))
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	if len(data) < 12 {
		return make(map[string]string), nil
	}

	key, err := s.getOrCreateKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("vault data too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt vault: %w", err)
	}

	var vault map[string]string
	if err := json.Unmarshal(plaintext, &vault); err != nil {
		return nil, fmt.Errorf("unmarshal vault: %w", err)
	}
	return vault, nil
}

func (s *KeyringCredentialStore) saveVault(vault map[string]string) error {
	plaintext, err := json.Marshal(vault)
	if err != nil {
		return fmt.Errorf("marshal vault: %w", err)
	}

	key, err := s.getOrCreateKey()
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	vaultPath := filepath.Clean(filepath.Join(s.configDir, vaultFileName))

	return os.WriteFile(vaultPath, ciphertext, 0o600)
}

func (s *KeyringCredentialStore) getVaultPassword(alias string) (string, error) {
	vault, err := s.loadVault()
	if err != nil {
		return "", err
	}
	return vault[alias], nil
}

func (s *KeyringCredentialStore) setVaultPassword(alias, password string) error {
	vault, err := s.loadVault()
	if err != nil {
		return err
	}
	vault[alias] = password
	return s.saveVault(vault)
}

func (s *KeyringCredentialStore) deleteVaultPassword(alias string) error {
	vault, err := s.loadVault()
	if err != nil {
		return err
	}
	if _, ok := vault[alias]; !ok {
		return nil
	}
	delete(vault, alias)
	return s.saveVault(vault)
}
