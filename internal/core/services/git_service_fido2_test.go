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

package services

import (
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"go.uber.org/zap"
)

func TestDetectKeyTypeFromPubHeader(t *testing.T) {
	gs := &gitService{logger: zap.NewNop().Sugar()}

	tests := []struct {
		name      string
		header    string
		wantType  string
		wantFIDO2 bool
	}{
		{
			name:      "ssh-ed25519",
			header:    "ssh-ed25519",
			wantType:  "ed25519",
			wantFIDO2: false,
		},
		{
			name:      "ssh-rsa",
			header:    "ssh-rsa",
			wantType:  "rsa",
			wantFIDO2: false,
		},
		{
			name:      "ecdsa-sha2-nistp256",
			header:    "ecdsa-sha2-nistp256",
			wantType:  "ecdsa",
			wantFIDO2: false,
		},
		{
			name:      "ecdsa-sha2-nistp384",
			header:    "ecdsa-sha2-nistp384",
			wantType:  "ecdsa",
			wantFIDO2: false,
		},
		{
			name:      "ecdsa-sha2-nistp521",
			header:    "ecdsa-sha2-nistp521",
			wantType:  "ecdsa",
			wantFIDO2: false,
		},
		{
			name:      "ssh-dss",
			header:    "ssh-dss",
			wantType:  "dsa",
			wantFIDO2: false,
		},
		{
			name:      "sk-ssh-ed25519 FIDO2",
			header:    "sk-ssh-ed25519@openssh.com",
			wantType:  "ed25519",
			wantFIDO2: true,
		},
		{
			name:      "sk-ecdsa FIDO2",
			header:    "sk-ecdsa-sha2-nistp256@openssh.com",
			wantType:  "ecdsa",
			wantFIDO2: true,
		},
		{
			name:      "unknown header keeps original type",
			header:    "unknown-algo",
			wantType:  "rsa", // original type should be unchanged
			wantFIDO2: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &domain.SSHKey{Type: "rsa"} // default type before detection
			gs.detectKeyTypeFromPubHeader(key, tt.header)

			if key.Type != tt.wantType {
				t.Errorf("detectKeyTypeFromPubHeader(%q): Type = %q, want %q",
					tt.header, key.Type, tt.wantType)
			}
			if key.IsFIDO2 != tt.wantFIDO2 {
				t.Errorf("detectKeyTypeFromPubHeader(%q): IsFIDO2 = %v, want %v",
					tt.header, key.IsFIDO2, tt.wantFIDO2)
			}
		})
	}
}

func TestParseAgentKeyLine_FIDO2(t *testing.T) {
	gs := &gitService{logger: zap.NewNop().Sugar()}

	tests := []struct {
		name      string
		line      string
		wantType  string
		wantFIDO2 bool
		wantNil   bool
	}{
		{
			name:      "standard ed25519",
			line:      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA user@host",
			wantType:  "ed25519",
			wantFIDO2: false,
		},
		{
			name:      "standard rsa",
			line:      "ssh-rsa AAAAB3NzaC1yc2EAAAA user@host",
			wantType:  "rsa",
			wantFIDO2: false,
		},
		{
			name:      "FIDO2 ed25519",
			line:      "sk-ssh-ed25519@openssh.com AAAAC3NzaC1lZDI1NTE5AAAA user@yubikey",
			wantType:  "ed25519",
			wantFIDO2: true,
		},
		{
			name:      "FIDO2 ecdsa",
			line:      "sk-ecdsa-sha2-nistp256@openssh.com AAAAC3NzaC1lY2RzYXNoYQ== user@solokey",
			wantType:  "ecdsa",
			wantFIDO2: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:    "single field",
			line:    "ssh-rsa",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := gs.parseAgentKeyLine(tt.line)
			if tt.wantNil {
				if key != nil {
					t.Errorf("parseAgentKeyLine(%q) = %+v, want nil", tt.line, key)
				}
				return
			}
			if key == nil {
				t.Fatalf("parseAgentKeyLine(%q) = nil, want non-nil", tt.line)
			}
			if key.Type != tt.wantType {
				t.Errorf("parseAgentKeyLine(%q): Type = %q, want %q",
					tt.line, key.Type, tt.wantType)
			}
			if key.IsFIDO2 != tt.wantFIDO2 {
				t.Errorf("parseAgentKeyLine(%q): IsFIDO2 = %v, want %v",
					tt.line, key.IsFIDO2, tt.wantFIDO2)
			}
			if !key.LoadedInAgent {
				t.Errorf("parseAgentKeyLine(%q): LoadedInAgent should be true", tt.line)
			}
		})
	}
}

func TestDetectKeyTypeAndEncryption_FIDO2Filename(t *testing.T) {
	gs := &gitService{logger: zap.NewNop().Sugar()}

	tests := []struct {
		name     string
		keyName  string
		keyPath  string
		content  string
		wantType string
	}{
		{
			name:     "openssh ed25519 by name",
			keyName:  "id_ed25519",
			keyPath:  "/home/user/.ssh/id_ed25519",
			content:  "-----BEGIN OPENSSH PRIVATE KEY-----\ndata\n-----END OPENSSH PRIVATE KEY-----",
			wantType: "ed25519",
		},
		{
			name:     "openssh ecdsa by name",
			keyName:  "id_ecdsa",
			keyPath:  "/home/user/.ssh/id_ecdsa",
			content:  "-----BEGIN OPENSSH PRIVATE KEY-----\ndata\n-----END OPENSSH PRIVATE KEY-----",
			wantType: "ecdsa",
		},
		{
			name:     "openssh sk-ed25519 by name",
			keyName:  "id_ed25519_sk",
			keyPath:  "/home/user/.ssh/id_ed25519_sk",
			content:  "-----BEGIN OPENSSH PRIVATE KEY-----\ndata\n-----END OPENSSH PRIVATE KEY-----",
			wantType: "ed25519",
		},
		{
			name:     "rsa private key",
			keyName:  "id_rsa",
			keyPath:  "/home/user/.ssh/id_rsa",
			content:  "-----BEGIN RSA PRIVATE KEY-----\ndata\n-----END RSA PRIVATE KEY-----",
			wantType: "rsa",
		},
		{
			name:     "dsa private key",
			keyName:  "id_dsa",
			keyPath:  "/home/user/.ssh/id_dsa",
			content:  "-----BEGIN DSA PRIVATE KEY-----\ndata\n-----END DSA PRIVATE KEY-----",
			wantType: "dsa",
		},
		{
			name:     "ec private key",
			keyName:  "id_ecdsa",
			keyPath:  "/home/user/.ssh/id_ecdsa",
			content:  "-----BEGIN EC PRIVATE KEY-----\ndata\n-----END EC PRIVATE KEY-----",
			wantType: "ecdsa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &domain.SSHKey{
				Name: tt.keyName,
				Path: tt.keyPath,
			}
			gs.detectKeyTypeAndEncryption(key, tt.content)

			if key.Type != tt.wantType {
				t.Errorf("detectKeyTypeAndEncryption(): Type = %q, want %q",
					key.Type, tt.wantType)
			}
		})
	}
}
