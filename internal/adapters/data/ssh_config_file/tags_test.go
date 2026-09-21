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
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

func TestExtractTagsFromComment(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected []string
	}{
		{
			name:     "standard comma-separated tags",
			comment:  " tags: prod, database",
			expected: []string{"prod", "database"},
		},
		{
			name:     "no space after colon",
			comment:  "tags:prod,database",
			expected: []string{"prod", "database"},
		},
		{
			name:     "singular tag keyword",
			comment:  " tag: prod, database",
			expected: []string{"prod", "database"},
		},
		{
			name:     "case-insensitive TAGS",
			comment:  " TAGS: PROD, DB",
			expected: []string{"PROD", "DB"},
		},
		{
			name:     "equals syntax",
			comment:  " tags = prod, database",
			expected: []string{"prod", "database"},
		},
		{
			name:     "prefixed by Added by neossh with hash",
			comment:  " Added by neossh # tags: prod, database",
			expected: []string{"prod", "database"},
		},
		{
			name:     "prefixed by Added by neossh with comma",
			comment:  " Added by neossh, tags: prod, database",
			expected: []string{"prod", "database"},
		},
		{
			name:     "bracket enclosed tags",
			comment:  " [tags: prod, database]",
			expected: []string{"prod", "database"},
		},
		{
			name:     "trailing comment after tags",
			comment:  " tags: prod, database # secondary note",
			expected: []string{"prod", "database"},
		},
		{
			name:     "deduplicate tags preserving order",
			comment:  " tags: prod, database, prod",
			expected: []string{"prod", "database"},
		},
		{
			name:     "space-separated tags without comma",
			comment:  " tags: prod database staging",
			expected: []string{"prod", "database", "staging"},
		},
		{
			name:     "single tag",
			comment:  " tags: production",
			expected: []string{"production"},
		},
		{
			name:     "unrelated comment without tags",
			comment:  " Just a normal comment line",
			expected: nil,
		},
		{
			name:     "empty comment",
			comment:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTagsFromComment(tt.comment)
			if !slices.Equal(got, tt.expected) {
				t.Errorf("extractTagsFromComment(%q) = %v, expected %v", tt.comment, got, tt.expected)
			}
		})
	}
}

func TestReplaceTagsInComment(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		newTags  []string
		expected string
	}{
		{
			name:     "replace standard tags",
			comment:  " tags: prod, db",
			newTags:  []string{"staging", "web"},
			expected: " tags: staging, web",
		},
		{
			name:     "replace with neossh prefix",
			comment:  " Added by neossh # tags: prod",
			newTags:  []string{"production"},
			expected: " Added by neossh # tags: production",
		},
		{
			name:     "replace with trailing comment",
			comment:  " tags: prod # keep this note",
			newTags:  []string{"staging"},
			expected: " tags: staging # keep this note",
		},
		{
			name:     "replace in empty comment",
			comment:  "",
			newTags:  []string{"newtag"},
			expected: " tags: newtag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceTagsInComment(tt.comment, tt.newTags)
			if got != tt.expected {
				t.Errorf("replaceTagsInComment(%q, %v) = %q, expected %q", tt.comment, tt.newTags, got, tt.expected)
			}
		})
	}
}

func TestRemoveTagsFromComment(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected string
	}{
		{
			name:     "remove pure tags comment",
			comment:  " tags: prod, db",
			expected: "",
		},
		{
			name:     "remove with neossh prefix",
			comment:  " Added by neossh # tags: prod, db",
			expected: " Added by neossh",
		},
		{
			name:     "remove with trailing note",
			comment:  " tags: prod, db # note",
			expected: " note",
		},
		{
			name:     "comment with no tags unchanged",
			comment:  " Production server",
			expected: " Production server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTagsFromComment(tt.comment)
			if got != tt.expected {
				t.Errorf("removeTagsFromComment(%q) = %q, expected %q", tt.comment, got, tt.expected)
			}
		})
	}
}

func TestListServers_ReadsTagsFromConfigComments(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	configContent := `Host server-inline # tags: prod, database
    HostName 10.0.0.1
    User ubuntu

Host server-block
    # tags: staging, cache
    HostName 10.0.0.2
    User dev

Host server-kv
    HostName 10.0.0.3 # tags: monitoring
    User admin
`
	fs.write(main, configContent)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}

	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	expectedTags := map[string][]string{
		"server-inline": {"prod", "database"},
		"server-block":  {"staging", "cache"},
		"server-kv":     {"monitoring"},
	}

	for _, s := range servers {
		exp, ok := expectedTags[s.Alias]
		if !ok {
			t.Fatalf("unexpected server alias: %s", s.Alias)
		}
		if !slices.Equal(s.Tags, exp) {
			t.Errorf("server %s tags = %v, expected %v", s.Alias, s.Tags, exp)
		}
	}
}

