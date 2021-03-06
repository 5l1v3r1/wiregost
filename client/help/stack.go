// Wiregost - Golang Exploitation Framework
// Copyright © 2020 Para
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package help

import (
	"fmt"

	"github.com/evilsocket/islazy/tui"
)

var (
	stackHelp = fmt.Sprintf(`%s%s Stack Commands%s 

%s About:%s Manage the modules loaded on the current workspace' stack 

%s Commands:%s
    stack           %sList all modules currently loaded on the stack, and the current one%s
    stack use       %sUse a module from the stack (or not) (used with stack modules completion)%s
    stack pop       %sUnload one, more or all modules from the stack (will forget about current options)%s

%s Examples:%s
    stack pop                                       %sPop the current module from the stack (will fall back on next one)%s
    stack pop all                                   %sFlush the stack%s
    stack pop payload/multi/single/reverse_dns      %sUnload a precise module from the stack (modules are auto-completed)%s
    stack use payload/multi/single/reverse_https    %sSwitch to a module already on the stack (auto-completed)%s`,
		tui.BLUE, tui.BOLD, tui.RESET,
		tui.YELLOW, tui.RESET,
		tui.YELLOW, tui.RESET,
		tui.DIM, tui.RESET,
		tui.DIM, tui.RESET,
		tui.DIM, tui.RESET,
		tui.YELLOW, tui.RESET,
		tui.DIM, tui.RESET,
		tui.DIM, tui.RESET,
		tui.DIM, tui.RESET,
		tui.DIM, tui.RESET,
	)
)
