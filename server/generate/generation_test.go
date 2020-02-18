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

package generate

import (
	"fmt"
	"testing"

	clientpb "github.com/maxlandon/wiregost/protobuf/client"
	"github.com/maxlandon/wiregost/server/log"
)

var (
	buildTestLog = log.ServerLogger("generate", "testbuild")
)

func TestGhostExecutableWindows(t *testing.T) {

	// mTLS C2
	mtlsExe(t, "windows", "amd64", false)
	//mtlsExe(t, "windows", "386", false)
	mtlsExe(t, "windows", "amd64", true)
	//mtlsExe(t, "windows", "386", true)

	// DNS C2
	dnsExe(t, "windows", "amd64", false)
	//dnsExe(t, "windows", "386", false)
	dnsExe(t, "windows", "amd64", true)
	//dnsExe(t, "windows", "386", true)

	// HTTP C2
	httpExe(t, "windows", "amd64", false)
	//httpExe(t, "windows", "386", false)
	httpExe(t, "windows", "amd64", true)
	//httpExe(t, "windows", "386", true)

	// Multiple C2s
	multiExe(t, "windows", "amd64", true)
	multiExe(t, "windows", "amd64", false)
	multiExe(t, "windows", "386", false)
	multiExe(t, "windows", "386", false)
}

func TestGhostSharedLibWindows(t *testing.T) {
	multiLibrary(t, "windows", "amd64", true)
	multiLibrary(t, "windows", "amd64", false)
	multiLibrary(t, "windows", "386", true)
	multiLibrary(t, "windows", "386", false)
}

func TestGhostExecutableLinux(t *testing.T) {
	multiExe(t, "linux", "amd64", true)
	multiExe(t, "linux", "amd64", false)
}

func TestGhostExecutableDarwin(t *testing.T) {
	multiExe(t, "darwin", "amd64", true)
	multiExe(t, "darwin", "amd64", false)
}

// func TestSymbolObfuscation(t *testing.T) {
// 	symbolObfuscation(t, "windows", "amd64")
// }

func mtlsExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[mtls] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &GhostConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []GhostC2{
			GhostC2{URL: "mtls://1.example.com"},
		},
		MTLSc2Enabled:    true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	_, err := GhostExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func dnsExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[dns] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &GhostConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []GhostC2{
			GhostC2{URL: "dns://3.example.com"},
		},
		DNSc2Enabled:     true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	_, err := GhostExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func httpExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[http] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &GhostConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []GhostC2{
			GhostC2{
				Priority: 1,
				URL:      "http://4.example.com",
				Options:  "asdf",
			},
		},
		HTTPc2Enabled:    true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	_, err := GhostExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func multiExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	config := &GhostConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []GhostC2{
			GhostC2{URL: "mtls://1.example.com"},
			GhostC2{URL: "mtls://2.example.com", Options: "asdf"},
			GhostC2{URL: "https://3.example.com"},
			GhostC2{Priority: 3, URL: "dns://4.example.com"},
		},
		MTLSc2Enabled:    true,
		HTTPc2Enabled:    true,
		DNSc2Enabled:     true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	_, err := GhostExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func multiLibrary(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[multi] LIB %s/%s - debug: %v", goos, goarch, debug)
	config := &GhostConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []GhostC2{
			GhostC2{URL: "mtls://1.example.com"},
			GhostC2{Priority: 2, URL: "mtls://2.example.com"},
			GhostC2{URL: "https://3.example.com"},
			GhostC2{URL: "dns://4.example.com", Options: "asdf"},
		},

		Debug:            debug,
		ObfuscateSymbols: false,
		Format:           clientpb.GhostConfig_SHARED_LIB,
		IsSharedLib:      true,
	}
	_, err := GhostSharedLibrary(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func symbolObfuscation(t *testing.T, goos string, goarch string) {
	t.Logf("[symbol obfuscation] %s/%s ...", goos, goarch)
	config := &GhostConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []GhostC2{
			GhostC2{URL: "mtls://1.example.com"},
			GhostC2{Priority: 2, URL: "mtls://2.example.com"},
			GhostC2{URL: "https://3.example.com"},
			GhostC2{URL: "dns://4.example.com", Options: "asdf"},
		},

		Debug:            false,
		ObfuscateSymbols: true,
	}
	_, err := GhostExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}
