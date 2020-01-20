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

package models

func (db *DB) CreateSchema() error {

	// ------------------------- Defining/importing models ---------------------------- //
	db.LogMode(true)

	// Workspaces
	db.AutoMigrate(&Workspace{})

	// Hosts
	db.AutoMigrate(&Host{})

	db.Model(&Host{}).AddForeignKey("workspace_id", "workspaces(id)", "CASCADE", "CASCADE")

	db.Exec(`CREATE INDEX index_hosts_on_workspace_id ON "hosts"("workspace_id") WHERE (workspace_id IS NOT NULL)`)

	// IP addresses
	db.AutoMigrate(&Address{})

	db.Model(&Address{}).AddForeignKey("host_id", "hosts(id)", "CASCADE", "CASCADE")

	// ------------------------- Default fields/items --------------------------------- //
	workspaces, _ := db.Workspaces()
	if len(workspaces) == 0 {
		db.AddWorkspaces([]string{"default"})

		updated, _ := db.Workspaces()
		updated[0].IsDefault = true
		db.UpdateWorkspace(*updated[0])
	}
	return nil
}
