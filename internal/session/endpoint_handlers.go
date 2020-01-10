package session

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/evilsocket/islazy/fs"
	"github.com/evilsocket/islazy/tui"
	"github.com/olekukonko/tablewriter"
)

func (s *Session) endpointConnect(cmd []string) {
	// If no endpoint provided, connect to default
	if len(cmd) == 2 {
		for _, v := range s.SavedEndpoints {
			if v.IsDefault {
				// In case where shell is already connected to a server, disconnect it
				if s.connection != nil {
					s.disconnect()
				}
				s.CurrentEndpoint = v
				s.connect()
			}
		}
	}

	if len(cmd) == 3 {
		recognized := false
		for _, v := range s.SavedEndpoints {
			if v.FQDN == cmd[2] {
				// In case where shell is already connected to a server, disconnect it
				if s.connection != nil {
					s.disconnect()
				}
				recognized = true
				s.CurrentEndpoint = v
				s.connect()
			}
		}
		if !recognized {
			fmt.Printf("%s[!]%s Endpoint name not in saved endpoints.'\n", tui.RED, tui.RESET)
		}
	}
}

func (s *Session) addEndpoint() error {
	// Load params from Env
	params := make(map[string]string)
	for k, v := range s.Env {
		if strings.HasPrefix(k, "endpoint") {
			params[k] = v
		}
	}

	// Check for parameters
	if _, ok := params["endpoint.name"]; !ok {
		fmt.Println()
		fmt.Printf("%s[!]%s %s is not set.", tui.RED, tui.RESET, "'endpoint.name'")
		fmt.Println()
		return nil
	}
	if _, ok := params["endpoint.address"]; !ok {
		fmt.Println()
		fmt.Printf("%s[!]%s %s is not set.", tui.RED, tui.RESET, "'endpoint.address'")
		fmt.Println()
		return nil
	}
	if _, ok := params["endpoint.port"]; !ok {
		fmt.Println()
		fmt.Printf("%s[!]%s %s is not set.", tui.RED, tui.RESET, "'endpoint.port'")
		fmt.Println()
		return nil
	}
	if _, ok := params["endpoint.certificate"]; !ok {
		fmt.Println()
		fmt.Printf("%s[!]%s %s is not set.", tui.RED, tui.RESET, "'endpoint.certificate'")
		fmt.Println()
		return nil
	}
	if _, ok := params["endpoint.key"]; !ok {
		fmt.Println()
		fmt.Printf("%s[!]%s %s is not set.", tui.RED, tui.RESET, "'endpoint.key'")
		fmt.Println()
		return nil
	}
	// Check for default
	isDefault := false
	if val, ok := params["endpoint.default"]; !ok {
	} else {
		if val == "true" {
			isDefault = true
		}
	}

	// Load into template, add to ServersList and save file
	port, _ := strconv.Atoi(params["endpoint.port"])
	template := Endpoint{
		IPAddress:   params["endpoint.address"],
		Port:        port,
		Certificate: params["endpoint.certificate"],
		Key:         params["endpoint.key"],
		FQDN:        params["endpoint.name"],
		IsDefault:   isDefault,
	}
	s.SavedEndpoints = append(s.SavedEndpoints, template)
	fmt.Println()
	fmt.Printf("%s[*]%s Added endpoint %s at %s:%d \n", tui.GREEN, tui.RESET, template.FQDN, template.IPAddress, template.Port)
	s.writeEndpointList()

	return nil
}

func (s *Session) deleteEndpoint(cmd []string) error {
	newList := s.SavedEndpoints[:0]
	for _, v := range s.SavedEndpoints {
		if v.FQDN != cmd[2] {
			newList = append(newList, v)
		}
	}
	s.SavedEndpoints = newList
	s.writeEndpointList()

	return nil
}

func (s *Session) setDefaultEndpoint(cmd []string) error {
	for _, v := range s.SavedEndpoints {
		if v.IsDefault == true {
			v.IsDefault = false
		}
		if v.FQDN == cmd[2] {
			v.IsDefault = true
		}
	}

	s.writeEndpointList()

	return nil
}

