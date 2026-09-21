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
	"sync"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

func TestListServers_CacheServesAndInvalidationRefreshes(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, "Host a\n  HostName 1.1.1.1\n")
	r := newRepoForFS(t, fs, filepath.Join(t.TempDir(), "metadata.json"))

	list := func() []domain.Server {
		t.Helper()
		servers, err := r.ListServers("")
		if err != nil {
			t.Fatalf("ListServers: %v", err)
		}
		return servers
	}

	if got := list(); len(got) != 1 {
		t.Fatalf("initial: want 1 server, got %d", len(got))
	}

	// External change (bypasses the repository) must not be seen while cached.
	fs.write(main, "Host a\n  HostName 1.1.1.1\nHost b\n  HostName 2.2.2.2\n")
	if got := list(); len(got) != 1 {
		t.Fatalf("expected cached result within TTL, got %d servers", len(got))
	}

	// Explicit invalidation re-reads from disk.
	r.InvalidateCache()
	if got := list(); len(got) != 2 {
		t.Fatalf("after invalidate: want 2 servers, got %d", len(got))
	}
}

func TestListServers_MutationInvalidatesCache(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, "Host a\n  HostName 1.1.1.1\n")
	r := newRepoForFS(t, fs, filepath.Join(t.TempDir(), "metadata.json"))

	if got, err := r.ListServers(""); err != nil || len(got) != 1 {
		t.Fatalf("prime: got %d servers, err=%v", len(got), err)
	}

	if err := r.AddServer(domain.Server{Alias: "b", Host: "2.2.2.2", User: "u"}); err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	got, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("cache not invalidated by mutation: want 2 servers, got %d", len(got))
	}
}

func TestListServers_ConcurrentWithInvalidation(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, "Host a\n  HostName 1.1.1.1\n")
	r := newRepoForFS(t, fs, filepath.Join(t.TempDir(), "metadata.json"))

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = r.ListServers("")
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 50; j++ {
			r.InvalidateCache()
		}
	}()
	wg.Wait()
}
