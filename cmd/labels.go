package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/davidecavestro/gmail-exporter/logger"
	"github.com/davidecavestro/gmail-exporter/svc"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(labelsCmd)
}

var labelsCmd = &cobra.Command{
	Use:   "labels [filter regex]",
	Short: "List available labels",
	Long:  `List all available labels - optionally matching a filter - so that you can use them to filter exported messages.`,
	Run: func(cmd *cobra.Command, args []string) {
		srv, err := svc.GetGmailSrv(TokenFile, BatchMode, NoBrowser, NoTokenSave)
		if err != nil {
			logger.Fatalf("Unable to retrieve Gmail client: %v", err)
		}

		user := User

		if labels, err := svc.ListLabels(srv, user); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else {
			patterns := []string{`.*`}

			if len(args) > 0 {
				patterns = args
			}
			filters := make([]*regexp.Regexp, 0)

			for _, pattern := range patterns {
				filters = append(filters, regexp.MustCompile(pattern))
			}

			for _, label := range labels {
				for _, filter := range filters {
					match := filter.Match([]byte(label.Name))
					if match {
						fmt.Printf("%s\n", label.Name)
						break
					}
				}
			}
			os.Exit(0)
		}
	},
}