func TestAddServer_WritesTagsToConfig(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, "")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	newServer := domain.Server{
		Alias: "tagged-srv",
		Host:  "192.168.1.50",
		User:  "root",
		Port:  22,
		Tags:  []string{"prod", "web"},
	}

	if err := r.AddServer(newServer); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "# tags: prod, web") {
		t.Errorf("expected config to contain '# tags: prod, web', got:\n%s", content)
	}

	// Verify reload from disk
	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if !slices.Equal(servers[0].Tags, []string{"prod", "web"}) {
		t.Errorf("expected tags [prod web], got %v", servers[0].Tags)
	}
}

func TestUpdateServer_UpdatesTagsInConfig(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, `Host my-server    # tags: prod, web
    HostName 10.0.0.1
    User ubuntu
`)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	// 1. Update tags to ["production", "db"]
	oldServer := servers[0]
	updatedServer := oldServer
	updatedServer.Tags = []string{"production", "db"}

	if err := r.UpdateServer(oldServer, updatedServer); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "# tags: production, db") {
		t.Errorf("expected config to contain updated tags '# tags: production, db', got:\n%s", content)
	}

	// Reload and check
	servers, err = r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if !slices.Equal(servers[0].Tags, []string{"production", "db"}) {
		t.Errorf("expected updated tags, got %v", servers[0].Tags)
	}

	// 2. Clear all tags
	clearedServer := servers[0]
	clearedServer.Tags = []string{}

	if err := r.UpdateServer(servers[0], clearedServer); err != nil {
		t.Fatalf("UpdateServer clear tags failed: %v", err)
	}

	contentAfterClear := fs.read(main)
	if strings.Contains(contentAfterClear, "tags:") {
		t.Errorf("expected tags to be removed from config, got:\n%s", contentAfterClear)
	}

	// Reload and check empty
	servers, err = r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers[0].Tags) != 0 {
		t.Errorf("expected 0 tags after clear, got %v", servers[0].Tags)
	}
}

func TestUpdateServer_PreservesBlockCommentTags(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, `Host block-srv
    # tags: dev
    HostName 10.0.0.1
`)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	oldServer := servers[0]
	updatedServer := oldServer
	updatedServer.Tags = []string{"dev", "frontend"}

	if err := r.UpdateServer(oldServer, updatedServer); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "# tags: dev, frontend") {
		t.Errorf("expected config to update block comment '# tags: dev, frontend', got:\n%s", content)
	}
}

func TestBackwardsCompatibility_MetadataJSON(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	// SSH config has NO tag comment
	fs.write(main, `Host legacy-srv
    HostName 10.0.0.10
    User admin
`)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	// Pre-populate metadata.json with tags
	existingMeta := map[string]ServerMetadata{
		"legacy-srv": {
			Tags:     []string{"legacy-tag-1", "legacy-tag-2"},
			SSHCount: 5,
		},
	}
	metaBytes, _ := json.Marshal(existingMeta)
	_ = os.WriteFile(tmpMeta, metaBytes, 0o600)

	r := newRepoForFS(t, fs, tmpMeta)

	// ListServers should load tags from metadata.json
	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if !slices.Equal(servers[0].Tags, []string{"legacy-tag-1", "legacy-tag-2"}) {
		t.Errorf("expected tags from metadata.json, got %v", servers[0].Tags)
	}

	// When updating the server, tags should now be written to the SSH config comment
	oldServer := servers[0]
	updatedServer := oldServer
	updatedServer.Port = 2222

	if err := r.UpdateServer(oldServer, updatedServer); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	content := fs.read(main)
	if !strings.Contains(content, "# tags: legacy-tag-1, legacy-tag-2") {
		t.Errorf("expected SSH config to now contain tags comment, got:\n%s", content)
	}
}

func TestSSHConfigTags_TakePrecedenceOverMetadataJSON(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	// SSH config has tags
	fs.write(main, `Host sync-srv    # tags: synced-tag
    HostName 10.0.0.20
`)

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	// metadata.json has old stale tags
	existingMeta := map[string]ServerMetadata{
		"sync-srv": {
			Tags: []string{"stale-local-tag"},
		},
	}
	metaBytes, _ := json.Marshal(existingMeta)
	_ = os.WriteFile(tmpMeta, metaBytes, 0o600)

	r := newRepoForFS(t, fs, tmpMeta)

	servers, err := r.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	// SSH config tags must take precedence, not stale metadata
	if !slices.Equal(servers[0].Tags, []string{"synced-tag"}) {
		t.Errorf("expected SSH config tags [synced-tag], got %v", servers[0].Tags)
	}
}
