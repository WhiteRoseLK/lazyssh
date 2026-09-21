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

package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/WhiteRoseLK/neossh/internal/core/ports"
	"github.com/WhiteRoseLK/neossh/internal/core/services"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// GitSSHSetup is a dialog for configuring Git SSH keys and profiles.
type GitSSHSetup struct {
	*tview.Flex
	app        *tview.Application
	gitService ports.GitService
	serverRepo ports.ServerRepository
	form       *tview.Form
	infoText   *tview.TextView
	onDone     func()
	onCancel   func()

	selectedKey   string
	selectedScope string
	keys          []domain.SSHKey
	repoPath      string
	isRepo        bool
}

// NewGitSSHSetup creates a new GitSSHSetup dialog.
func NewGitSSHSetup(app *tview.Application, gitService ports.GitService, serverRepo ports.ServerRepository) *GitSSHSetup {
	setup := &GitSSHSetup{
		Flex:       tview.NewFlex().SetDirection(tview.FlexRow),
		app:        app,
		gitService: gitService,
		serverRepo: serverRepo,
		form:       tview.NewForm(),
		infoText:   tview.NewTextView(),
	}

	setup.infoText.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 2, 2)

	setup.form.SetBorderPadding(1, 1, 2, 2)

	setup.AddItem(setup.infoText, 0, 1, false).
		AddItem(setup.form, 0, 2, true)

	setup.SetBorder(true).
		SetTitle(" Configure Git SSH Key & Profiles ").
		SetTitleAlign(tview.AlignLeft)

	return setup
}

// OnDone sets the handler for successful completion.
func (g *GitSSHSetup) OnDone(handler func()) *GitSSHSetup {
	g.onDone = handler
	return g
}

// OnCancel sets the handler for cancellation.
func (g *GitSSHSetup) OnCancel(handler func()) *GitSSHSetup {
	g.onCancel = handler
	return g
}

// Show displays the Git SSH key setup dialog.
func (g *GitSSHSetup) Show() error {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	g.isRepo = g.gitService.IsGitRepository(cwd)
	if g.isRepo {
		if root, err := g.gitService.GetGitRootPath(cwd); err == nil {
			g.repoPath = root
		} else {
			g.repoPath = cwd
		}
	} else {
		g.repoPath = ""
	}

	currentConfig, _ := g.gitService.GetCurrentGitSSHConfig(g.repoPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	sshDir := filepath.Join(homeDir, ".ssh")

	keys, err := g.gitService.ListSSHKeys(sshDir, g.serverRepo)
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}
	g.keys = keys

	if len(keys) == 0 {
		return fmt.Errorf("no SSH private keys found in %s or SSH config", sshDir)
	}

	g.buildInfoText(currentConfig)
	g.buildForm(currentConfig)

	g.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if g.onCancel != nil {
				g.onCancel()
			}
			return nil
		}
		return event
	})

	g.app.SetRoot(g, true)
	g.app.SetFocus(g.form)

	return nil
}

func (g *GitSSHSetup) buildInfoText(currentConfig string) {
	var info string
	if g.isRepo {
		info = fmt.Sprintf("[yellow]Git Repository:[-] %s\n", g.repoPath)
		if remoteName, remoteURL, err := g.gitService.GetPushRemoteURL(g.repoPath); err == nil && remoteURL != "" {
			info += fmt.Sprintf("[blue]Push Remote (%s):[-] %s\n", remoteName, remoteURL)
		}
	} else {
		info = "[yellow]Global Git Configuration[-] (not inside a Git repository)\n"
	}

	if currentConfig != "" {
		info += fmt.Sprintf("\n[green]Current SSH Command:[-]\n%s\n", currentConfig)
	} else {
		info += "\n[dim]No custom Git SSH key configured (using default OpenSSH identity)[-]\n"
	}

	keysInAgent := 0
	for _, key := range g.keys {
		if key.LoadedInAgent {
			keysInAgent++
		}
	}
	if keysInAgent > 0 {
		info += fmt.Sprintf("[green]%d key(s) loaded in ssh-agent[-]\n", keysInAgent)
	}

	g.infoText.SetText(info)
}

