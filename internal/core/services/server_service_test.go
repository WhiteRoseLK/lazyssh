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
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestCopySSHKeyMissingBinary(t *testing.T) {
	// Temporarily empty PATH to simulate missing ssh-copy-id binary
	origPath := os.Getenv("PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", origPath)
	})
	_ = os.Setenv("PATH", "")

	logger := zap.NewNop().Sugar()
	svc := &serverService{
		logger: logger,
	}

	err := svc.CopySSHKey("test-server")
	if err == nil {
		t.Fatal("expected error when ssh-copy-id is not in PATH, got nil")
	}
}
