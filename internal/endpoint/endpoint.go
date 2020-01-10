package endpoint

import (
	// Standard
	"net"
	"strings"

	// 3rd party
	uuid "github.com/satori/go.uuid"

	// Wiregost
	"github.com/maxlandon/wiregost/internal/logging"
	"github.com/maxlandon/wiregost/internal/messages"
	"github.com/maxlandon/wiregost/internal/user"
)

// Endpoint manages all connections and message passing between client shells
// and the Wiregost components/managers.
type Endpoint struct {
	IPAddress string
	clients   []*Client
	connect   chan net.Conn
	requests  chan messages.ClientRequest
}

// NewEndpoint instantiates a new Endpoint object, which handles all requests
// from clients and responses from managers.
func NewEndpoint() *Endpoint {
	e := &Endpoint{
		IPAddress: getEndpointIP(),
		clients:   make([]*Client, 0),
		connect:   make(chan net.Conn),
		requests:  make(chan messages.ClientRequest),
	}

	go e.listen()
	go e.forwardResponses()
	go e.forwardNotifications()
	go e.forwardLogs()

	return e
}

func getEndpointIP() string {
	addrs, _ := net.InterfaceAddrs()
	var ip string
	for _, addr := range addrs {
		network, ok := addr.(*net.IPNet)
		if ok && !network.IP.IsLoopback() && network.IP.To4() != nil {
			ip = network.IP.String()
		}
	}
	return ip
}

func (e *Endpoint) authenticateConn(msg messages.ClientRequest, id int) {
	switch id {
	case 0:
		status := "rejected"
		connected := messages.EndpointResponse{
			Connected: false,
			Status:    status,
		}
		res := messages.Message{
			ClientID: msg.ClientID,
			Type:     "connection",
			Content:  connected,
		}
		for _, client := range e.clients {
			if client.id == res.ClientID {
				client.responses <- res
				client.disconnect <- true
			}
		}
	default:
		status := "authenticated"
		connected := messages.EndpointResponse{
			Connected: true,
			Status:    status,
		}
		res := messages.Message{
			ClientID: msg.ClientID,
			Type:     "connection",
			Content:  connected,
		}
		for _, client := range e.clients {
			if client.id == res.ClientID {
				// Send response back...
				client.responses <- res
				// And fill client information from message TEMPORARY WE NEED TO REWRITE THIS
				client.UserID = id
			}
		}

	}
}

// Listen listens for connections and messages to broadcast
func (e *Endpoint) listen() {
	for {
		select {
		case conn := <-e.connect:
			e.Join(conn)
		case msg := <-e.requests:
			user.AuthReqs <- msg
			auth := <-user.AuthResp
			switch {
			// Case clients is connecting and wants confirmation
			case auth.Command[0] == "connect":
				e.authenticateConn(msg, auth.UserID)
				e.pushEndpointIP(auth)
				e.pushLastWorkspace(auth)
				// Client wants to modify its log level
			case strings.Join(auth.Command[:2], " ") == "log level":
				for _, client := range e.clients {
					if client.id == auth.ClientID {
						client.Logger.SetLevel(auth)
					}
				}
			default:
				// Else, authenticate the request anyway
				switch auth.UserID {
				case 0:
					connected := messages.EndpointResponse{
						Connected: false,
						Status:    "rejected",
					}
					res := messages.Message{
						ClientID: msg.ClientID,
						Type:     "connection",
						Content:  connected,
					}
					for _, client := range e.clients {
						if client.id == res.ClientID {
							client.responses <- res
							client.disconnect <- true
						}
					}
				default:
					// If authenticated, dispath resquest.
					e.dispatchRequest(auth)
				}
			}
		}
	}
}

// Connect passing connection to the server
func (e *Endpoint) Connect(conn net.Conn) {
	e.connect <- conn
}

// Join creates new client and starts listening for client messages
func (e *Endpoint) Join(conn net.Conn) {
	client := CreateClient(conn)
	e.clients = append(e.clients, client)
	go func() {
		for {
			e.requests <- <-client.requests
		}
	}()
}

// Remove disconnected client from list
func (e *Endpoint) Remove(i int) {
	e.clients = append(e.clients[:i], e.clients[i+1:]...)
}

func (e *Endpoint) forwardLogs() {
	for {
		// Remove disconnected clients
		for i, client := range e.clients {
			if client.status == 0 {
				e.Remove(i)
			}
		}
		// Prepare message when its a log event
		res := <-logging.ForwardLogs
		for _, client := range e.clients {
			if client.CurrentWorkspaceID == res.Data["workspaceId"] {
				client.Logger.Forward(res)
			}
		}

	}
}

