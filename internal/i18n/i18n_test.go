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

package i18n

import (
	"sync"
	"testing"
)

func TestI18n_NormalizeLang(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"en", LangEn},
		{"EN-US", LangEn},
		{"english", LangEn},
		{"fr", LangFr},
		{"FR-FR", LangFr},
		{"french", LangFr},
		{"francais", LangFr},
		{"zh", LangZhCN},
		{"zh-CN", LangZhCN},
		{"cn", LangZhCN},
		{"chinese", LangZhCN},
		{"unknown", LangEn},
	}

	for _, tc := range tests {
		got := NormalizeLang(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeLang(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestI18n_Translation_And_Fallback(t *testing.T) {
	Init("en")
	if GetLanguage() != LangEn {
		t.Fatalf("expected language %s, got %s", LangEn, GetLanguage())
	}

	enTitle := T("app.title_servers")
	if enTitle != " 1 Servers " {
		t.Errorf("unexpected en title: %s", enTitle)
	}

	SetLanguage("fr")
	if GetLanguage() != LangFr {
		t.Fatalf("expected language %s, got %s", LangFr, GetLanguage())
	}
	frTitle := T("app.title_servers")
	if frTitle != " 1 Serveurs " {
		t.Errorf("unexpected fr title: %s", frTitle)
	}

	SetLanguage("zh-CN")
	if GetLanguage() != LangZhCN {
		t.Fatalf("expected language %s, got %s", LangZhCN, GetLanguage())
	}
	zhTitle := T("app.title_servers")
	if zhTitle != " 1 服务器列表 " {
		t.Errorf("unexpected zh title: %s", zhTitle)
	}

	// Test fallback to English when key missing in Chinese
	fallback := T("unknown.key.fallback")
	if fallback != "unknown.key.fallback" {
		t.Errorf("expected missing key to fallback to key name, got: %s", fallback)
	}

	// Test formatting args
	SetLanguage("en")
	formatted := T("dialog.confirm_delete", "my-host")
	if formatted != "Are you sure you want to delete server 'my-host'?" {
		t.Errorf("unexpected formatted string: %s", formatted)
	}
}

func TestI18n_Concurrency(t *testing.T) {
	Init("en")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			switch idx % 3 {
			case 0:
				SetLanguage("fr")
			case 1:
				SetLanguage("zh-CN")
			default:
				SetLanguage("en")
			}
			_ = T("app.title_servers")
			_ = GetLanguage()
		}(i)
	}
	wg.Wait()
}
