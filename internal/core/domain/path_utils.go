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
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ToTildePath converts an absolute path starting with the user's home directory
// into a relative path starting with '~'.
// For example:
//
//	/home/user/.ssh/id_rsa -> ~/.ssh/id_rsa
//	/Users/user/.ssh/id_rsa -> ~/.ssh/id_rsa
//	C:\Users\user\.ssh\id_rsa -> ~/.ssh/id_rsa
//
// If the path is already using '~' or does not start with the home directory,
// it is returned unchanged (with slashes normalized for tilde paths).
func ToTildePath(path string) string {
	homeDir, _ := os.UserHomeDir()
	return toTildePathWithHome(path, homeDir, runtime.GOOS)
}

func toTildePathWithHome(path, homeDir, goos string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~") {
		return strings.ReplaceAll(path, `\`, "/")
	}
	if homeDir == "" {
		return path
	}

	normHome := strings.TrimRight(strings.ReplaceAll(homeDir, `\`, "/"), "/")
	normPath := strings.TrimRight(strings.ReplaceAll(path, `\`, "/"), "/")

	if goos == "windows" {
		if strings.EqualFold(normPath, normHome) {
			return "~"
		}
		prefix := normHome + "/"
		if len(normPath) > len(prefix) && strings.EqualFold(normPath[:len(prefix)], prefix) {
			return "~/" + normPath[len(prefix):]
		}
	} else {
		if normPath == normHome {
			return "~"
		}
		prefix := normHome + "/"
		if strings.HasPrefix(normPath, prefix) {
			return "~/" + normPath[len(prefix):]
		}
	}

	return path
}

// ExpandTilde replaces a leading '~' in path with the user's home directory.
func ExpandTilde(path string) string {
	homeDir, _ := os.UserHomeDir()
	return expandTildeWithHome(path, homeDir)
}

func expandTildeWithHome(path, homeDir string) string {
	if path == "" || homeDir == "" {
		return path
	}
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
