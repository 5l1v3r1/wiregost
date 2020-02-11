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

// CHANGE THE NAME OF THE PACKAGE WITH THE NAME OF YOUR MODULE/DIRECTORY
package reverse_dns

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	consts "github.com/maxlandon/wiregost/client/constants"
	pb "github.com/maxlandon/wiregost/protobuf/client"
	"github.com/maxlandon/wiregost/server/assets"
	"github.com/maxlandon/wiregost/server/c2"
	"github.com/maxlandon/wiregost/server/core"
	"github.com/maxlandon/wiregost/server/log"
	"github.com/maxlandon/wiregost/server/module/templates"
)

// metadataFile - Full path to module metadata
var metadataFile = filepath.Join(assets.GetModulesDir(), "payload/multi/single/reverse_dns/metadata.json")

// [ Base Methods ] ------------------------------------------------------------------------//

// ReverseDNS - A single stage MTLS implant
type ReverseDNS struct {
	Base *templates.Module
}

// New - Instantiates a reverse MTLS module, empty.
func New() *ReverseDNS {
	return &ReverseDNS{Base: &templates.Module{}}
}

// Init - Module initialization, loads metadata. ** DO NOT ERASE **
func (s *ReverseDNS) Init() error {
	return s.Base.Init(metadataFile)
}

// ToProtobuf - Returns protobuf version of module
func (s *ReverseDNS) ToProtobuf() *pb.Module {
	return s.Base.ToProtobuf()
}

// SetOption - Sets a module option through its base object.
func (s *ReverseDNS) SetOption(option, name string) {
	s.Base.SetOption(option, name)
}

// [ Module Methods ] ------------------------------------------------------------------------//

var rpcLog = log.ServerLogger("rpc", "server")

// Run - Module entrypoint. ** DO NOT ERASE **
func (s *ReverseDNS) Run(command string) (result string, err error) {

	switch command {

	case "to_listener":

		// Listener domains
		domains := strings.Split(s.Base.Options["ListenerDomains"].Value, ",")
		if (len(domains) == 1) && (domains[0] == "") {
			return "", errors.New("No domains provided for DNS listener")
		}
		for _, domain := range domains {
			if !strings.HasSuffix(domain, ".") {
				domain += "."
			}
		}

		// Implant canaries
		enableCanaries := true
		if s.Base.Options["DisableCanaries"].Value == "true" {
			enableCanaries = false
		}

		server := c2.StartDNSListener(domains, enableCanaries)
		description := fmt.Sprintf("%s (canaries %v)", strings.Join(domains, " "), enableCanaries)

		job := &core.Job{
			ID:          core.GetJobID(),
			Name:        "dns",
			Description: description,
			Protocol:    "udp",
			Port:        53,
			JobCtrl:     make(chan bool),
		}

		go func() {
			<-job.JobCtrl
			rpcLog.Infof("Stopping DNS listener (%d) ...", job.ID)
			server.Shutdown()

			core.Jobs.RemoveJob(job)

			core.EventBroker.Publish(core.Event{
				Job:       job,
				EventType: consts.StoppedEvent,
			})
		}()

		core.Jobs.AddJob(job)

		// There is no way to call DNS's ListenAndServe() without blocking
		// but we also need to check the error in the case the server
		// fails to start at all, so we setup all the Job mechanics
		// then kick off the server and if it fails we kill the job
		// ourselves.
		go func() {
			err := server.ListenAndServe()
			if err != nil {
				rpcLog.Errorf("DNS listener error %v", err)
				job.JobCtrl <- true
			}
		}()

		return fmt.Sprintf("Reverse DNS listener started with parent domain(s) %v...", domains), nil

	}

	return "Reverse DNS listener started", nil
}
