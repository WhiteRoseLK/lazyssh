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

	"github.com/WhiteRoseLK/neossh/internal/adapters/data/ssh_config_file"
	"github.com/WhiteRoseLK/neossh/internal/logger"

	"github.com/WhiteRoseLK/neossh/internal/adapters/ui"
	"github.com/WhiteRoseLK/neossh/internal/core/services"
	"github.com/spf13/cobra"
)

var (
	version       = "develop"
	gitCommit     = "unknown"
	sshConfigFile string

	rootCmd = &cobra.Command{
		Use:   ui.AppName,
		Short: "NeoSSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if sshConfigFile == "" {
				sshConfigFile = filepath.Join(home, ".ssh", "config")
			} else {
				stat, err := os.Stat(sshConfigFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
					os.Exit(1)
				}
				if stat.Mode()&os.ModeType != 0 {
					f, err := os.CreateTemp("", "tmpfile-")
					if err != nil {
						log.Fatal(err)
					}

					// close and remove the temporary file at the end of the program
					defer func() {
						_ = f.Close()
						_ = os.Remove(f.Name())
					}()

					// write data to the temporary file
					fd, err := os.Open(sshConfigFile) //nolint:gosec // G304: path comes from user flag, intentional
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
						os.Exit(1)
					}
					defer func() {
						_ = fd.Close()
					}()

					// Read the entire contents at once
					content, err := io.ReadAll(fd)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
						os.Exit(1)
					}
					if _, err := f.WriteString(string(content)); err != nil {
						log.Fatal(err)
					}

					sshConfigFile = f.Name()
				}
			}

			configDir := filepath.Join(home, ".neossh")
			if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
				configDir = filepath.Join(xdgConfig, "neossh")
			}
			if err := os.MkdirAll(configDir, 0o750); err != nil {
				log.Errorw("failed to create config directory", "error", err)
				os.Exit(1)
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

			serverRepo := ssh_config_file.NewRepository(log, sshConfigFile, metaDataFile)
			serverService := services.NewServerService(log, serverRepo)
			tui := ui.NewTUI(log, serverService, version, gitCommit)

			return tui.Run()
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&sshConfigFile, "sshconfig", "", "path to ssh config file (default: ~/.ssh/config)",
	)

	rootCmd.SilenceUsage = true
}
