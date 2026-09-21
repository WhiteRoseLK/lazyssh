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

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/WhiteRoseLK/neossh/internal/adapters/data/ssh_config_file"
	"github.com/WhiteRoseLK/neossh/internal/adapters/ui"
	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/WhiteRoseLK/neossh/internal/core/ports"
	"github.com/WhiteRoseLK/neossh/internal/core/services"
	"github.com/WhiteRoseLK/neossh/internal/logger"
	"github.com/spf13/cobra"
)

var (
	version   = "develop"
	gitCommit = "unknown"

	sshConfigFile     string
	exitOnDisconnect  bool
	sshConfigReadonly bool
	filterQuery       string
	connectDirectly   bool
	importKnownHosts  bool
	knownHostsFile    string
	showHidden        bool
	themeFlag         string

	rootCmd = newRootCmd()
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   ui.AppName + " [filter]",
		Short: "NeoSSH server picker TUI",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			isReadonly := sshConfigReadonly
			if ro, err := cmd.Flags().GetBool("readonly"); err == nil && ro {
				isReadonly = true
			}
			if ro, err := cmd.Flags().GetBool("ssh-config-readonly"); err == nil && ro {
				isReadonly = true
			}

			filter := filterQuery
			if len(args) > 0 {
				filter = args[0]
			}
			if f, err := cmd.Flags().GetString("filter"); err == nil && f != "" {
				filter = f
			}

			isConnect := connectDirectly
			if c, err := cmd.Flags().GetBool("connect"); err == nil && c {
				isConnect = true
			}

			isImportKH := importKnownHosts
			if ikh, err := cmd.Flags().GetBool("import-known-hosts"); err == nil && ikh {
				isImportKH = true
			}
			if kh, err := cmd.Flags().GetString("known-hosts"); err == nil && kh != "" {
				knownHostsFile = kh
			}

			log, err := logger.New("NEOSSH")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			//nolint:errcheck // log.Sync may return an error which is safe to ignore here
			defer log.Sync()

			home, err := os.UserHomeDir()
			if err != nil {
				log.Errorw("failed to get user home directory", "error", err)
				os.Exit(1)
			}

			resolvedConfig, cleanup, err := resolveSSHConfigFile(home, sshConfigFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error resolving config file: %v\n", err)
				os.Exit(1)
			}
			defer cleanup()

			metaDataFile, err := ensureMetadataFile(home)
			if err != nil {
				log.Errorw("failed to setup metadata file", "error", err)
				os.Exit(1)
			}

			serverRepo := ssh_config_file.NewRepository(log, resolvedConfig, metaDataFile)
			serverService := services.NewServerService(log, serverRepo, services.WithReadOnly(isReadonly))

			if isImportKH {
				if isReadonly {
					return fmt.Errorf("cannot import known_hosts in read-only mode")
				}
				khPath := knownHostsFile
				if khPath == "" {
					khPath = filepath.Join(home, ".ssh", "known_hosts")
				}
				result, err := serverService.ImportKnownHosts(khPath)
				if err != nil {
					return fmt.Errorf("failed to import known_hosts: %w", err)
				}
				if result.Imported == 0 {
					fmt.Printf("No new hosts to import from %s (%d host(s) already configured or skipped).\n",
						khPath, result.Skipped)
				} else {
					fmt.Printf("Successfully imported %d host(s) from %s (%d already configured or skipped).\n",
						result.Imported, khPath, result.Skipped)
				}
				return nil

			}

			if isConnect {
				connected, err := handleDirectConnect(filter, serverService)
				if err != nil {
					return err
				}
				if connected {
					return nil
				}
				// If multiple matches without exact match, launch interactive TUI pre-filtered so user can choose
			}

			theme := themeFlag
			if t, err := cmd.Flags().GetString("theme"); err == nil && t != "" {
				theme = t
			}

			tui := ui.NewTUI(log, serverService, version, gitCommit, ui.Config{
				ExitOnDisconnect: exitOnDisconnect,
				ReadOnly:         isReadonly,
				InitialFilter:    filter,
				ShowHidden:       showHidden,
				Theme:            theme,
			})

			return tui.Run()
		},
	}

	cmd.PersistentFlags().StringVar(
		&sshConfigFile, "sshconfig", "", "path to ssh config file (default: ~/.ssh/config)",
	)
	cmd.PersistentFlags().BoolVarP(
		&exitOnDisconnect, "exit-on-disconnect", "x", false, "exit neossh after SSH session finishes",
	)
	cmd.PersistentFlags().BoolVar(
		&exitOnDisconnect, "auto-exit", false, "exit neossh after SSH session finishes",
	)
	cmd.PersistentFlags().BoolVar(
		&sshConfigReadonly, "ssh-config-readonly", false, "run in read-only mode (prevent modifying ~/.ssh/config)",
	)
	cmd.PersistentFlags().BoolVarP(
		&sshConfigReadonly, "readonly", "r", false, "run in read-only mode (alias for --ssh-config-readonly)",
	)
	cmd.PersistentFlags().StringVarP(
		&filterQuery, "filter", "f", "", "pre-filter server list by alias, hostname, or tag",
	)
	cmd.PersistentFlags().BoolVarP(
		&connectDirectly, "connect", "c", false, "connect directly to matching server without launching full TUI picker",
	)
	cmd.PersistentFlags().BoolVar(
		&importKnownHosts, "import-known-hosts", false, "import hosts from ~/.ssh/known_hosts into SSH config",
	)
	cmd.PersistentFlags().StringVar(
		&knownHostsFile, "known-hosts", "", "path to known_hosts file (default: ~/.ssh/known_hosts)",
	)
	cmd.PersistentFlags().BoolVarP(
		&showHidden, "show-hidden", "H", false, "display hidden servers in UI list",
	)
	cmd.PersistentFlags().StringVarP(
		&themeFlag, "theme", "t", "", "set color theme: dark, light, or system",
	)

	cmd.SilenceUsage = true
	return cmd
}

