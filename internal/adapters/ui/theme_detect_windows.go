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

//go:build windows

package ui

import (
	"os/exec"
	"strings"
)

// detectOSTheme queries Windows registry for the system theme preference.
// Returns ThemeDark or ThemeLight. Defaults to ThemeDark if detection fails.
func detectOSTheme() string {
	cmd := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		"/v", "AppsUseLightTheme")

	output, err := cmd.Output()
	if err != nil {
		return ThemeDark
	}

	result := string(output)
	if strings.Contains(result, "0x1") {
		return ThemeLight
	}
	return ThemeDark
}
