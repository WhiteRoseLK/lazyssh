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
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// =============================================================================
// Event Handlers (handle user input/events)
// =============================================================================
const (
	ForwardTypeLocal   = "Local"
	ForwardTypeRemote  = "Remote"
	ForwardTypeDynamic = "Dynamic"

	ForwardModeOnlyForward = "Only forward"
	ForwardModeForwardSSH  = "Forward + SSH"
)

func (t *tui) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	// Don't handle global keys when search has focus
	if t.app.GetFocus() == t.searchBar {
		return event
	}

	switch event.Rune() {
	case '0':
		t.handleSearchFocus()
		return nil
	case '1':
		t.handleServerListFocus()
		return nil
	case '2':
		t.handleDetailsFocus()
		return nil
	case 'q':
		t.handleQuit()
		return nil
	case '/':
		t.handleSearchFocus()
		return nil
	case 'a':
		t.handleServerAdd()
		return nil
	case 'e':
		t.handleServerEdit()
		return nil
	case 'd':
		t.handleServerDelete()
		return nil
	case 'p':
		t.handleServerPin()
		return nil
	case 's':
		t.handleSortToggle()
		return nil
	case 'S':
		t.handleSortReverse()
		return nil
	case 'c':
		t.handleCopyCommand()
		return nil
	case 'v':
		t.handlePasteCommand()
		return nil
	case 'y', 'C':
		t.handleServerClone()
		return nil
	case 'h':
		t.handleCopyHost()
		return nil
	case 'g':
		t.handlePingSelected()
		return nil
	case 'G':
		t.handlePingAll()
		return nil
	case 'r':
		t.handleRefreshBackground()
		return nil
	case 't':
		t.handleTagsEdit()
		return nil
	case 'f':
		t.handlePortForward()
		return nil
	case 'x':
		t.handleStopForwarding()
		return nil
	case 'j':
		t.handleNavigateDown()
		return nil
	case 'k':
		t.handleNavigateUp()
		return nil
	case 'K':
		t.handleInstallSSHKey()
		return nil
	}

	if event.Key() == tcell.KeyEnter {
		t.handleServerConnect()
		return nil
	}

	return event
}

func (t *tui) handleQuit() {
	t.app.Stop()
}

func (t *tui) handleServerPin() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		pinned := server.PinnedAt.IsZero()
		_ = t.serverService.SetPinned(server.Alias, pinned)
		t.refreshServerList()
	}
}

func (t *tui) handleSortToggle() {
	t.sortMode = t.sortMode.ToggleField()
	t.showStatusTemp("Sort: " + t.sortMode.String())
	t.updateListTitle()
	t.persistSortMode()
	t.refreshServerList()
}

func (t *tui) handleSortReverse() {
	t.sortMode = t.sortMode.Reverse()
	t.showStatusTemp("Sort: " + t.sortMode.String())
	t.updateListTitle()
	t.persistSortMode()
	t.refreshServerList()
}

func (t *tui) handleCopyCommand() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		cmd := BuildSSHCommand(server)
		if err := clipboard.WriteAll(cmd); err == nil {
			t.showStatusTemp("Copied: " + cmd)
		} else {
			t.showStatusTemp("Failed to copy to clipboard")
		}
	}
}

func (t *tui) handleCopyHost() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		host := server.Host
		if err := clipboard.WriteAll(host); err == nil {
			t.showStatusTemp("Copied: " + host)
		} else {
			t.showStatusTemp("Failed to copy to clipboard")
		}
	}
}

func (t *tui) handlePasteCommand() {
	// Read from clipboard
	clipContent, err := clipboard.ReadAll()
	if err != nil {
		t.showStatusTemp("Failed to read from clipboard")
		return
	}

	// Try to parse as SSH command
	server, err := ParseSSHCommand(clipContent)
	if err != nil {
		t.showStatusTemp("Invalid SSH command in clipboard: " + err.Error())
		return
	}

	// Check for duplicate alias and auto-adjust if necessary
	existingAliases := t.getExistingAliases()
	server.Alias = GenerateUniqueAlias(server.Alias, existingAliases)

	// Show the server form with parsed data
	// Note: For Add mode, original should be nil. We'll set initial data separately.
	form := NewServerForm(ServerFormAdd, nil).
		SetInitialData(server).
		SetApp(t.app).
		SetVersionInfo(t.version, t.commit).
		OnSave(t.handleServerSave).
		OnCancel(t.handleFormCancel).
		SetExistingAliases(existingAliases)
	t.app.SetRoot(form, true)
}

