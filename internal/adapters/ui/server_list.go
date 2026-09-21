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
	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

type ServerList struct {
	*tview.List
	servers           []domain.Server
	currentWidth      int
	onSelection       func(domain.Server)
	onSelectionChange func(domain.Server)
	onReturnToSearch  func()
	onTab             func()
	onBacktab         func()
}

func NewServerList() *ServerList {
	list := &ServerList{
		List: tview.NewList(),
	}
	list.build()
	return list
}

func (sl *ServerList) build() {
	sl.List.ShowSecondaryText(false)
	sl.List.SetBorder(true).
		SetTitle(" 1 Servers ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(CurrentTheme.BorderColorUnfocused).
		SetTitleColor(CurrentTheme.TitleColorUnfocused)
	sl.List.
		SetSelectedBackgroundColor(CurrentTheme.SelectedBackground).
		SetSelectedTextColor(CurrentTheme.SelectedText).
		SetHighlightFullLine(true)

	sl.List.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(sl.servers) && sl.onSelectionChange != nil {
			sl.onSelectionChange(sl.servers[index])
		}
	})

	sl.List.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		//nolint:exhaustive // We only handle specific keys and pass through others
		switch event.Key() {
		case tcell.KeyTab:
			if sl.onTab != nil {
				sl.onTab()
				return nil
			}
		case tcell.KeyBacktab:
			if sl.onBacktab != nil {
				sl.onBacktab()
				return nil
			}
		case tcell.KeyLeft, tcell.KeyRight, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyESC:
			if sl.onReturnToSearch != nil {
				sl.onReturnToSearch()
			}
			return nil
		}
		return event
	})
}

func (sl *ServerList) UpdateServers(servers []domain.Server) {
	currentAlias := ""
	if idx := sl.List.GetCurrentItem(); idx >= 0 && idx < len(sl.servers) {
		currentAlias = sl.servers[idx].Alias
	}

	sl.servers = servers
	sl.List.Clear()

	// Calculate the maximum alias width for alignment
	maxAliasWidth := 0
	for _, s := range servers {
		width := runewidth.StringWidth(s.Alias)
		if width > maxAliasWidth {
			maxAliasWidth = width
		}
	}

	_, _, listWidth, _ := sl.List.GetInnerRect() //nolint:dogsled // only width is needed
	sl.currentWidth = listWidth

	for i := range servers {
		primary, secondary := formatServerLine(servers[i], maxAliasWidth, listWidth)
		idx := i
		sl.List.AddItem(primary, secondary, 0, func() {
			if sl.onSelection != nil {
				sl.onSelection(sl.servers[idx])
			}
		})
	}

	restoreIdx := 0
	if currentAlias != "" {
		for i, s := range servers {
			if s.Alias == currentAlias {
				restoreIdx = i
				break
			}
		}
	}

	if sl.List.GetItemCount() > 0 {
		sl.List.SetCurrentItem(restoreIdx)
		if sl.onSelectionChange != nil {
			sl.onSelectionChange(sl.servers[restoreIdx])
		}
	}
}

// RefreshDisplay re-renders the list if the component width has changed
func (sl *ServerList) RefreshDisplay() {
	_, _, width, _ := sl.List.GetInnerRect() //nolint:dogsled // only width is needed
	if width != sl.currentWidth && width > 0 {
		sl.currentWidth = width
		currentIdx := sl.List.GetCurrentItem()
		sl.UpdateServers(sl.servers)
		if currentIdx >= 0 && currentIdx < sl.List.GetItemCount() {
			sl.List.SetCurrentItem(currentIdx)
		}
	}
}

func (sl *ServerList) GetSelectedServer() (domain.Server, bool) {
	idx := sl.List.GetCurrentItem()
	if idx >= 0 && idx < len(sl.servers) {
		return sl.servers[idx], true
	}
	return domain.Server{}, false
}

func (sl *ServerList) OnSelection(fn func(server domain.Server)) *ServerList {
	sl.onSelection = fn
	return sl
}

func (sl *ServerList) OnSelectionChange(fn func(server domain.Server)) *ServerList {
	sl.onSelectionChange = fn
	return sl
}

func (sl *ServerList) OnReturnToSearch(fn func()) *ServerList {
	sl.onReturnToSearch = fn
	return sl
}

func (sl *ServerList) OnTab(fn func()) *ServerList {
	sl.onTab = fn
	return sl
}

func (sl *ServerList) OnBacktab(fn func()) *ServerList {
	sl.onBacktab = fn
	return sl
}