func (g *GitSSHSetup) buildForm(currentConfig string) {
	g.form.Clear(true)

	keyOptions := make([]string, len(g.keys))
	for i, key := range g.keys {
		status := ""
		switch {
		case key.LoadedInAgent:
			status = " [green](in agent)[-]"
		case !key.HasPublicKey:
			status = " [red](no .pub)[-]"
		case key.IsEncrypted:
			status = " [yellow](encrypted)[-]"
		}
		commentPart := ""
		if key.Comment != "" {
			commentPart = fmt.Sprintf(" - %s", key.Comment)
		}
		keyOptions[i] = fmt.Sprintf("%s%s%s", key.Name, commentPart, status)
	}

	g.form.AddDropDown("SSH Key", keyOptions, 0, func(_ string, optionIndex int) {
		if optionIndex >= 0 && optionIndex < len(g.keys) {
			g.selectedKey = g.keys[optionIndex].Path
		}
	})

	if len(g.keys) > 0 {
		g.selectedKey = g.keys[0].Path
	}

	var scopeOptions []string
	if g.isRepo {
		scopeOptions = []string{"local (this repository only)", "global (all repositories)"}
		g.selectedScope = services.ScopeLocal
	} else {
		scopeOptions = []string{"global (all repositories)"}
		g.selectedScope = services.ScopeGlobal
	}

	g.form.AddDropDown("Scope", scopeOptions, 0, func(_ string, optionIndex int) {
		if g.isRepo && optionIndex == 0 {
			g.selectedScope = services.ScopeLocal
		} else {
			g.selectedScope = services.ScopeGlobal
		}
	})

	g.form.AddButton("Configure", g.handleConfigure)

	if currentConfig != "" {
		g.form.AddButton("Clear Config", g.handleClearConfig)
	}

	g.form.AddButton("Cancel", func() {
		if g.onCancel != nil {
			g.onCancel()
		}
	})
}

func (g *GitSSHSetup) handleConfigure() {
	if err := g.gitService.ConfigureGitSSHKey(g.repoPath, g.selectedKey, g.selectedScope); err != nil {
		errorModal := tview.NewModal().
			SetText(fmt.Sprintf("Configuration failed:\n\n%v", err)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(_ int, _ string) {
				g.app.SetRoot(g, true)
				g.app.SetFocus(g.form)
			})
		g.app.SetRoot(errorModal, true)
		return
	}

	successModal := tview.NewModal().
		SetText(fmt.Sprintf("Git SSH key configured successfully!\n\nSSH Key: %s\nScope: %s",
			filepath.Base(g.selectedKey), g.selectedScope)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			if g.onDone != nil {
				g.onDone()
			}
		})
	g.app.SetRoot(successModal, true)
}

func (g *GitSSHSetup) handleClearConfig() {
	buttons := []string{"Cancel", "Clear Local", "Clear Both"}
	if !g.isRepo {
		buttons = []string{"Cancel", "Clear Global"}
	}

	confirmModal := tview.NewModal().
		SetText("Clear Git SSH configuration?\n\nThis will reset Git to use default SSH behavior.").
		AddButtons(buttons).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 0 {
				g.app.SetRoot(g, true)
				g.app.SetFocus(g.form)
				return
			}

			scope := services.ScopeLocal
			if !g.isRepo {
				scope = services.ScopeGlobal
			} else if buttonIndex == 2 {
				scope = services.ScopeBoth
			}

			if err := g.gitService.ClearGitSSHConfig(g.repoPath, scope); err != nil {
				errorModal := tview.NewModal().
					SetText(fmt.Sprintf("Failed to clear configuration:\n\n%v", err)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(_ int, _ string) {
						g.app.SetRoot(g, true)
						g.app.SetFocus(g.form)
					})
				g.app.SetRoot(errorModal, true)
				return
			}

			successModal := tview.NewModal().
				SetText("Git SSH configuration cleared successfully!").
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(_ int, _ string) {
					if g.onDone != nil {
						g.onDone()
					}
				})
			g.app.SetRoot(successModal, true)
		})

	g.app.SetRoot(confirmModal, true)
}