// getExistingAliases returns all existing server aliases
func (t *tui) getExistingAliases() []string {
	servers, err := t.serverService.ListServers("")
	if err != nil {
		return []string{}
	}

	aliases := make([]string, 0)
	for _, s := range servers {
		if len(s.Aliases) > 0 {
			aliases = append(aliases, s.Aliases...)
		} else if s.Alias != "" {
			aliases = append(aliases, s.Alias)
		}
	}
	return aliases
}

// getExistingAliasesExcept returns all existing server aliases except those belonging to exclude
func (t *tui) getExistingAliasesExcept(exclude domain.Server) []string {
	servers, err := t.serverService.ListServers("")
	if err != nil {
		return []string{}
	}

	aliases := make([]string, 0)
	for _, s := range servers {
		if s.Alias == exclude.Alias || slices.Contains(s.Aliases, exclude.Alias) {
			continue
		}
		if len(s.Aliases) > 0 {
			aliases = append(aliases, s.Aliases...)
		} else if s.Alias != "" {
			aliases = append(aliases, s.Alias)
		}
	}
	return aliases
}

func (t *tui) handleTagsEdit() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.showEditTagsForm(server)
	}
}

func (t *tui) handleNavigateDown() {
	if t.isServerListFocused() {
		currentIdx := t.serverList.GetCurrentItem()
		itemCount := t.serverList.GetItemCount()
		if currentIdx < itemCount-1 {
			t.serverList.SetCurrentItem(currentIdx + 1)
		} else {
			t.serverList.SetCurrentItem(0)
		}
	}
}

func (t *tui) handleNavigateUp() {
	if t.isServerListFocused() {
		currentIdx := t.serverList.GetCurrentItem()
		if currentIdx > 0 {
			t.serverList.SetCurrentItem(currentIdx - 1)
		} else {
			t.serverList.SetCurrentItem(t.serverList.GetItemCount() - 1)
		}
	}
}

func (t *tui) handleSearchInput(query string) {
	filtered, _ := t.serverService.ListServers(query)
	if strings.TrimSpace(query) == "" {
		sortServersForUI(filtered, t.sortMode)
	}
	t.serverList.UpdateServers(filtered)
	if len(filtered) == 0 {
		t.details.ShowEmpty()
	}
}

func (t *tui) isServerListFocused() bool {
	if t == nil || t.app == nil || t.serverList == nil {
		return false
	}
	focus := t.app.GetFocus()
	return focus == t.serverList || focus == t.serverList.List
}

func (t *tui) handleSearchFocus() {
	if t.app != nil && t.searchBar != nil {
		t.app.SetFocus(t.searchBar)
	}
}

func (t *tui) handleServerListFocus() {
	if t.app != nil && t.serverList != nil {
		t.app.SetFocus(t.serverList)
	}
}

func (t *tui) handleDetailsFocus() {
	if t.app != nil && t.details != nil {
		t.app.SetFocus(t.details)
	}
}

func (t *tui) handleSearchNavigate(direction int) {
	if t.serverList != nil {
		t.app.SetFocus(t.serverList)

		currentIdx := t.serverList.GetCurrentItem()
		itemCount := t.serverList.GetItemCount()

		if itemCount == 0 {
			return
		}

		if direction > 0 {
			if currentIdx < itemCount-1 {
				t.serverList.SetCurrentItem(currentIdx + 1)
			} else {
				t.serverList.SetCurrentItem(0)
			}
		} else {
			if currentIdx > 0 {
				t.serverList.SetCurrentItem(currentIdx - 1)
			} else {
				t.serverList.SetCurrentItem(itemCount - 1)
			}
		}

		if server, ok := t.serverList.GetSelectedServer(); ok {
			t.details.UpdateServer(server)
		}
	}
}

func (t *tui) handleReturnToSearch() {
	if t.searchBar != nil {
		t.app.SetFocus(t.searchBar)
	}
}