// List Servers
func (s *Session) listEndpoints() error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetCenterSeparator(tui.Dim("|"))
	table.SetRowSeparator(tui.Dim("-"))
	table.SetColumnSeparator(tui.Dim("|"))
	table.SetHeader([]string{"FQDN (Common Name)", "Address", "Certificate", "Connected", "Default"})
	table.SetAutoWrapText(true)
	table.SetColWidth(80)
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Normal, tablewriter.FgHiBlackColor},
		tablewriter.Colors{tablewriter.Normal, tablewriter.FgHiBlackColor},
		tablewriter.Colors{tablewriter.Normal, tablewriter.FgHiBlackColor},
		tablewriter.Colors{tablewriter.Normal, tablewriter.FgHiBlackColor},
		tablewriter.Colors{tablewriter.Normal, tablewriter.FgHiBlackColor},
	)

	table.SetBorder(false)

	connected := ""
	defaultEndpoint := ""
	for _, l := range s.SavedEndpoints {
		if s.CurrentEndpoint == l {
			connected = tui.Green("Connected")
		} else {
			connected = ""
		}
		if l.IsDefault {
			defaultEndpoint = "default"
		} else {
			defaultEndpoint = ""
		}
		table.Append([]string{l.FQDN, l.IPAddress + ":" + strconv.Itoa(l.Port), l.Certificate, connected, defaultEndpoint})
	}

	fmt.Println()
	table.Render()

	return nil
}

func (s *Session) writeEndpointList() error {
	endpointFile, _ := fs.Expand("~/.wiregost/client/server.conf")
	var servConf *os.File
	if !fs.Exists(endpointFile) {
		servConf, _ = os.Create(endpointFile)
	} else {
		servConf, _ = os.Open(endpointFile)
	}
	defer servConf.Close()

	// Marshal to JSON
	var jsonData []byte
	jsonData, err := json.MarshalIndent(s.SavedEndpoints, "", "    ")
	if err != nil {
		fmt.Println("Error: Failed to write JSON data to server configuration file")
		fmt.Println(err)
	} else {
		_ = ioutil.WriteFile(endpointFile, jsonData, 0755)
	}

	return nil
}

// ----------------------------------------------------------------------
// ENDPOINT LOADING

// Endpoint is a struct used to load, add delete and connect to a Wiregost endpoint.
type Endpoint struct {
	IPAddress   string
	Port        int
	Certificate string
	Key         string
	FQDN        string
	IsDefault   bool
}

func (s *Session) loadEndpointList() error {
	serverList := []Endpoint{}

	userDir, _ := fs.Expand("~/.wiregost/client/")
	if !fs.Exists(userDir) {
		os.MkdirAll(userDir, 0755)
		fmt.Println(tui.Dim("User directory was not found: creating ~/.wiregost/client/"))
	}
	path, _ := fs.Expand("~/.wiregost/client/server.conf")
	if !fs.Exists(path) {
		fmt.Println(tui.Red("Endpoint Configuration file not found: check for issues," +
			" or run the configuration script again"))
		os.Exit(1)
	} else {
		configBlob, _ := ioutil.ReadFile(path)
		json.Unmarshal(configBlob, &serverList)
	}

	// Format certificate path for each server, add server to EndpointManager
	for _, i := range serverList {
		i.Certificate, _ = fs.Expand(i.Certificate)
		s.SavedEndpoints = append(s.SavedEndpoints,
			Endpoint{IPAddress: i.IPAddress,
				Port:        i.Port,
				Certificate: i.Certificate,
				Key:         i.Key,
				FQDN:        i.FQDN,
				IsDefault:   i.IsDefault})
	}
	return nil
}

func (s *Session) getDefaultEndpoint() error {
	for _, i := range s.SavedEndpoints {
		if i.IsDefault == true {
			s.CurrentEndpoint = i
			break
		}
	}
	return nil
}
