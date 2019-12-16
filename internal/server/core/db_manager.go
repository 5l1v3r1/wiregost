package core

// This file contains all the code used for Database management.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/evilsocket/islazy/fs"
	"github.com/evilsocket/islazy/tui"
	"github.com/go-pg/pg"
)

type DBManager struct {
	User     string
	Database string
	Password string

	DB *pg.DB
}

func NewDBManager() *DBManager {
	man := &DBManager{}

	// Load credentials
	dbFile := "~/.wiregost/server/database.conf"
	path, _ := fs.Expand(dbFile)
	if !fs.Exists(path) {
		fmt.Println(tui.Red("Database file not found: check for issues, or run the configuration script again"))
		os.Exit(1)
	} else {
		// Load Creds
		configBlob, _ := ioutil.ReadFile(path)
		json.Unmarshal(configBlob, &man)
		fmt.Println(tui.Dim("Database credentials loaded."))
		// And connect to DB
		man.DB = pg.Connect(&pg.Options{
			User:     man.User,
			Password: man.Password,
			Database: man.Database,
		})
	}

	return man
}