func (e *Endpoint) forwardNotifications() {
	for {
		// Remove disconnected clients
		for i, client := range e.clients {
			if client.status == 0 {
				e.Remove(i)
			}
		}
		// Forward notification
		res := <-messages.Notifications
		switch {
		case res.Type == "workspace" && res.Action == "delete":
			for _, client := range e.clients {
				if client.CurrentWorkspaceID == res.WorkspaceID {
					msg := messages.Message{
						ClientID: client.id,
						Type:     "notification",
						Content:  res,
					}
					client.responses <- msg
				}
			}
		case res.Type == "module" && res.Action == "pop":
			for _, client := range e.clients {
				if client.CurrentWorkspaceID == res.WorkspaceID && client.id != res.NotConcerned {
					msg := messages.Message{
						ClientID: client.id,
						Type:     "notification",
						Content:  res,
					}
					client.responses <- msg
				}
			}
		case res.Type == "server":
			for _, client := range e.clients {
				if client.CurrentWorkspaceID == res.WorkspaceID && client.id != res.NotConcerned {
					msg := messages.Message{
						ClientID: client.id,
						Type:     "notification",
						Content:  res,
					}
					client.responses <- msg
				}
			}
		}
	}
}

func (e *Endpoint) forwardResponses() {
	for {
		// Remove disconnected clients
		for i, client := range e.clients {
			if client.status == 0 {
				e.Remove(i)
			}
		}
		res := <-messages.Responses
		// If its a workspace response, save the current workspace on the client-side, to server is aware.
		if res.Type == "workspace" {
			content := res.Content.(messages.WorkspaceResponse)
			if content.Workspace != "" {
				for _, client := range e.clients {
					if client.id == res.ClientID {
						client.CurrentWorkspaceID = content.WorkspaceID
						client.CurrentWorkspace = content.Workspace
					}
				}
			}
		}
		if res.Type == "server" {
			content := res.Content.(messages.ServerResponse)
			if content.ServerID.String() != "" {
				for _, client := range e.clients {
					if client.id == res.ClientID {
						client.CurrentServerID = content.ServerID
					}
				}
			}
		}
		// Forward response
		for _, client := range e.clients {
			if client.id == res.ClientID {
				client.responses <- res
			}
		}
	}
}

func (e *Endpoint) dispatchRequest(req messages.ClientRequest) {
	// 1. Check commands: most of them can be run in either context
	// 2. For context-sensitive commands, check context
	switch req.Command[0] {
	// Server
	case "server":
		messages.ForwardServerManager <- req
	// Log
	case "log":
		messages.ForwardLogger <- req
	// Stack
	case "stack":
		messages.ForwardModuleStack <- req
	// Workspace
	case "workspace":
		messages.ForwardWorkspace <- req
	// Module
	case "run", "show", "reload", "module":
		messages.ForwardModuleStack <- req
	// Compiler:
	case "list", "compile", "compiler":
		messages.ForwardCompiler <- req
	// Agent
	case "agent", "ls", "cd", "pwd", "cmd", "download",
		"execute-shellcode", "kill", "shell", "upload":
		messages.ForwardAgents <- req
	// For these commands we need to check context
	case "use":
		switch req.Context {
		case "main":
			messages.ForwardModuleStack <- req
		case "module", "agent":
			messages.ForwardModuleStack <- req
		case "compiler":
			messages.ForwardCompiler <- req
		}
	case "set":
		switch req.Context {
		case "main":
			messages.ForwardModuleStack <- req
		case "module":
			messages.ForwardModuleStack <- req
		case "agent":
			messages.ForwardAgents <- req
		case "compiler":
			messages.ForwardCompiler <- req
		}
	case "info":
		switch req.Context {
		case "main":
			messages.ForwardModuleStack <- req
		case "module":
			messages.ForwardModuleStack <- req
		case "agent":
			messages.ForwardAgents <- req
		case "compiler":
			messages.ForwardCompiler <- req
		}
	}
}

func (e *Endpoint) pushEndpointIP(request messages.ClientRequest) {
	// Craft notification
	ipRes := messages.Notification{
		Type:   "endpoint",
		Action: e.IPAddress,
	}
	ipMsg := messages.Message{
		ClientID: request.ClientID,
		Type:     "notification",
		Content:  ipRes,
	}

	for _, client := range e.clients {
		if client.id == request.ClientID {
			client.responses <- ipMsg
		}
	}
}

