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

package gogo

import (
	"fmt"
	"testing"

	"github.com/maxlandon/wiregost/server/assets"
)

func TestGoGoVersion(t *testing.T) {
	appDir := assets.GetRootAppDir()
	winConfig := GoConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		GOROOT: GetGoRootDir(appDir),
	}
	_, err := GoVersion(winConfig)
	if err != nil {
		t.Errorf(fmt.Sprintf("version cmd failed %v", err))
	}
}