func (t *tui) handleServerConnect() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		var sshErr error
		t.app.Suspend(func() {
			if err := t.serverService.SSH(server.Alias); err != nil {
				sshErr = err
				t.logger.Errorw("ssh session error", "alias", server.Alias, "error", err)
			}
		})
		t.app.Sync()
		t.refreshServerList()
		if sshErr != nil {
			t.showSSHErrorModal(server.Alias, sshErr.Error())
		}
	}
}

func (t *tui) handleInstallSSHKey() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		alias := server.Alias
		t.showStatusTemp(fmt.Sprintf("Installing key to %s…", alias))
		var copyErr error
		t.app.Suspend(func() {
			if err := t.serverService.CopySSHKey(alias); err != nil {
				copyErr = err
				t.logger.Errorw("failed to install ssh key", "alias", alias, "error", err)
			}
		})
		t.app.Sync()
		t.refreshServerList()
		if copyErr != nil {
			t.showErrorModal(fmt.Sprintf("Failed to install SSH key to %q", alias), copyErr.Error())
		}
	}
}

func (t *tui) handleServerSelectionChange(server domain.Server) {
	t.details.UpdateServer(server)
}

func (t *tui) handleServerAdd() {
	form := NewServerForm(ServerFormAdd, nil).
		SetApp(t.app).
		SetVersionInfo(t.version, t.commit).
		OnSave(t.handleServerSave).
		OnCancel(t.handleFormCancel).
		SetExistingAliases(t.getExistingAliases())
	t.app.SetRoot(form, true)
}

func (t *tui) handleServerEdit() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		form := NewServerForm(ServerFormEdit, &server).
			SetApp(t.app).
			SetVersionInfo(t.version, t.commit).
			OnSave(t.handleServerSave).
			OnCancel(t.handleFormCancel).
			SetExistingAliases(t.getExistingAliasesExcept(server))
		t.app.SetRoot(form, true)
	}
}

func (t *tui) handleServerClone() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		cloned := cloneServer(server)
		existingAliases := t.getExistingAliases()
		cloned.Alias = GenerateUniqueAlias(server.Alias, existingAliases)
		cloned.Aliases = []string{cloned.Alias}
		cloned.PinnedAt = time.Time{}
		cloned.LastSeen = time.Time{}
		cloned.PingStatus = ""
		cloned.PingLatency = 0

		form := NewServerForm(ServerFormAdd, nil).
			SetInitialData(&cloned).
			SetApp(t.app).
			SetVersionInfo(t.version, t.commit).
			OnSave(t.handleServerSave).
			OnCancel(t.handleFormCancel).
			SetExistingAliases(existingAliases)
		t.app.SetRoot(form, true)
	}
}

func cloneServer(s domain.Server) domain.Server {
	cloned := s
	cloned.Aliases = nil
	if s.IdentityFiles != nil {
		cloned.IdentityFiles = make([]string, len(s.IdentityFiles))
		copy(cloned.IdentityFiles, s.IdentityFiles)
	}
	if s.Tags != nil {
		cloned.Tags = make([]string, len(s.Tags))
		copy(cloned.Tags, s.Tags)
	}
	if s.LocalForward != nil {
		cloned.LocalForward = make([]string, len(s.LocalForward))
		copy(cloned.LocalForward, s.LocalForward)
	}
	if s.RemoteForward != nil {
		cloned.RemoteForward = make([]string, len(s.RemoteForward))
		copy(cloned.RemoteForward, s.RemoteForward)
	}
	if s.DynamicForward != nil {
		cloned.DynamicForward = make([]string, len(s.DynamicForward))
		copy(cloned.DynamicForward, s.DynamicForward)
	}
	if s.SendEnv != nil {
		cloned.SendEnv = make([]string, len(s.SendEnv))
		copy(cloned.SendEnv, s.SendEnv)
	}
	if s.SetEnv != nil {
		cloned.SetEnv = make([]string, len(s.SetEnv))
		copy(cloned.SetEnv, s.SetEnv)
	}
	return cloned
}

