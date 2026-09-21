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
	"strings"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/WhiteRoseLK/neossh/internal/i18n"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SCPModal is a modal dialog for generating and copying SCP commands.
type SCPModal struct {
	*tview.Flex
	app         *tview.Application
	server      domain.Server
	form        *tview.Form
	infoText    *tview.TextView
	previewText *tview.TextView

	isUpload   bool
	localPath  string
	remotePath string
	recursive  bool
	useAlias   bool

	onCopied func(cmd string)
	onCancel func()
}

// NewSCPModal creates a new SCP command generator modal dialog.
func NewSCPModal(app *tview.Application, server domain.Server) *SCPModal {
	m := &SCPModal{
		Flex:        tview.NewFlex().SetDirection(tview.FlexRow),
		app:         app,
		server:      server,
		form:        tview.NewForm(),
		infoText:    tview.NewTextView(),
		previewText: tview.NewTextView(),
		isUpload:    true,
		localPath:   "./file.ext",
		remotePath:  "~/",
		recursive:   false,
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

	title := i18n.T("scp.title")
	if title == "scp.title" {
		title = " SCP Command Generator "
	}
	m.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft)

	return m
}

// OnCopied sets the callback called when the command is copied.
func (m *SCPModal) OnCopied(fn func(cmd string)) *SCPModal {
	m.onCopied = fn
	return m
}

// OnCancel sets the callback called when the modal is canceled.
func (m *SCPModal) OnCancel(fn func()) *SCPModal {
	m.onCancel = fn
	return m
}

// currentCommand returns the currently configured SCP command string.
func (m *SCPModal) currentCommand() string {
	return BuildSCPCommand(m.server, m.isUpload, m.localPath, m.remotePath, m.recursive, m.useAlias)
}

// updatePreview refreshes the preview TextView with the latest command.
func (m *SCPModal) updatePreview() {
	previewLabel := i18n.T("scp.preview")
	if previewLabel == "scp.preview" {
		previewLabel = "Generated Command:"
	}
	cmd := m.currentCommand()
	m.previewText.SetText(fmt.Sprintf("[%s::b]%s[-::-]\n[green::b]%s[-::-]",
		CurrentTheme.HintKey, previewLabel, cmd))
}

// Show builds and displays the SCP modal.
func (m *SCPModal) Show() error {
	m.form.Clear(true)

	// Display server info header
	target := m.server.Host
	if m.server.User != "" {
		target = fmt.Sprintf("%s@%s", m.server.User, m.server.Host)
	}
	if m.server.Port != 0 && m.server.Port != 22 {
		target = fmt.Sprintf("%s:%d", target, m.server.Port)
	}
	m.infoText.SetText(fmt.Sprintf("[yellow::b]%s[-::-] (%s)\n[gray]Configure transfer options and copy command to clipboard.[-]",
		m.server.Alias, target))

	dirLabel := i18n.T("scp.direction")
	if dirLabel == "scp.direction" {
		dirLabel = "Direction:"
	}
	uploadText := i18n.T("scp.upload")
	if uploadText == "scp.upload" {
		uploadText = "Upload (Local -> Remote)"
	}
	downloadText := i18n.T("scp.download")
	if downloadText == "scp.download" {
		downloadText = "Download (Remote -> Local)"
	}

	m.form.AddDropDown(dirLabel, []string{uploadText, downloadText}, 0, func(_ string, index int) {
		m.isUpload = (index == 0)
		m.updatePreview()
	})

	localLabel := i18n.T("scp.local_path")
	if localLabel == "scp.local_path" {
		localLabel = "Local Path:"
	}
	m.form.AddInputField(localLabel, m.localPath, 40, nil, func(text string) {
		m.localPath = strings.TrimSpace(text)
		m.updatePreview()
	})

	remoteLabel := i18n.T("scp.remote_path")
	if remoteLabel == "scp.remote_path" {
		remoteLabel = "Remote Path:"
	}
	m.form.AddInputField(remoteLabel, m.remotePath, 40, nil, func(text string) {
		m.remotePath = strings.TrimSpace(text)
		m.updatePreview()
	})

	recurLabel := i18n.T("scp.recursive")
	if recurLabel == "scp.recursive" {
		recurLabel = "Recursive (-r)"
	}
	m.form.AddCheckbox(recurLabel, m.recursive, func(checked bool) {
		m.recursive = checked
		m.updatePreview()
	})

	aliasLabel := i18n.T("scp.use_alias")
	if aliasLabel == "scp.use_alias" {
		aliasLabel = "Use SSH Alias"
	}
	m.form.AddCheckbox(aliasLabel, m.useAlias, func(checked bool) {
		m.useAlias = checked
		m.updatePreview()
	})

	btnCopy := i18n.T("scp.btn_copy")
	if btnCopy == "scp.btn_copy" {
		btnCopy = "Copy Command"
	}
	m.form.AddButton(btnCopy, func() {
		m.copyAndClose()
	})

	btnCancel := i18n.T("scp.btn_cancel")
	if btnCancel == "scp.btn_cancel" {
		btnCancel = "Cancel"
	}
	m.form.AddButton(btnCancel, func() {
		m.cancel()
	})

	m.updatePreview()

	m.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		//nolint:exhaustive // Only escape and ctrl keys handled
		switch event.Key() {
		case tcell.KeyEscape:
			m.cancel()
			return nil
		case tcell.KeyCtrlS:
			m.copyAndClose()
			return nil
		default:
			return event
		}
	})

	m.app.SetRoot(m, true)
	m.app.SetFocus(m.form)

	return nil
}

func (m *SCPModal) copyAndClose() {
	cmd := m.currentCommand()
	_ = clipboard.WriteAll(cmd)
	if m.onCopied != nil {
		m.onCopied(cmd)
	}
}

func (m *SCPModal) cancel() {
	if m.onCancel != nil {
		m.onCancel()
	}
}
