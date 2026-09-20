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

package domain

import (
	"path/filepath"
	"testing"
)

func TestToTildePathWithHome(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		homeDir  string
		goos     string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			homeDir:  "/home/user",
			goos:     "linux",
			expected: "",
		},
		{
			name:     "already tilde path",
			path:     "~/.ssh/id_rsa",
			homeDir:  "/home/user",
			goos:     "linux",
			expected: "~/.ssh/id_rsa",
		},
		{
			name:     "already tilde path windows slash",
			path:     `~/.ssh/id_rsa`,
			homeDir:  `C:\Users\user`,
			goos:     "windows",
			expected: "~/.ssh/id_rsa",
		},
		{
			name:     "exact home dir linux",
			path:     "/home/user",
			homeDir:  "/home/user",
			goos:     "linux",
			expected: "~",
		},
		{
			name:     "linux ssh key in home",
			path:     "/home/user/.ssh/id_ed25519",
			homeDir:  "/home/user",
			goos:     "linux",
			expected: "~/.ssh/id_ed25519",
		},
		{
			name:     "macos ssh key in home",
			path:     "/Users/mathieu/.ssh/custom_key",
			homeDir:  "/Users/mathieu",
			goos:     "darwin",
			expected: "~/.ssh/custom_key",
		},
		{
			name:     "windows ssh key in home with backslashes",
			path:     `C:\Users\User\.ssh\id_rsa`,
			homeDir:  `C:\Users\User`,
			goos:     "windows",
			expected: "~/.ssh/id_rsa",
		},
		{
			name:     "windows ssh key case-insensitive",
			path:     `c:\users\user\.ssh\id_rsa`,
			homeDir:  `C:\Users\User`,
			goos:     "windows",
			expected: "~/.ssh/id_rsa",
		},
		{
			name:     "windows ssh key forward slashes",
			path:     "C:/Users/User/.ssh/id_rsa",
			homeDir:  `C:\Users\User`,
			goos:     "windows",
			expected: "~/.ssh/id_rsa",
		},
		{
			name:     "unrelated absolute path",
			path:     "/etc/ssh/ssh_host_rsa_key",
			homeDir:  "/home/user",
			goos:     "linux",
			expected: "/etc/ssh/ssh_host_rsa_key",
		},
		{
			name:     "prefix matches user name substring",
			path:     "/home/user_other/.ssh/id_rsa",
			homeDir:  "/home/user",
			goos:     "linux",
			expected: "/home/user_other/.ssh/id_rsa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toTildePathWithHome(tt.path, tt.homeDir, tt.goos)
			if got != tt.expected {
				t.Errorf("toTildePathWithHome(%q, %q, %q) = %q; want %q",
					tt.path, tt.homeDir, tt.goos, got, tt.expected)
			}
		})
	}
}

func TestExpandTildeWithHome(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		homeDir  string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			homeDir:  "/home/user",
			expected: "",
		},
		{
			name:     "exact tilde",
			path:     "~",
			homeDir:  "/home/user",
			expected: "/home/user",
		},
		{
			name:     "tilde with relative path",
			path:     "~/.ssh/id_rsa",
			homeDir:  "/home/user",
			expected: filepath.Join("/home/user", ".ssh/id_rsa"),
		},
		{
			name:     "not a tilde path",
			path:     "/etc/ssh/config",
			homeDir:  "/home/user",
			expected: "/etc/ssh/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandTildeWithHome(tt.path, tt.homeDir)
			if got != tt.expected {
				t.Errorf("expandTildeWithHome(%q, %q) = %q; want %q",
					tt.path, tt.homeDir, got, tt.expected)
			}
		})
	}
}
