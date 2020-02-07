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

package templates

import (
	pb "github.com/maxlandon/wiregost/protobuf/client"
)

// Module - Contains all properties of a module
type Module struct {
	Name        string   `json:"name"`        // Name of the module
	Type        string   `json:"type"`        // Type of module (auxiliary, exploit, post, payload)
	Path        []string `json:"path"`        // Path to the module (ie. post/windows/x64/powershell/gather/powerview)
	Description string   `json:"description"` // Description of the module
	Notes       string   `json:"notes"`       // Notes about the module
	References  []string `json:"references"`  // A list of references to vulnerabilities/others (ie. CVEs)
	Author      []string `json:"author"`      // A list of module authors
	Credits     []string `json:"credits"`     // A list of people to credit for underlying tools/techniques
	Platform    string   `json:"platform"`    // Operating system the module can run on.
	Targets     []string `json:"targets"`     // A list of operating system versions the modules works on
	Arch        string   `json:"arch"`        // CPU architecture for which the module works
	Lang        string   `json:"lang"`        // Programming language in which the module is written
	Priviledged bool     `json:"priviledged"` // Does the module requires administrator privileges

	Options map[string]Option
}

// Option - Module option
type Option struct {
	Name        string `json:"name"`        // Name of the option
	Value       string `json:"value"`       // Value of the option (default is filled here)
	Required    bool   `json:"required"`    // Is this a required option ?
	Description string `json:"description"` // A description of the option
}

// ToProtobuf - Returns the protobuf version of a module
func (m *Module) ToProtobuf() *pb.Module {
	mod := &pb.Module{
		Name:        m.Name,
		Type:        m.Type,
		Path:        m.Path,
		Description: m.Description,
		Notes:       m.Notes,
		References:  m.References,
		Author:      m.Author,
		Credits:     m.Credits,
		Platform:    m.Platform,
		Targets:     m.Targets,
		Arch:        m.Arch,
		Lang:        m.Lang,
		Priviledged: m.Priviledged,
		Options:     map[string]*pb.Option{},
	}

	for name, opt := range m.Options {
		mod.Options[name] = opt.ToProtobuf()
	}

	return mod
}

// ToProtobuf - Returns the protobuf version of a module option
func (o *Option) ToProtobuf() *pb.Option {
	return &pb.Option{
		Name:        o.Name,
		Value:       o.Value,
		Required:    o.Required,
		Description: o.Description,
	}
}
