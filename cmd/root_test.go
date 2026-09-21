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
	"testing"
)

func TestRootCmd_ExitOnDisconnectFlags(t *testing.T) {
	tests := []struct {
		name                  string
		args                  []string
		expectedExitOnDisc    bool
		expectedSSHConfigFile string
	}{
		{
			name:                  "default is false and empty config",
			args:                  []string{},
			expectedExitOnDisc:    false,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag --exit-on-disconnect",
			args:                  []string{"--exit-on-disconnect"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag shorthand -x",
			args:                  []string{"-x"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag --auto-exit",
			args:                  []string{"--auto-exit"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "",
		},
		{
			name:                  "flag -x combined with --sshconfig",
			args:                  []string{"--sshconfig", "/custom/ssh/config", "-x"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "/custom/ssh/config",
		},
		{
			name:                  "flag --exit-on-disconnect combined with --sshconfig",
			args:                  []string{"--exit-on-disconnect", "--sshconfig", "/custom/ssh/config"},
			expectedExitOnDisc:    true,
			expectedSSHConfigFile: "/custom/ssh/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables before each test case
			exitOnDisconnect = false
			sshConfigFile = ""

			cmd := newRootCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("unexpected error parsing flags %v: %v", tt.args, err)
			}

			if exitOnDisconnect != tt.expectedExitOnDisc {
				t.Errorf("exitOnDisconnect = %v, want %v", exitOnDisconnect, tt.expectedExitOnDisc)
			}

			if sshConfigFile != tt.expectedSSHConfigFile {
				t.Errorf("sshConfigFile = %q, want %q", sshConfigFile, tt.expectedSSHConfigFile)
			}
		})
	}
}

func TestRootCmd_FlagsExist(t *testing.T) {
	cmd := newRootCmd()

	exitFlag := cmd.PersistentFlags().Lookup("exit-on-disconnect")
	if exitFlag == nil {
		t.Fatal("expected persistent flag --exit-on-disconnect to exist")
	}
	if exitFlag.Shorthand != "x" {
		t.Errorf("expected shorthand 'x', got %q", exitFlag.Shorthand)
	}

	roFlag := cmd.PersistentFlags().Lookup("ssh-config-readonly")
	if roFlag == nil {
		t.Fatal("expected persistent flag --ssh-config-readonly to exist")
	}

	roAlias := cmd.PersistentFlags().Lookup("readonly")
	if roAlias == nil {
		t.Fatal("expected persistent flag --readonly to exist")
	}
	if roAlias.Shorthand != "r" {
		t.Errorf("expected shorthand 'r', got %q", roAlias.Shorthand)
	}
}

func TestRootCmd_ReadOnlyFlags(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedReadOnly bool
	}{
		{
			name:             "default without readonly",
			args:             []string{},
			expectedReadOnly: false,
		},
		{
			name:             "flag --ssh-config-readonly",
			args:             []string{"--ssh-config-readonly"},
			expectedReadOnly: true,
		},
		{
			name:             "flag --readonly",
			args:             []string{"--readonly"},
			expectedReadOnly: true,
		},
		{
			name:             "flag -r",
			args:             []string{"-r"},
			expectedReadOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sshConfigReadonly = false
			cmd := newRootCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("unexpected error parsing flags %v: %v", tt.args, err)
			}
			if sshConfigReadonly != tt.expectedReadOnly {
				t.Errorf("sshConfigReadonly = %v, want %v", sshConfigReadonly, tt.expectedReadOnly)
			}
		})
	}
}

func TestRootCmd_FilterAndConnectFlags(t *testing.T) {
	cmd := newRootCmd()

	filterFlag := cmd.PersistentFlags().Lookup("filter")
	if filterFlag == nil {
		t.Fatal("expected persistent flag --filter to exist")
	}
	if filterFlag.Shorthand != "f" {
		t.Errorf("expected shorthand 'f', got %q", filterFlag.Shorthand)
	}

	connectFlag := cmd.PersistentFlags().Lookup("connect")
	if connectFlag == nil {
		t.Fatal("expected persistent flag --connect to exist")
	}
	if connectFlag.Shorthand != "c" {
		t.Errorf("expected shorthand 'c', got %q", connectFlag.Shorthand)
	}
}

func TestRootCmd_FilterParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedFilter  string
		expectedConnect bool
	}{
		{
			name:            "default no filter",
			args:            []string{},
			expectedFilter:  "",
			expectedConnect: false,
		},
		{
			name:            "flag --filter",
			args:            []string{"--filter", "prod-web"},
			expectedFilter:  "prod-web",
			expectedConnect: false,
		},
		{
			name:            "flag -f",
			args:            []string{"-f", "staging"},
			expectedFilter:  "staging",
			expectedConnect: false,
		},
		{
			name:            "flag --connect",
			args:            []string{"--connect"},
			expectedFilter:  "",
			expectedConnect: true,
		},
		{
			name:            "flag -c with -f",
			args:            []string{"-c", "-f", "my-server"},
			expectedFilter:  "my-server",
			expectedConnect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterQuery = ""
			connectDirectly = false
			cmd := newRootCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("unexpected error parsing flags %v: %v", tt.args, err)
			}
			if filterQuery != tt.expectedFilter {
				t.Errorf("filterQuery = %q, want %q", filterQuery, tt.expectedFilter)
			}
			if connectDirectly != tt.expectedConnect {
				t.Errorf("connectDirectly = %v, want %v", connectDirectly, tt.expectedConnect)
			}
		})
	}
}
