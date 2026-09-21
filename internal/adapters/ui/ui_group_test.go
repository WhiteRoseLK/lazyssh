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
	"time"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

func TestServerList_NestedGroupsAndCollapsing(t *testing.T) {
	list := NewServerList()

	servers := []domain.Server{
		{Alias: "pinned-1", Host: "1.1.1.1", PinnedAt: time.Now()},
		{Alias: "web-1", Host: "10.0.0.1", Group: "Prod/Web"},
		{Alias: "web-2", Host: "10.0.0.2", Group: "Prod/Web"},
		{Alias: "db-1", Host: "10.0.0.3", Group: "Prod/DB"},
		{Alias: "stage-1", Host: "10.0.1.1", Group: "Staging"},
		{Alias: "solo-1", Host: "10.0.2.1"}, // ungrouped
	}

	sortServersForUI(servers, SortByAliasAsc)
	list.UpdateServers(servers)

	if list.GetItemCount() == 0 {
		t.Fatalf("expected non-empty list item count")
	}

	// Verify headers were created
	foundProdHeader := false
	foundWebHeader := false
	for _, h := range list.displayedHeaders {
		if h == "Prod" {
			foundProdHeader = true
		}
		if h == "Prod/Web" {
			foundWebHeader = true
		}
	}
	if !foundProdHeader || !foundWebHeader {
		t.Fatalf("expected group headers 'Prod' and 'Prod/Web', got: %v", list.displayedHeaders)
	}

	// Test collapsing Prod
	initialCount := list.GetItemCount()
	list.collapsedGroups["Prod"] = true
	list.UpdateServers(servers)
	collapsedCount := list.GetItemCount()

	if collapsedCount >= initialCount {
		t.Fatalf("expected item count after collapse (%d) to be strictly less than initial (%d)", collapsedCount, initialCount)
	}

	// Ensure servers under Prod are hidden
	for _, item := range list.displayedItems {
		if item != nil && (item.Alias == "web-1" || item.Alias == "db-1") {
			t.Fatalf("expected server %s to be hidden when Prod is collapsed", item.Alias)
		}
	}
}

func TestGroupAutocomplete(t *testing.T) {
	form := NewServerForm(ServerFormAdd, nil)
	form.SetExistingGroups([]string{"Production/Web", "Production/DB", "Staging", "QA"})

	autocomplete := form.createGroupAutocomplete()

	// Empty query returns nil to allow tab navigation
	if res := autocomplete(""); res != nil {
		t.Fatalf("expected nil for empty autocomplete input, got %v", res)
	}

	// Substring / fuzzy match
	res := autocomplete("prod")
	if len(res) != 2 {
		t.Fatalf("expected 2 matches for 'prod', got %d: %v", len(res), res)
	}

	resStage := autocomplete("stag")
	if len(resStage) != 1 || resStage[0] != "Staging" {
		t.Fatalf("expected 'Staging' for 'stag', got %v", resStage)
	}
}

func TestBuildTmuxCommand(t *testing.T) {
	groupServers := []domain.Server{
		{Alias: "srv-a", Host: "10.0.0.1"},
		{Alias: "srv-b", Host: "10.0.0.2"},
	}

	cmd := buildTmuxCommand("neossh-test-session", groupServers)

	if !strings.Contains(cmd, "tmux new-session -d -s 'neossh-test-session'") {
		t.Errorf("expected tmux new-session command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "tmux split-window -t 'neossh-test-session'") {
		t.Errorf("expected tmux split-window command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "tmux select-layout -t 'neossh-test-session' tiled") {
		t.Errorf("expected tiled layout command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "tmux attach-session -t 'neossh-test-session'") {
		t.Errorf("expected attach-session command, got: %s", cmd)
	}
}
