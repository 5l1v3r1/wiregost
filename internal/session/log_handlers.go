package session

import "fmt"

func (s *Session) LogLevel(cmd []string) {
	// Send(cmd)
	log := <-s.logReqs
	fmt.Println(log)
	// Handle change of state here
}

func (s *Session) LogShow(cmd []string) {
	// Send(cmd)
	log := <-s.logReqs
	fmt.Println(log)
	// Handle printing the logs here
}

// Handle all log messages coming from the server
func (s *Session) LogListen() {
	go func() {
		for {
			msg := <-s.logReqs
			fmt.Println(msg)
		}
	}()
}
