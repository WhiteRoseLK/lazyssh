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

func TestParseSearchQuery_Filters(t *testing.T) {
	srv1 := domain.Server{
		Alias:      "web-prod-01",
		Host:       "192.168.1.10",
		User:       "ubuntu",
		Port:       22,
		Tags:       []string{"prod", "web", "frontend"},
		Group:      "production",
		PingStatus: "up",
	}

	srv2 := domain.Server{
		Alias:      "db-prod-01",
		Host:       "192.168.1.20",
		User:       "postgres",
		Port:       5432,
		Tags:       []string{"prod", "database"},
		Group:      "production",
		PingStatus: "down",
	}

	srv3 := domain.Server{
		Alias:      "bastion-staging",
		Host:       "staging.example.com",
		User:       "root",
		Port:       2222,
		Tags:       []string{"staging", "bastion"},
		Group:      "staging",
		PingStatus: "checking",
	}

	tests := []struct {
		name     string
		query    string
		server   domain.Server
		expected bool
	}{
		// Plain free terms
		{"Free term matches alias", "web", srv1, true},
		{"Free term no match", "web", srv2, false},
		{"Free term matches tag", "frontend", srv1, true},

		// Tag filter
		{"Tag filter matches", "tag:prod", srv1, true},
		{"Tag filter matches db", "tag:database", srv2, true},
		{"Tag filter no match", "tag:database", srv1, false},
		{"Tags alias matches", "tags:staging", srv3, true},
		{"Negative tag filter matches", "-tag:staging", srv1, true},
		{"Negative tag filter rejects", "-tag:staging", srv3, false},

		// User filter
		{"User filter matches", "user:ubuntu", srv1, true},
		{"User filter partial match", "user:post", srv2, true},
		{"User filter no match", "user:root", srv1, false},
		{"Negative user filter", "-user:root", srv1, true},
		{"Negative user filter rejects", "-user:root", srv3, false},

		// Host filter
		{"Host filter matches IP", "host:192.168.1.", srv1, true},
		{"Host filter matches domain", "host:staging.example", srv3, true},
		{"Negative host filter", "-host:example.com", srv1, true},
		{"Negative host filter rejects", "-host:example.com", srv3, false},

		// Port filter
		{"Port filter matches standard port", "port:22", srv1, true},
		{"Port filter matches non-standard port", "port:5432", srv2, true},
		{"Port filter no match", "port:22", srv2, false},
		{"Negative port filter", "-port:5432", srv1, true},
		{"Negative port filter rejects", "-port:5432", srv2, false},

		// Status filter
		{"Status up matches", "status:up", srv1, true},
		{"Status online alias matches", "status:online", srv1, true},
		{"Status down matches", "status:down", srv2, true},
		{"Status offline alias matches", "status:offline", srv2, true},
		{"Status unknown matches", "status:unknown", srv3, true},
		{"Negative status down matches", "-status:down", srv1, true},
		{"Negative status down rejects", "-status:down", srv2, false},

		// Group filter
		{"Group filter matches", "group:production", srv1, true},
		{"Negative group filter", "-group:staging", srv1, true},
		{"Negative group filter rejects", "-group:staging", srv3, false},

		// Combined filter and free terms
		{"Combined tag and user", "tag:prod user:ubuntu", srv1, true},
		{"Combined tag and user no match", "tag:prod user:root", srv1, false},
		{"Combined tag, status, and free term", "tag:prod status:up web", srv1, true},
		{"Combined tag and wrong status", "tag:prod status:down web", srv1, false},
		{"Combined tag and negative tag", "tag:prod -tag:database", srv1, true},
		{"Combined tag and negative tag rejects", "tag:prod -tag:database", srv2, false},

		// Quoted values
		{"Quoted tag term", `tag:"web"`, srv1, true},
		{"Quoted free term", `"web-prod"`, srv1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := parseSearchQuery(tt.query)
			got := filter.Matches(tt.server)
			if got != tt.expected {
				t.Errorf("parseSearchQuery(%q).Matches() = %v, want %v", tt.query, got, tt.expected)
			}
		})
	}
}

func TestServerService_ListServers_AdvancedFilters(t *testing.T) {
	mockRepo := &mockServerRepository{
		servers: []domain.Server{
			{
				Alias:      "web-prod-01",
				Host:       "10.0.0.1",
				User:       "ubuntu",
				Port:       22,
				Tags:       []string{"prod", "web"},
				Group:      "production",
				PingStatus: "up",
			},
			{
				Alias:      "web-prod-02",
				Host:       "10.0.0.2",
				User:       "ubuntu",
				Port:       22,
				Tags:       []string{"prod", "web"},
				Group:      "production",
				PingStatus: "down",
			},
			{
				Alias:      "db-prod-01",
				Host:       "10.0.0.3",
				User:       "postgres",
				Port:       5432,
				Tags:       []string{"prod", "db"},
				Group:      "production",
				PingStatus: "up",
			},
			{
				Alias:      "staging-api",
				Host:       "10.0.1.1",
				User:       "node",
				Port:       3000,
				Tags:       []string{"staging", "api"},
				Group:      "staging",
				PingStatus: "up",
			},
		},
	}

	logger := zap.NewNop().Sugar()
	svc := NewServerService(logger, mockRepo)

	// Filter by tag
	prodServers, err := svc.ListServers("tag:prod")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(prodServers) != 3 {
		t.Errorf("expected 3 prod servers, got %d", len(prodServers))
	}

	// Filter by tag and status
	prodUp, err := svc.ListServers("tag:prod status:up")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(prodUp) != 2 {
		t.Errorf("expected 2 prod up servers, got %d", len(prodUp))
	}

	// Filter by user
	ubuntuServers, err := svc.ListServers("user:ubuntu")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(ubuntuServers) != 2 {
		t.Errorf("expected 2 ubuntu servers, got %d", len(ubuntuServers))
	}

	// Filter by port
	customPortServers, err := svc.ListServers("port:5432")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(customPortServers) != 1 || customPortServers[0].Alias != "db-prod-01" {
		t.Errorf("expected db-prod-01, got %v", customPortServers)
	}

	// Negative filter
	nonProdServers, err := svc.ListServers("-group:production")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(nonProdServers) != 1 || nonProdServers[0].Alias != "staging-api" {
		t.Errorf("expected staging-api, got %v", nonProdServers)
	}

	// Combined filter with free search term
	webProdUp, err := svc.ListServers("tag:prod status:up web")
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(webProdUp) != 1 || webProdUp[0].Alias != "web-prod-01" {
		t.Errorf("expected web-prod-01, got %v", webProdUp)
	}
}
