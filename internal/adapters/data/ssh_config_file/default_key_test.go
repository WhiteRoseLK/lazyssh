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
	"testing"
)

func TestRepository_DefaultIdentityKey(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()
	main := "/home/u/.ssh/config"
	fs.write(main, "Host my-box\n    HostName 10.1.1.1\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	// Initially empty
	k, err := r.GetDefaultIdentityKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "" {
		t.Errorf("expected empty key initially, got %q", k)
	}

	// Save default key
	expectedKey := "~/.ssh/id_ed25519"
	if err := r.SaveDefaultIdentityKey(expectedKey); err != nil {
		t.Fatalf("SaveDefaultIdentityKey failed: %v", err)
	}

	// Retrieve default key
	k, err = r.GetDefaultIdentityKey()
	if err != nil {
		t.Fatalf("unexpected error getting default key: %v", err)
	}
	if k != expectedKey {
		t.Errorf("expected %q, got %q", expectedKey, k)
	}
}
