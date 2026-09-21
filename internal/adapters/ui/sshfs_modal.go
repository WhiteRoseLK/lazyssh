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

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/WhiteRoseLK/neossh/internal/i18n"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SSHFSModal is a modal dialog for generating and copying SSHFS mount commands.
type SSHFSModal struct {
	*tview.Flex
	app         *tview.Application
	server      domain.Server
	form        *tview.Form
	infoText    *tview.TextView
	previewText *tview.TextView

	remotePath string
	mountPoint string
	reconnect  bool
	readOnly   bool
	useAlias   bool

	onCopied func(cmd string)
	onCancel func()
}

// NewSSHFSModal creates a new SSHFS mount generator modal dialog.
func NewSSHFSModal(app *tview.Application, server domain.Server) *SSHFSModal {
	defaultMount := "~/mounts/" + server.Alias
	if server.Alias == "" {
		defaultMount = "~/mounts/" + server.Host
	}

	m := &SSHFSModal{
		Flex:        tview.NewFlex().SetDirection(tview.FlexRow),
		app:         app,
		server:      server,
		form:        tview.NewForm(),
		infoText:    tview.NewTextView(),
		previewText: tview.NewTextView(),
		remotePath:  "/",
		mountPoint:  defaultMount,
		reconnect:   true,
		readOnly:    false,
		useAlias:    false,
	}

	m.infoText.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1)

	m.previewText.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1)

	m.form.SetBorderPadding(0, 0, 1, 1)

	m.AddItem(m.infoText, 3, 0, false).
		AddItem(m.form, 0, 1, true).
		AddItem(m.previewText, 4, 0, false)

	title := i18n.T("sshfs.title")
	if title == "sshfs.title" {
		title = " SSHFS Remote Mount Generator "
	}
	m.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft)

	return m
}

// OnCopied sets the callback called when a command is copied.
func (m *SSHFSModal) OnCopied(fn func(cmd string)) *SSHFSModal {
	m.onCopied = fn
	return m
}

// OnCancel sets the callback called when the modal is canceled.
func (m *SSHFSModal) OnCancel(fn func()) *SSHFSModal {
	m.onCancel = fn
	return m
}

// currentMountCommand returns the currently configured SSHFS mount command string.
func (m *SSHFSModal) currentMountCommand() string {
	return BuildSSHFSCommand(m.server, m.remotePath, m.mountPoint, m.readOnly, m.reconnect, m.useAlias)
}

// currentUnmountCommand returns the unmount command string for the mount point.
func (m *SSHFSModal) currentUnmountCommand() string {
	return BuildSSHFSUnmountCommand(m.mountPoint)
}

// updatePreview refreshes the preview TextView with the latest commands.
func (m *SSHFSModal) updatePreview() {
	mountCmd := m.currentMountCommand()
	unmountCmd := m.currentUnmountCommand()

	m.previewText.SetText(fmt.Sprintf("[%s::b]Mount Command:[-::-] [green::b]%s[-::-]\n[%s::b]Unmount:[-::-] [yellow]%s[-]",
		CurrentTheme.HintKey, mountCmd, CurrentTheme.HintKey, unmountCmd))
}

// Show builds and displays the SSHFS modal.
func (m *SSHFSModal) Show() error {
	m.form.Clear(true)

	// Display server info header
	target := m.server.Host
	if m.server.User != "" {
		target = fmt.Sprintf("%s@%s", m.server.User, m.server.Host)
	}
	if m.server.Port != 0 && m.server.Port != 22 {
		target = fmt.Sprintf("%s:%d", target, m.server.Port)
	}
	m.infoText.SetText(fmt.Sprintf("[yellow::b]%s[-::-] (%s)\n[gray]Configure remote mount options and copy command to clipboard.[-]",
		m.server.Alias, target))

	m.form.AddInputField("Remote Path:", m.remotePath, 35, nil, func(text string) {
		m.remotePath = text
		m.updatePreview()
	})

	m.form.AddInputField("Local Mount Point:", m.mountPoint, 35, nil, func(text string) {
		m.mountPoint = text
		m.updatePreview()
	})

	m.form.AddCheckbox("Auto-Reconnect (-o reconnect):", m.reconnect, func(checked bool) {
		m.reconnect = checked
		m.updatePreview()
	})

	m.form.AddCheckbox("Read-Only (-o ro):", m.readOnly, func(checked bool) {
		m.readOnly = checked
		m.updatePreview()
	})

	m.form.AddCheckbox("Use SSH Config Alias:", m.useAlias, func(checked bool) {
		m.useAlias = checked
		m.updatePreview()
	})

	m.form.AddButton("Copy Mount Command", func() {
		cmd := m.currentMountCommand()
		_ = clipboard.WriteAll(cmd)
		if m.onCopied != nil {
			m.onCopied(cmd)
		}
	})

	m.form.AddButton("Copy Unmount Command", func() {
		cmd := m.currentUnmountCommand()
		_ = clipboard.WriteAll(cmd)
		if m.onCopied != nil {
			m.onCopied(cmd)
		}
	})

	m.form.AddButton("Cancel", func() {
		if m.onCancel != nil {
			m.onCancel()
		}
	})

	m.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if m.onCancel != nil {
				m.onCancel()
			}
			return nil
		}
		return event
	})

	m.updatePreview()

	// Show modal in app
	m.app.SetRoot(m, true)
	m.app.SetFocus(m.form)
	return nil
}