func (t *tui) handleServerSave(server domain.Server, original *domain.Server) {
	var err error
	if original != nil {
		// Edit mode
		err = t.serverService.UpdateServer(*original, server)
	} else {
		// Add mode
		err = t.serverService.AddServer(server)
	}
	if err != nil {
		var ambig *domain.ErrAmbiguousHost
		if errors.As(err, &ambig) && original != nil {
			t.showFileChoiceModal(ambig.Alias, ambig.Candidates, "Save", func(chosen string) {
				origCopy := *original
				origCopy.SourceFile = chosen
				newCopy := server
				newCopy.SourceFile = chosen
				if err := t.serverService.UpdateServer(origCopy, newCopy); err != nil {
					t.showStatusTempColor("Save failed: "+err.Error(), "#FF6B6B")
					return
				}
				t.showStatusTemp(fmt.Sprintf("Updated %s in %s", newCopy.Alias, chosen))
				t.refreshServerList()
				t.handleFormCancel()
			})
			return
		}
		// Stay on form; show a small modal with the error
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Save failed: %v", err)).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) { t.handleModalClose() })
		t.app.SetRoot(modal, true)
		return
	}

	if server.SourceFile != "" {
		verb := "Added"
		if original != nil {
			verb = "Updated"
		}
		t.showStatusTemp(fmt.Sprintf("%s %s in %s", verb, server.Alias, server.SourceFile))
	}
	t.refreshServerList()
	t.handleFormCancel()
}

func (t *tui) handleServerDelete() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.showDeleteConfirmModal(server)
	}
}

func (t *tui) handleFormCancel() {
	t.returnToMain()
}

func (t *tui) handlePingSelected() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		alias := server.Alias

		// Set checking status
		server.PingStatus = StatusChecking
		t.pingStatuses[alias] = server
		t.updateServerListWithPingStatus()

		t.showStatusTemp(fmt.Sprintf("Pinging %s…", alias))
		go func() {
			up, dur, err := t.serverService.Ping(server)
			t.app.QueueUpdateDraw(func() {
				if ps, ok := t.pingStatuses[alias]; ok {
					if err != nil || !up {
						ps.PingStatus = StatusDown
						ps.PingLatency = 0
						t.showStatusTempColor(fmt.Sprintf("Ping %s: DOWN", alias), "#FF6B6B")
					} else {
						ps.PingStatus = StatusUp
						ps.PingLatency = dur
						t.showStatusTempColor(fmt.Sprintf("Ping %s: UP (%s)", alias, dur), "#A0FFA0")
					}
					t.pingStatuses[alias] = ps
					t.updateServerListWithPingStatus()
				}
			})
		}()
	}
}

func (t *tui) updateServerListWithPingStatus() {
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	servers, _ := t.serverService.ListServers(query)
	sortServersForUI(servers, t.sortMode)

	for i := range servers {
		if ps, ok := t.pingStatuses[servers[i].Alias]; ok {
			servers[i].PingStatus = ps.PingStatus
			servers[i].PingLatency = ps.PingLatency
		}
	}

	t.serverList.UpdateServers(servers)
}

func (t *tui) handlePingAll() {
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	servers, err := t.serverService.ListServers(query)
	if err != nil {
		t.showStatusTempColor(fmt.Sprintf("Failed to get servers: %v", err), "#FF6B6B")
		return
	}

	if len(servers) == 0 {
		t.showStatusTemp("No servers to ping")
		return
	}

	t.showStatusTemp(fmt.Sprintf("Pinging all %d servers…", len(servers)))

	// Set all servers to checking status
	t.pingStatuses = make(map[string]domain.Server)
	for _, server := range servers {
		s := server
		s.PingStatus = StatusChecking
		t.pingStatuses[s.Alias] = s
	}
	t.updateServerListWithPingStatus()

	// Ping all servers concurrently
	for _, server := range servers {
		go func(srv domain.Server) {
			up, dur, err := t.serverService.Ping(srv)
			t.app.QueueUpdateDraw(func() {
				if ps, ok := t.pingStatuses[srv.Alias]; ok {
					if err != nil || !up {
						ps.PingStatus = StatusDown
						ps.PingLatency = 0
					} else {
						ps.PingStatus = StatusUp
						ps.PingLatency = dur
					}
					t.pingStatuses[srv.Alias] = ps
					t.updateServerListWithPingStatus()
				}
			})
		}(server)
	}

	// Show summary after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		t.app.QueueUpdateDraw(func() {
			upCount := 0
			downCount := 0
			for _, ps := range t.pingStatuses {
				if ps.PingStatus == StatusUp {
					upCount++
				} else if ps.PingStatus == StatusDown {
					downCount++
				}
			}
			t.showStatusTempColor(fmt.Sprintf("Ping completed: %d UP, %d DOWN", upCount, downCount), "#A0FFA0")
		})
	}()
}

