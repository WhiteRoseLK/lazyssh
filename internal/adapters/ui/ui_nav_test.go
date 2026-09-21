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
	"strings"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

type navTestMockService struct {
	mockServerService
	deletedServer domain.Server
	deleteCalled  bool
}

func (m *navTestMockService) DeleteServer(s domain.Server) error {
	m.deleteCalled = true
	m.deletedServer = s
	return nil
}

func setupNavTestTUI() (*tui, *navTestMockService, tcell.SimulationScreen) {
	logger := zap.NewNop().Sugar()
	srv := domain.Server{Alias: "srv1", Host: "1.2.3.4", Port: 22, User: "root"}
	svc := &navTestMockService{
		mockServerService: mockServerService{servers: []domain.Server{srv}},
	}

	appInstance := NewTUI(logger, svc, "v1.0.0", "abc").(*tui)
	simScreen := tcell.NewSimulationScreen("UTF-8")
	_ = simScreen.Init()
	appInstance.app.SetScreen(simScreen)

	appInstance.buildComponents()
	appInstance.buildLayout()
	appInstance.bindEvents()
	appInstance.serverList.UpdateServers(svc.servers)
	appInstance.app.SetRoot(appInstance.root, true)
	appInstance.app.SetFocus(appInstance.serverList)

	return appInstance, svc, simScreen
}

func simulationScreenContent(screen tcell.SimulationScreen) string {
	cells, _, _ := screen.GetContents()
	var text strings.Builder
	for _, cell := range cells {
		for _, r := range cell.Runes {
			text.WriteRune(r)
		}
	}
	return text.String()
}

func TestNormalizeGlobalHotkey(t *testing.T) {
	tests := map[rune]rune{
		'e': 'e',
		'E': 'e',
		'd': 'd',
		'D': 'd',
		's': 's',
		'S': 'S',
		'g': 'g',
		'G': 'G',
		'k': 'k',
		'K': 'K',
		'c': 'c',
		'C': 'C',
		'y': 'y',
		'Y': 'y',
		'q': 'q',
		'Q': 'q',
		'!': '!',
		'.': '.',
		'1': '1',
	}

	for input, expected := range tests {
		if actual := normalizeGlobalHotkey(input); actual != expected {
			t.Fatalf("normalizeGlobalHotkey(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestCommandKeyPreservesNonLatinRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected rune
	}{
		{name: "latin lower", input: 'd', expected: 'd'},
		{name: "latin caps", input: 'D', expected: 'd'},
		{name: "non-latin rune", input: '\u03bb', expected: '\u03bb'},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := tcell.NewEventKey(tcell.KeyRune, test.input, tcell.ModNone)
			if actual := commandKey(event); actual != test.expected {
				t.Fatalf("commandKey(%q) = %q, want %q", test.input, actual, test.expected)
			}
		})
	}
}

func TestPanelNavigation_TabAndBacktabCycling(t *testing.T) {
	app, _, _ := setupNavTestTUI()

	// Initial focus is serverList
	app.handleServerListFocus()
	if app.app.GetFocus() != app.serverList {
		t.Fatalf("expected serverList to have focus")
	}

	// Tab from serverList -> details
	app.handleNextPanel()
	if app.app.GetFocus() != app.details {
		t.Fatalf("expected details to have focus after Tab from serverList")
	}

	// Tab from details -> searchBar
	app.handleNextPanel()
	if app.app.GetFocus() != app.searchBar {
		t.Fatalf("expected searchBar to have focus after Tab from details")
	}

	// Tab from searchBar -> serverList
	app.handleNextPanel()
	if app.app.GetFocus() != app.serverList {
		t.Fatalf("expected serverList to have focus after Tab from searchBar")
	}

	// Shift-Tab from serverList -> searchBar
	app.handlePrevPanel()
	if app.app.GetFocus() != app.searchBar {
		t.Fatalf("expected searchBar to have focus after Shift-Tab from serverList")
	}

	// Shift-Tab from searchBar -> details
	app.handlePrevPanel()
	if app.app.GetFocus() != app.details {
		t.Fatalf("expected details to have focus after Shift-Tab from searchBar")
	}

	// Shift-Tab from details -> serverList
	app.handlePrevPanel()
	if app.app.GetFocus() != app.serverList {
		t.Fatalf("expected serverList to have focus after Shift-Tab from details")
	}
}

