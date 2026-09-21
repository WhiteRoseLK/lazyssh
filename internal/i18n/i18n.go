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
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed locales/en.json
var enBytes []byte

//go:embed locales/fr.json
var frBytes []byte

//go:embed locales/zh-CN.json
var zhCNBytes []byte

const (
	LangEn   = "en"
	LangFr   = "fr"
	LangZhCN = "zh-CN"
)

var (
	mu          sync.RWMutex
	messages    map[string]string
	enFallback  map[string]string
	currentLang string
	initialized bool
	// defaultLang can be injected via -ldflags: -X github.com/WhiteRoseLK/neossh/internal/i18n.defaultLang=fr
	defaultLang string
)

// NormalizeLang converts user or system locale strings to a supported language code.
func NormalizeLang(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch {
	case strings.HasPrefix(raw, "fr") || raw == "french" || raw == "francais":
		return LangFr
	case strings.HasPrefix(raw, "zh") || raw == "cn" || raw == "chs" || raw == "chinese":
		return LangZhCN
	case strings.HasPrefix(raw, "en") || raw == "english":
		return LangEn
	default:
		return LangEn
	}
}

// Init initializes the i18n subsystem.
func Init(preferredLang string) {
	mu.Lock()
	defer mu.Unlock()

	// Load English fallback catalog
	enFallback = make(map[string]string)
	_ = json.Unmarshal(enBytes, &enFallback)

	targetLang := preferredLang
	if targetLang == "" {
		targetLang = os.Getenv("NEOSSH_LANG")
	}
	if targetLang == "" {
		targetLang = os.Getenv("LAZYSSH_LANG")
	}
	if targetLang == "" {
		targetLang = defaultLang
	}
	if targetLang == "" {
		targetLang = LangEn
	}

	loadLanguageLocked(NormalizeLang(targetLang))
	initialized = true
}

func loadLanguageLocked(lang string) {
	currentLang = lang
	messages = make(map[string]string)

	var data []byte
	switch lang {
	case LangFr:
		data = frBytes
	case LangZhCN:
		data = zhCNBytes
	default:
		data = enBytes
	}

	if err := json.Unmarshal(data, &messages); err != nil {
		messages = make(map[string]string)
	}
}

// SetLanguage changes the current active language at runtime.
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	loadLanguageLocked(NormalizeLang(lang))
	initialized = true
}

// GetLanguage returns the current active language code.
func GetLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	if !initialized {
		return LangEn
	}
	return currentLang
}

// SupportedLanguages returns list of supported language codes.
func SupportedLanguages() []string {
	return []string{LangEn, LangFr, LangZhCN}
}

// T translates a key into the current language, with formatting arguments.
func T(key string, args ...any) string {
	mu.RLock()
	if !initialized {
		mu.RUnlock()
		Init("")
		mu.RLock()
	}

	msg, ok := messages[key]
	if !ok || msg == "" {
		if fb, exists := enFallback[key]; exists && fb != "" {
			msg = fb
		} else {
			msg = key
		}
	}
	mu.RUnlock()

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}
