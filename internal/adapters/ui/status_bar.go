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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func DefaultStatusText() string {
	return "[white]Tab[-] Panels  • [white]↑↓[-] Navigate  • [white]Enter[-] SSH  • [white]f[-] Forward  • [white]x[-] Stop Forward  • [white]c[-] Copy SSH  • [white]v[-] Paste SSH  • [white]y[-] Clone  • [white]h[-] Copy Host  • [white]a[-] Add  • [white]e[-] Edit  • [white]g/G[-] Ping (All)  • [white]K[-] Install Key  • [white]d[-] Delete  • [white]p[-] Pin/Unpin  • [white]i[-] Import  • [white]/[-] Search  • [white]q[-] Quit"
}

func ReadonlyStatusText() string {
	return "[white]Tab[-] Panels  • [white]↑↓[-] Navigate  • [white]Enter[-] SSH  • [white]f[-] Forward  • [white]x[-] Stop Forward  • [white]c[-] Copy SSH  • [white]h[-] Copy Host  • [white]g/G[-] Ping (All)  • [white]p[-] Pin/Unpin  • [white]/[-] Search  • [white]q[-] Quit  • [red::b][READONLY][-]"
}

func StatusText(readonly bool) string {
	if readonly {
		return ReadonlyStatusText()
	}
	return DefaultStatusText()
}

func NewStatusBar(readonly ...bool) *tview.TextView {
	ro := false
	if len(readonly) > 0 {
		ro = readonly[0]
	}
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBackgroundColor(tcell.Color235)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText(StatusText(ro))
	return status
}
