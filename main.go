package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/davidecavestro/gmail-exporter/cmd"
)

func main() {
	log.SetOutput(os.Stdout)
	cmd.Execute()
}
