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
	"reflect"
	"testing"
	"time"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

func TestCloneServer_DeepCopiesSlices(t *testing.T) {
	orig := domain.Server{
		Alias:          "prod-server",
		Host:           "10.0.0.1",
		User:           "admin",
		Port:           2222,
		PinnedAt:       time.Now(),
		LastSeen:       time.Now(),
		PingStatus:     StatusUp,
		PingLatency:    25 * time.Millisecond,
		IdentityFiles:  []string{"~/.ssh/id_rsa", "~/.ssh/id_ed25519"},
		Tags:           []string{"production", "web"},
		LocalForward:   []string{"8080:localhost:80"},
		RemoteForward:  []string{"9090:localhost:9090"},
		DynamicForward: []string{"1080"},
		SendEnv:        []string{"LANG", "LC_*"},
		SetEnv:         []string{"FOO=bar"},
	}

	cloned := cloneServer(orig)

	// Verify values are equal
	if cloned.Alias != orig.Alias || cloned.Host != orig.Host || cloned.User != orig.User || cloned.Port != orig.Port {
		t.Fatalf("Cloned scalar fields mismatch: got %+v, want %+v", cloned, orig)
	}

	// Slices should have equal elements
	if !reflect.DeepEqual(cloned.IdentityFiles, orig.IdentityFiles) {
		t.Errorf("IdentityFiles mismatch")
	}
	if !reflect.DeepEqual(cloned.Tags, orig.Tags) {
		t.Errorf("Tags mismatch")
	}
	if !reflect.DeepEqual(cloned.LocalForward, orig.LocalForward) {
		t.Errorf("LocalForward mismatch")
	}
	if !reflect.DeepEqual(cloned.RemoteForward, orig.RemoteForward) {
		t.Errorf("RemoteForward mismatch")
	}
	if !reflect.DeepEqual(cloned.DynamicForward, orig.DynamicForward) {
		t.Errorf("DynamicForward mismatch")
	}
	if !reflect.DeepEqual(cloned.SendEnv, orig.SendEnv) {
		t.Errorf("SendEnv mismatch")
	}
	if !reflect.DeepEqual(cloned.SetEnv, orig.SetEnv) {
		t.Errorf("SetEnv mismatch")
	}

	// Verify slices are distinct memory allocations (modifying clone does not mutate original)
	cloned.IdentityFiles[0] = "~/.ssh/other_key"
	if orig.IdentityFiles[0] == "~/.ssh/other_key" {
		t.Errorf("IdentityFiles slice was not deep-copied")
	}

	cloned.Tags[0] = "staging"
	if orig.Tags[0] == "staging" {
		t.Errorf("Tags slice was not deep-copied")
	}

	cloned.LocalForward[0] = "8081:localhost:80"
	if orig.LocalForward[0] == "8081:localhost:80" {
		t.Errorf("LocalForward slice was not deep-copied")
	}
}

func TestCloneServer_AliasDeduplication(t *testing.T) {
	existingAliases := []string{"web", "web_1", "db"}

	alias := GenerateUniqueAlias("web", existingAliases)
	if alias != "web_2" {
		t.Errorf("expected alias 'web_2', got '%s'", alias)
	}

	aliasStandalone := GenerateUniqueAlias("db", existingAliases)
	if aliasStandalone != "db_1" {
		t.Errorf("expected alias 'db_1', got '%s'", aliasStandalone)
	}
}
