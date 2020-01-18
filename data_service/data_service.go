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

package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/evilsocket/islazy/tui"
	"github.com/maxlandon/wiregost/data_service/handlers"
)

func main() {
	// Setup DB and environment -------------------------------

	// Load environment configuration: DB credentials and data_service parameters
	env := handlers.LoadEnv()

	// Check if DB schema exists
	if env.DB.SchemaIsUpdated() {
		fmt.Println(tui.Green("[*] Database schema is up-to-date"))
	} else {
		fmt.Println(tui.Red("[!] Database schema is not up-to-date: run function for updating it"))
		// return
	}

	// Create Schema
	// err := env.DB.CreateSchema()
	// if err != nil {
	//         fmt.Println(err.Error())
	// }

	// Instantiate ServerMultiplexer
	mux := http.NewServeMux()

	// Register handlers ---------------------------------------
	wh := &handlers.WorkspaceHandler{env}
	mux.Handle(handlers.WorkspaceAPIPath, wh)

	hh := &handlers.HostHandler{env}
	mux.Handle(handlers.HostAPIPath, hh)

	sh := &handlers.ServiceHandler{env}
	mux.Handle(handlers.ServiceAPIPath, sh)

	ch := &handlers.CredentialHandler{env}
	mux.Handle(handlers.CredentialAPIPath, ch)

	// Start server --------------------------------------------
	fmt.Println("Listening for requests...")
	http.ListenAndServeTLS(env.Service.Address+":"+strconv.Itoa(env.Service.Port),
		env.Service.Certificate, env.Service.Key, mux)
}
