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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kevinburke/ssh_config"
)

var reMultipleNewlines = regexp.MustCompile(`\n{3,}`)

// configCacheTTL bounds how long a parsed config may be reused for read paths.
// Kept short so external file edits are picked up quickly; explicit refreshes
// and local mutations invalidate the cache immediately regardless.
const configCacheTTL = 2 * time.Second

// loadConfig reads and parses the SSH config file plus every file pulled in
// via top-level `Include` directives. Returns a loadedConfig containing all
// per-file parses in OpenSSH precedence order (main first).
func (r *Repository) loadConfig() (*loadedConfig, error) {
	lc, err := r.resolveIncludes(r.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return lc, nil
}

// saveFiles writes only the entries of lc whose paths appear in dirty back to
// disk. Each file gets its own atomic temp+rename and its own rolling backup.
func (r *Repository) saveFiles(lc *loadedConfig, dirty []string) error {
	dirtySet := make(map[string]bool, len(dirty))
	for _, p := range dirty {
		dirtySet[p] = true
	}

	for _, f := range lc.files {
		if !dirtySet[f.path] {
			continue
		}
		if err := r.writeOneFile(f.path, f.cfg); err != nil {
			return err
		}
	}
	// Any successful write changes the parsed config on disk; drop the cache so
	// the next read reflects it immediately.
	r.InvalidateCache()
	return nil
}

// loadConfigCached is the read-path counterpart to loadConfig: it reuses a very
// recent parse so the UI's frequent listings (one per ping result) don't re-read
// and re-parse every included file each time. Mutations bypass the cache via
// loadConfig and invalidate it in saveFiles, and InvalidateCache forces a
// re-read (e.g. on explicit refresh).
func (r *Repository) loadConfigCached() (*loadedConfig, error) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	if r.cachedLC != nil && time.Since(r.cacheAt) < configCacheTTL {
		return r.cachedLC, nil
	}

	lc, err := r.loadConfig()
	if err != nil {
		return nil, err
	}
	r.cachedLC = lc
	r.cacheAt = time.Now()
	return lc, nil
}

// InvalidateCache drops the cached parse so the next read re-reads from disk.
func (r *Repository) InvalidateCache() {
	r.cacheMu.Lock()
	r.cachedLC = nil
	r.cacheMu.Unlock()
}

func (r *Repository) writeOneFile(path string, cfg *ssh_config.Config) error {
	configDir := filepath.Dir(path)

	tempFile, err := r.createTempFile(configDir)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for %s: %w", path, err)
	}

	defer func() {
		if removeErr := r.fileSystem.Remove(tempFile); removeErr != nil {
			r.logger.Warnf("failed to remove temporary file %s: %v", tempFile, removeErr)
		}
	}()

	if err := r.writeConfigToFile(tempFile, cfg); err != nil {
		return fmt.Errorf("failed to write config to temporary file: %w", err)
	}

	if err := r.createOriginalBackupForIfNeeded(path); err != nil {
		return fmt.Errorf("failed to create original backup for %s: %w", path, err)
	}

	if err := r.createBackupFor(path); err != nil {
		return fmt.Errorf("failed to create backup for %s: %w", path, err)
	}

	// Resolve symlinks before atomic rename so we don't replace a symlink with a regular file.
	target := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		target = resolved
	}

	if err := r.fileSystem.Rename(tempFile, target); err != nil {
		return fmt.Errorf("failed to atomically replace %s: %w", target, err)
	}

	r.logger.Infof("SSH config successfully updated: %s", target)
	return nil
}

// writeConfigToFile writes the SSH config content to the specified file.
func (r *Repository) writeConfigToFile(filePath string, cfg *ssh_config.Config) error {
	file, err := r.fileSystem.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, SSHConfigPerms)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close file %s: %v", filePath, cerr)
		}
	}()

	configContent := cfg.String()
	// Collapse 3 or more consecutive newlines to 2 (one blank line between blocks)
	configContent = reMultipleNewlines.ReplaceAllString(configContent, "\n\n")
	// Turn the inert Include markers back into real directives.
	configContent = restoreIncludeDirectives(configContent)

	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file to disk: %w", err)
	}

	return nil
}

// createTempFile creates a temporary file in the specified directory.
func (r *Repository) createTempFile(dir string) (string, error) {
	timestamp := time.Now().Format("20060102150405.000000")
	tempFileName := fmt.Sprintf("config%s%s", timestamp, TempSuffix)
	tempFilePath := filepath.Join(dir, tempFileName)

	f, err := r.fileSystem.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, SSHConfigPerms)
	if err != nil {
		return "", err
	}
	if cerr := f.Close(); cerr != nil {
		r.logger.Warnf("failed to close temporary file %s: %v", tempFilePath, cerr)
	}

	return tempFilePath, nil
}