func TestPanelNavigation_NumericKeys(t *testing.T) {
	app, _, _ := setupNavTestTUI()

	// '0' focuses search
	app.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, '0', tcell.ModNone))
	if app.app.GetFocus() != app.searchBar {
		t.Fatalf("expected '0' to focus searchBar")
	}

	// Blur search bar
	app.blurSearchBar()
	if app.app.GetFocus() != app.serverList {
		t.Fatalf("expected blurSearchBar to restore focus to serverList")
	}

	// '2' focuses details
	app.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone))
	if app.app.GetFocus() != app.details {
		t.Fatalf("expected '2' to focus details")
	}

	// '1' focuses serverList
	app.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone))
	if app.app.GetFocus() != app.serverList {
		t.Fatalf("expected '1' to focus serverList")
	}

	// '3' also focuses details
	app.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, '3', tcell.ModNone))
	if app.app.GetFocus() != app.details {
		t.Fatalf("expected '3' to focus details")
	}
}

func TestFocusBorderColors(t *testing.T) {
	app, _, _ := setupNavTestTUI()

	// Focus serverList
	app.handleServerListFocus()
	app.updateFocusBorders()

	// Check focused styling on serverList vs searchBar
	// serverList should have focused border color, searchBar unfocused
	if app.serverList.GetBorderColor() != BorderColorFocused {
		t.Fatalf("serverList border color = %v, want %v", app.serverList.GetBorderColor(), BorderColorFocused)
	}
	if app.searchBar.GetBorderColor() != BorderColorUnfocused {
		t.Fatalf("searchBar border color = %v, want %v", app.searchBar.GetBorderColor(), BorderColorUnfocused)
	}
	if app.details.GetBorderColor() != BorderColorUnfocused {
		t.Fatalf("details border color = %v, want %v", app.details.GetBorderColor(), BorderColorUnfocused)
	}

	// Switch focus to searchBar
	app.handleSearchFocus()
	app.updateFocusBorders()
	if app.searchBar.GetBorderColor() != BorderColorFocused {
		t.Fatalf("searchBar border color = %v, want %v", app.searchBar.GetBorderColor(), BorderColorFocused)
	}
	if app.serverList.GetBorderColor() != BorderColorUnfocused {
		t.Fatalf("serverList border color = %v, want %v", app.serverList.GetBorderColor(), BorderColorUnfocused)
	}

	// Switch focus to details
	app.handleDetailsFocus()
	app.updateFocusBorders()
	if app.details.GetBorderColor() != BorderColorFocused {
		t.Fatalf("details border color = %v, want %v", app.details.GetBorderColor(), BorderColorFocused)
	}
	if app.serverList.GetBorderColor() != BorderColorUnfocused {
		t.Fatalf("serverList border color = %v, want %v", app.serverList.GetBorderColor(), BorderColorUnfocused)
	}
}

func TestDeleteConfirmationModal_CancelAndEscape(t *testing.T) {
	app, svc, screen := setupNavTestTUI()
	srv := svc.servers[0]

	modal, pages := app.newDeleteConfirmationOverlay(srv, "Delete server srv1?")
	app.app.SetRoot(pages, true)
	app.app.SetFocus(modal)
	app.app.ForceDraw()

	content := simulationScreenContent(screen)
	if !strings.Contains(content, "Confirm Deletion") {
		t.Fatalf("modal title 'Confirm Deletion' not rendered on screen")
	}

	// Pressing escape cancels and restores root + focus
	modal.InputHandler()(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), func(p tview.Primitive) {
		app.app.SetFocus(p)
	})

	if svc.deleteCalled {
		t.Fatalf("escape unexpectedly triggered server deletion")
	}
	if app.app.GetFocus() != app.serverList {
		t.Fatalf("expected focus restored to serverList, got %v", app.app.GetFocus())
	}
}

