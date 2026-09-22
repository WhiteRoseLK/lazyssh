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

package ports

// CredentialStore provides secure storage for server passwords and secrets,
// backed by the OS native keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager)
// or an encrypted AES-GCM local vault.
type CredentialStore interface {
	// GetPassword retrieves the password for the given server alias.
	// Returns ("", nil) if no password is stored for the alias.
	GetPassword(alias string) (string, error)

	// SetPassword securely stores the password for the given server alias.
	SetPassword(alias, password string) error

	// DeletePassword removes the stored password for the given server alias.
	DeletePassword(alias string) error
}
