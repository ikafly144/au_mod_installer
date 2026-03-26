package main

import (
	"github.com/ikafly144/au_mod_installer/cmd/mus-mgr/internal/musmgr"
	"log"
	"os"
)

func main() {
	app := musmgr.NewApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