func (t *tui) handleModalClose() {
	t.returnToMain()
}

// handleRefreshBackground refreshes the server list in the background without leaving the current screen.
// It preserves the current search query and selection, shows transient status, and avoids concurrent runs.
func (t *tui) handleRefreshBackground() {
	currentIdx := t.serverList.GetCurrentItem()
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}

	t.showStatusTemp("Refreshing…")

	go func(prevIdx int, q string) {
		servers, err := t.serverService.ListServers(q)
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.showStatusTempColor(fmt.Sprintf("Refresh failed: %v", err), "#FF6B6B")
			})
			return
		}
		if strings.TrimSpace(q) == "" {
			sortServersForUI(servers, t.sortMode)
		}
		t.app.QueueUpdateDraw(func() {
			t.serverList.UpdateServers(servers)
			// Try to restore selection if still valid
			if prevIdx >= 0 && prevIdx < t.serverList.List.GetItemCount() {
				t.serverList.SetCurrentItem(prevIdx)
				if srv, ok := t.serverList.GetSelectedServer(); ok {
					t.details.UpdateServer(srv)
				}
			}
			t.showStatusTemp(fmt.Sprintf("Refreshed %d servers", len(servers)))
		})
	}(currentIdx, query)
}

// =============================================================================
// UI Display Functions (show UI elements/modals)
// =============================================================================

func (t *tui) showDeleteConfirmModal(server domain.Server) {
	msg := fmt.Sprintf("Delete server %s (%s@%s:%d)?\n\nThis action cannot be undone.",
		server.Alias, server.User, server.Host, server.Port)

	doDelete := func() {
		err := t.serverService.DeleteServer(server)
		if err != nil {
			var ambig *domain.ErrAmbiguousHost
			if errors.As(err, &ambig) {
				t.showFileChoiceModal(ambig.Alias, ambig.Candidates, "Delete", func(chosen string) {
					srv := server
					srv.SourceFile = chosen
					if err := t.serverService.DeleteServer(srv); err != nil {
						t.showStatusTempColor("Delete failed: "+err.Error(), "#FF6B6B")
						return
					}
					t.showStatusTemp(fmt.Sprintf("Deleted %s from %s", srv.Alias, chosen))
					t.refreshServerList()
				})
				return
			}
			t.showStatusTempColor("Delete failed: "+err.Error(), "#FF6B6B")
			return
		}
		t.refreshServerList()
		t.handleModalClose()
	}

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"[yellow]C[-]ancel", "[yellow]D[-]elete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 1 {
				doDelete()
				return
			}
			t.handleModalClose()
		})

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'c', 'C':
			t.handleModalClose()
			return nil
		case 'd', 'D':
			doDelete()
			return nil
		}
		return event
	})

	t.app.SetRoot(modal, true)
}

// showFileChoiceModal asks the user which config file to apply an action to
// when the same alias is defined in multiple files. action is a verb shown
// on the confirmation buttons (e.g. "Save", "Delete"). onChoose is called
// with the chosen absolute file path; Cancel closes the modal.
func (t *tui) showFileChoiceModal(alias string, candidates []string, action string, onChoose func(path string)) {
	msg := fmt.Sprintf("Host %q is defined in multiple files.\nWhich file should %s use?", alias, action)
	buttons := append([]string{}, candidates...)
	buttons = append(buttons, "Cancel")

	modal := tview.NewModal().
		SetText(msg).
		AddButtons(buttons).
		SetDoneFunc(func(idx int, label string) {
			if idx < 0 || idx >= len(candidates) {
				t.handleModalClose()
				return
			}
			t.handleModalClose()
			onChoose(candidates[idx])
		})
	t.app.SetRoot(modal, true)
}

func (t *tui) showErrorModal(title, errMsg string) {
	text := fmt.Sprintf("[red]%s:[-]\n\n%s", tview.Escape(title), tview.Escape(errMsg))
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.handleModalClose()
		})
	t.app.SetRoot(modal, true)
	t.app.SetFocus(modal)
}

func (t *tui) showSSHErrorModal(alias, errMsg string) {
	t.showErrorModal(fmt.Sprintf("SSH connection to %q failed", alias), errMsg)
}

