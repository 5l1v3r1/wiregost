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

package console

import (
	"context"

	"github.com/maxlandon/wiregost/client/commands"
	"github.com/maxlandon/wiregost/data_service/models"
)

// ExecCmd executes a single command and provides it all the context it might need
func ExecCmd(args []string, menu string, ctx *context.Context, ws *models.Workspace, mod string) error {
	if len(args) < 1 {
		return nil
	}

	command := commands.FindCommand(menu, args[0])
	if command != nil {
		return command.Handle(commands.NewRequest(command, args[1:], ctx, ws, mod))
	}

	return nil
}