func resolveSSHConfigFile(home, customPath string) (string, func(), error) {
	if customPath == "" {
		return filepath.Join(home, ".ssh", "config"), func() {}, nil
	}

	stat, err := os.Stat(customPath)
	if err != nil {
		return "", nil, err
	}

	if stat.Mode()&os.ModeType != 0 {
		f, err := os.CreateTemp("", "tmpfile-")
		if err != nil {
			return "", nil, err
		}

		cleanup := func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}

		fd, err := os.Open(customPath) //nolint:gosec // G304: path comes from user flag, intentional
		if err != nil {
			cleanup()
			return "", nil, err
		}
		defer func() {
			_ = fd.Close()
		}()

		content, err := io.ReadAll(fd)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		if _, err := f.WriteString(string(content)); err != nil {
			cleanup()
			return "", nil, err
		}

		return f.Name(), cleanup, nil
	}

	return customPath, func() {}, nil
}

func ensureMetadataFile(home string) (string, error) {
	configDir := filepath.Join(home, ".neossh")
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = filepath.Join(xdgConfig, "neossh")
	}
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return "", err
	}
	metaDataFile := filepath.Join(configDir, "metadata.json")

	// Migrate metadata from legacy lazyssh if neossh metadata doesn't exist yet
	if _, err := os.Stat(metaDataFile); os.IsNotExist(err) {
		legacyFile := filepath.Join(home, ".lazyssh", "metadata.json")
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			legacyFile = filepath.Join(xdgConfig, "lazyssh", "metadata.json")
		}
		//nolint:gosec // G304: path constructed from user home directory
		if data, err := os.ReadFile(legacyFile); err == nil {
			_ = os.WriteFile(metaDataFile, data, 0o600)
		}
	}

	return metaDataFile, nil
}

func handleDirectConnect(filter string, serverService ports.ServerService) (bool, error) {
	if strings.TrimSpace(filter) == "" {
		return false, fmt.Errorf("--connect requires a server alias or filter argument (e.g. neossh -c <alias>)")
	}

	matches, err := serverService.ListServers(filter)
	if err != nil {
		return false, fmt.Errorf("failed to query servers: %w", err)
	}

	var targetServer *domain.Server
	for i := range matches {
		if strings.EqualFold(matches[i].Alias, filter) || slices.ContainsFunc(matches[i].Aliases, func(a string) bool {
			return strings.EqualFold(a, filter)
		}) {
			targetServer = &matches[i]
			break
		}
	}
	if targetServer == nil && len(matches) == 1 {
		targetServer = &matches[0]
	}

	if targetServer != nil {
		if targetServer.IsWildcardServer() {
			return false, fmt.Errorf("cannot initiate direct SSH connection to wildcard pattern block '%s'", targetServer.Alias)
		}
		return true, serverService.SSH(targetServer.Alias)
	}

	if len(matches) == 0 {
		return false, fmt.Errorf("no server matching '%s' found", filter)
	}

	return false, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