func (t *tui) showEditTagsForm(server domain.Server) {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit Tags: %s ", server.Alias)).
		SetTitleAlign(tview.AlignCenter)

	defaultTags := strings.Join(server.Tags, ", ")
	form.AddInputField("Tags (comma):", defaultTags, 40, nil, nil)

	form.AddButton("Save", func() {
		text := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		var tags []string

		for _, part := range strings.Split(text, ",") {
			if s := strings.TrimSpace(part); s != "" {
				tags = append(tags, s)
			}
		}

		newServer := server
		newServer.Tags = tags
		err := t.serverService.UpdateServer(server, newServer)
		if err != nil {
			var ambig *domain.ErrAmbiguousHost
			if errors.As(err, &ambig) {
				t.showFileChoiceModal(ambig.Alias, ambig.Candidates, "Update", func(chosen string) {
					orig := server
					orig.SourceFile = chosen
					nu := newServer
					nu.SourceFile = chosen
					if err := t.serverService.UpdateServer(orig, nu); err != nil {
						t.showStatusTempColor("Tags update failed: "+err.Error(), "#FF6B6B")
						return
					}
					t.refreshServerList()
					t.showStatusTemp("Tags updated")
				})
				return
			}
			t.showStatusTempColor("Tags update failed: "+err.Error(), "#FF6B6B")
			return
		}
		t.refreshServerList()
		t.returnToMain()
		t.showStatusTemp("Tags updated")
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	t.app.SetRoot(form, true)
	toFocus := form
	t.app.SetFocus(toFocus)
}

func (t *tui) handlePortForward() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.showPortForwardForm(server)
	}
}