func (e *Endpoint) pushLastWorkspace(request messages.ClientRequest) {
	switch {
	case len(e.clients) == 1:
		// Forward request to workspace
		request.Command = []string{"workspace", "switch", "default"}
		messages.FromEndpoint <- request
		res := <-messages.ForwardEnpoint
		wsRes := messages.Notification{
			Type:        "workspace",
			Action:      "switch",
			WorkspaceID: res.WorkspaceID,
			Workspace:   res.Workspace,
		}
		wsMsg := messages.Message{
			ClientID: request.ClientID,
			Type:     "notification",
			Content:  wsRes,
		}
		messages.EndpointToServer <- request
		sRes := <-messages.ForwardServer
		// Craft server notification
		servRes := messages.Notification{
			Type:          "server",
			ServerID:      sRes.ServerID,
			ServerRunning: sRes.ServerRunning,
		}
		servMsg := messages.Message{
			ClientID: request.ClientID,
			Type:     "notification",
			Content:  servRes,
		}
		for _, client := range e.clients {
			if client.id == request.ClientID {
				// Fill client with workspace info and send notification
				client.CurrentWorkspaceID = wsRes.WorkspaceID
				client.CurrentWorkspace = wsRes.Workspace
				client.responses <- wsMsg
				// Fill client with server info and send notification
				client.CurrentServerID = servRes.ServerID
				client.serverRunning = servRes.ServerRunning
				client.responses <- servMsg
			}
		}
	case len(e.clients) == 2:
		if e.clients[0].UserID == request.UserID {
			// Craft workspace notification
			wsRes := messages.Notification{
				Type:        "workspace",
				Action:      "switch",
				WorkspaceID: e.clients[0].CurrentWorkspaceID,
				Workspace:   e.clients[0].CurrentWorkspace,
			}
			wsMsg := messages.Message{
				ClientID: request.ClientID,
				Type:     "notification",
				Content:  wsRes,
			}
			// Craft server notification
			servRes := messages.Notification{
				Type:          "server",
				ServerID:      e.clients[0].CurrentServerID,
				ServerRunning: e.clients[0].serverRunning,
			}
			servMsg := messages.Message{
				ClientID: request.ClientID,
				Type:     "notification",
				Content:  servRes,
			}
			for _, client := range e.clients {
				if client.id == request.ClientID {
					// Fill client with workspace info and send notification
					client.CurrentWorkspaceID = wsRes.WorkspaceID
					client.CurrentWorkspace = wsRes.Workspace
					client.responses <- wsMsg
					// Fill client with server info and send notification
					client.CurrentServerID = servRes.ServerID
					client.serverRunning = servRes.ServerRunning
					client.responses <- servMsg
				}
			}
		}
	case len(e.clients) > 2:
		var lastMatch int
		var lastMatchString string
		var lastMatchServer uuid.UUID
		var lastMatchServerRunning bool
		count := len(e.clients)
		for _, c := range e.clients {
			if c.UserID == request.UserID && count > 1 {
				lastMatch = c.CurrentWorkspaceID
				lastMatchString = c.CurrentWorkspace
				lastMatchServer = c.CurrentServerID
				lastMatchServerRunning = c.serverRunning
				count--
			}
		}
		// Craft workspace notification
		wsRes := messages.Notification{
			Type:        "workspace",
			Action:      "switch",
			WorkspaceID: lastMatch,
			Workspace:   lastMatchString,
		}
		wsMsg := messages.Message{
			ClientID: request.ClientID,
			Type:     "notification",
			Content:  wsRes,
		}
		// Craft server notification
		servRes := messages.Notification{
			Type:          "server",
			ServerID:      lastMatchServer,
			ServerRunning: lastMatchServerRunning,
		}
		servMsg := messages.Message{
			ClientID: request.ClientID,
			Type:     "notification",
			Content:  servRes,
		}
		for _, client := range e.clients {
			if client.id == request.ClientID {
				// Fill client with workspace info and send notification
				client.CurrentWorkspaceID = wsRes.WorkspaceID
				client.CurrentWorkspace = wsRes.Workspace
				client.responses <- wsMsg
				// Fill client with server info and send notification
				client.CurrentServerID = servRes.ServerID
				client.serverRunning = servRes.ServerRunning
				client.responses <- servMsg
			}
		}
	}
}
