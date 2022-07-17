package cmd

import (
	"fmt"
	"os"

	"github.com/davidecavestro/gmail-exporter/svc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(labelsCmd)
}

var labelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "List available labels",
	Long:  `List all available labels, so that you can use them to filter exported messages.`,
	Run: func(cmd *cobra.Command, args []string) {
		srv, err := svc.GetGmailSrv(CredsFile, TokenFile, BatchMode, NoBrowser, NoTokenSave)
		if err != nil {
			log.Fatalf("Unable to retrieve Gmail client: %v", err)
		}

		user := User

		if labels, err := svc.ListLabels(srv, user); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else {
			for _, label := range labels {
				fmt.Fprintln(os.Stdout, label.Name)
			}
			os.Exit(0)
		}
	},
}
