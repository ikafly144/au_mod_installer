package main

import (
	"context"
	"log"
	"os"

	"github.com/ikafly144/au_mod_installer/cmd/mus-mgr/internal/musmgr"
)

func main() {
	app := musmgr.NewApp()

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