func (t *tui) showPortForwardForm(server domain.Server) {
	typeChoices := []string{ForwardTypeLocal, ForwardTypeRemote, ForwardTypeDynamic}
	modeChoices := []string{ForwardModeOnlyForward, ForwardModeForwardSSH}

	currentTypeIdx := 0
	currentModeIdx := 0
	portVal := ""
	hostVal := "localhost"
	hostPortVal := ""
	bindAddrVal := ""

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Port Forwarding: %s ", server.Alias)).
		SetTitleAlign(tview.AlignCenter)

	dd := tview.NewDropDown()
	hostField := tview.NewInputField()
	hostPortField := tview.NewInputField()
	portField := tview.NewInputField()
	bindAddrField := tview.NewInputField()

	dd.SetOptions(typeChoices, func(text string, index int) {
		currentTypeIdx = index
		// Toggle fields when switching type
		isDynamic := typeChoices[currentTypeIdx] == ForwardTypeDynamic
		if isDynamic {
			hostField.SetText("").SetDisabled(true)
			hostPortField.SetText("").SetDisabled(true)
		} else {
			hostField.SetDisabled(false)
			hostPortField.SetDisabled(false)
		}
	})
	dd.SetCurrentOption(currentTypeIdx)
	form.AddFormItem(dd.SetLabel("Type"))

	portField.SetLabel("Port").SetText(portVal).SetFieldWidth(8).SetChangedFunc(func(text string) { portVal = strings.TrimSpace(text) })
	form.AddFormItem(portField)

	hostField.SetLabel("Host").SetText(hostVal).SetFieldWidth(40).SetChangedFunc(func(text string) { hostVal = strings.TrimSpace(text) })
	form.AddFormItem(hostField)

	hostPortField.SetLabel("Host Port").SetText(hostPortVal).SetFieldWidth(8).SetChangedFunc(func(text string) { hostPortVal = strings.TrimSpace(text) })
	form.AddFormItem(hostPortField)

	bindAddrField.SetLabel("Bind Address (optional)").SetText(bindAddrVal).SetFieldWidth(40).SetChangedFunc(func(text string) { bindAddrVal = strings.TrimSpace(text) })
	form.AddFormItem(bindAddrField)

	mode := tview.NewDropDown().SetOptions(modeChoices, func(text string, index int) { currentModeIdx = index })
	mode.SetCurrentOption(currentModeIdx)
	form.AddFormItem(mode.SetLabel("Mode"))

	isDynamic := typeChoices[currentTypeIdx] == ForwardTypeDynamic
	if isDynamic {
		hostField.SetText("").SetDisabled(true)
		hostPortField.SetText("").SetDisabled(true)
	}

	form.AddButton("Start", func() {
		if err := validatePort(portVal); err != nil {
			t.showStatusTempColor("Invalid port: "+err.Error(), "#FF6B6B")
			return
		}
		if bindAddrVal != "" {
			if err := validateBindAddress(bindAddrVal); err != nil {
				t.showStatusTempColor("Invalid bind address: "+err.Error(), "#FF6B6B")
				return
			}
		}

		ft := typeChoices[currentTypeIdx]
		var args []string
		if ft == ForwardTypeDynamic {
			spec := portVal
			if bindAddrVal != "" {
				spec = bindAddrVal + ":" + portVal
			}
			args = append(args, "-D", spec)
		} else {
			if err := validateHost(hostVal); err != nil {
				t.showStatusTempColor("Invalid host: "+err.Error(), "#FF6B6B")
				return
			}
			if err := validatePort(hostPortVal); err != nil {
				t.showStatusTempColor("Invalid host port: "+err.Error(), "#FF6B6B")
				return
			}
			spec := portVal + ":" + hostVal + ":" + hostPortVal
			if bindAddrVal != "" {
				spec = bindAddrVal + ":" + spec
			}
			if ft == ForwardTypeLocal {
				args = append(args, "-L", spec)
			} else {
				args = append(args, "-R", spec)
			}
		}

		onlyForward := modeChoices[currentModeIdx] == ForwardModeOnlyForward
		alias := server.Alias
		if onlyForward {
			t.returnToMain()
			t.showStatusTemp("Starting port forward…")
			go func() {
				pid, err := t.serverService.StartForward(alias, args)
				t.app.QueueUpdateDraw(func() {
					if err != nil {
						t.showStatusTempColor("Forward failed: "+err.Error(), "#FF6B6B")
					} else {
						t.refreshServerList()
						t.showStatusTemp(fmt.Sprintf("Port forwarding started (pid %d)", pid))
					}
				})
			}()
			return
		}

		var sshErr error
		t.app.Suspend(func() {
			if err := t.serverService.SSHWithArgs(alias, args); err != nil {
				sshErr = err
				t.logger.Errorw("ssh session error", "alias", alias, "error", err)
			}
		})
		t.app.Sync()
		t.returnToMain()
		if sshErr != nil {
			t.showSSHErrorModal(alias, sshErr.Error())
		}
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	t.app.SetRoot(form, true)
	t.app.SetFocus(form)
}

// =============================================================================
// UI State Management (hide UI elements)
// =============================================================================

// blurSearchBar moves focus back to the server list without changing layout.
func (t *tui) blurSearchBar() {
	if t.app != nil && t.serverList != nil {
		t.app.SetFocus(t.serverList)
	}
}

// =============================================================================
// Internal Operations (perform actual work)
// =============================================================================

func (t *tui) refreshServerList() {
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	filtered, _ := t.serverService.ListServers(query)
	if strings.TrimSpace(query) == "" {
		sortServersForUI(filtered, t.sortMode)
	}
	t.serverList.UpdateServers(filtered)
}

func (t *tui) returnToMain() {
	t.app.SetRoot(t.root, true)
}

// showStatusTemp displays a temporary message in the status bar (default green) and then restores the default text.
func (t *tui) showStatusTemp(msg string) {
	if t.statusBar == nil {
		return
	}
	t.showStatusTempColor(msg, "#A0FFA0")
}

// showStatusTempColor displays a temporary colored message in the status bar and restores default text after 2s.
func (t *tui) showStatusTempColor(msg string, color string) {
	if t.statusBar == nil {
		return
	}
	t.statusBar.SetText("[" + color + "]" + msg + "[-]")
	time.AfterFunc(2*time.Second, func() {
		if t.app != nil {
			t.app.QueueUpdateDraw(func() {
				if t.statusBar != nil {
					t.statusBar.SetText(DefaultStatusText())
				}
			})
		}
	})
}

// Stop any active port forwarding for the selected server.
func (t *tui) handleStopForwarding() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		alias := server.Alias
		go func() {
			err := t.serverService.StopForwarding(alias)
			t.app.QueueUpdateDraw(func() {
				if err != nil {
					t.showStatusTempColor("Failed to stop forwarding: "+err.Error(), "#FF6B6B")
				} else {
					t.showStatusTemp("Stopped forwarding for " + alias)
				}
				t.refreshServerList()
			})
		}()
	}
}