func TestDeleteConfirmationModal_ConfirmDelete(t *testing.T) {
	app, svc, _ := setupNavTestTUI()
	srv := svc.servers[0]

	modal, pages := app.newDeleteConfirmationOverlay(srv, "Delete server srv1?")
	app.app.SetRoot(pages, true)
	app.app.SetFocus(modal)

	// Pressing 'd' triggers deletion
	modal.InputHandler()(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone), func(p tview.Primitive) {
		app.app.SetFocus(p)
	})

	if !svc.deleteCalled {
		t.Fatalf("expected server deletion to be called on 'd'")
	}
	if svc.deletedServer.Alias != srv.Alias {
		t.Fatalf("deleted server alias = %q, want %q", svc.deletedServer.Alias, srv.Alias)
	}
	if app.app.GetFocus() != app.serverList {
		t.Fatalf("expected focus restored to serverList after delete, got %v", app.app.GetFocus())
	}
}

func TestDeleteConfirmationModal_BlocksClicksBehindModal(t *testing.T) {
	app, svc, _ := setupNavTestTUI()
	srv := svc.servers[0]

	modal, pages := app.newDeleteConfirmationOverlay(srv, "Delete server srv1?")
	app.app.SetRoot(pages, true)
	app.app.SetFocus(modal)
	app.app.ForceDraw()

	if modal.InRect(0, 0) {
		t.Fatal("test coordinate unexpectedly inside modal rect")
	}

	consumed, _ := pages.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone),
		func(p tview.Primitive) { app.app.SetFocus(p) },
	)
	if !consumed {
		t.Fatal("click outside confirmation modal was not consumed")
	}
}

func TestHighlightedFormItem_And_UnwrapFormItem(t *testing.T) {
	input := tview.NewInputField().SetLabel("Test:")
	wrapped := highlightFormItem(input)

	if unwrapped := unwrapFormItem(wrapped); unwrapped != input {
		t.Fatalf("unwrapFormItem did not return the original input field")
	}

	if unwrappedNormal := unwrapFormItem(input); unwrappedNormal != input {
		t.Fatalf("unwrapFormItem on unwrapped item did not return the item itself")
	}
}

func TestMoveFormFocus(t *testing.T) {
	form := tview.NewForm()
	f1 := highlightFormItem(tview.NewInputField().SetLabel("Field1:"))
	f2 := highlightFormItem(tview.NewInputField().SetLabel("Field2:"))
	form.AddFormItem(f1)
	form.AddFormItem(f2)
	form.AddButton("Submit", nil)

	form.SetFocus(0)
	f1.Focus(func(p tview.Primitive) {})
	itemIdx, _ := form.GetFocusedItemIndex()
	if itemIdx != 0 {
		t.Fatalf("expected initial focused item index 0, got %d", itemIdx)
	}

	// Move forward
	moved := moveFormFocus(form, 1)
	if !moved {
		t.Fatalf("expected moveFormFocus(1) to return true")
	}
	itemIdx, _ = form.GetFocusedItemIndex()
	if itemIdx != 1 {
		t.Fatalf("expected focused item index 1, got %d", itemIdx)
	}

	// Move backward
	moved = moveFormFocus(form, -1)
	if !moved {
		t.Fatalf("expected moveFormFocus(-1) to return true")
	}
	itemIdx, _ = form.GetFocusedItemIndex()
	if itemIdx != 0 {
		t.Fatalf("expected focused item index 0, got %d", itemIdx)
	}
}
