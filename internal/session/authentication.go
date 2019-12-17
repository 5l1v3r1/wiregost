package session

// This file contains all objects and functions related to LOCAL client-side authentication,
// Because user credentials are also used for remote authentication with the server,
// The data structs in this file will be reused in other parts of the code.

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/evilsocket/islazy/fs"
	"github.com/evilsocket/islazy/tui"
	"golang.org/x/crypto/ssh/terminal"
)

type User struct {
	Name               string
	PasswordHashString string
	PasswordHash       [32]byte
	CredsFile          string
}

func NewUser() *User {
	user := &User{CredsFile: "~/.wiregost/client/.auth"}

	return user
}

func (user *User) LoadCreds() (err error) {

	// Check for personal directory, exit if not present.
	credsFile, _ := fs.Expand(user.CredsFile)
	if fs.Exists(credsFile) == false {
		fmt.Println(tui.Red(" ERROR: No ID and authentication information found."))
		fmt.Println(tui.Red("        Please run the ghost_setup.go script (in the " +
			"scripts directory), for initializing and configuring the client first"))
		os.Exit(1)
	} else {
		// Load authentication parameters
		fmt.Println(tui.Dim("Authentication parameters found."))
		credsFile, _ := fs.Expand(user.CredsFile)
		configBlob, _ := ioutil.ReadFile(credsFile)
		json.Unmarshal(configBlob, &user)
		fmt.Println(tui.Dim("Authentication file loaded."))
	}

	return err
}

// Local Authentication
func (user *User) Authenticate() error {
	fmt.Println()
	attempts := 0

	fmt.Printf(tui.Bold("Password: \n"))
	pass, _ := terminal.ReadPassword(int(syscall.Stdin))
	hash := sha256.Sum256(pass)

	for {
		// Success, authenticate
		if bytes.Equal(hash[:], user.PasswordHash[:]) {
			fmt.Println(tui.Green("Authentication success"))
			return nil
		}
		// Failure, 3 chances and then exit
		if !bytes.Equal(hash[:], user.PasswordHash[:]) {
			if attempts <= 3 {
				fmt.Println("Wrong password. Retry:")
				pass, _ = terminal.ReadPassword(int(syscall.Stdin))
				hash = sha256.Sum256(pass)
				attempts += 1
			}
			if attempts == 3 {
				fmt.Println(tui.Red("Authentication failure. Leaving WireGost"))
				os.Exit(1)
			}
		}
	}

	return nil
}
