package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gmail-exporter export [msg labels...]",
	Short: "gmail-exporter is a mail message and attachments export utility",
	Long: `A simple CLI for exporting messages from Gmail, filtering by labels.

	Valid credentials needs to be configured for relevant account. 
	Full docs available at https://github.com/davidecavestro/gmail-exporter`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var User string
var BatchMode bool
var NoBrowser bool
var NoTokenSave bool
var TokenFile string

func init() {
	rootCmd.PersistentFlags().StringVarP(&User, "user", "u", "me", "User - 'me' is a shortcut to credentials account")
	rootCmd.PersistentFlags().StringVarP(&TokenFile, "token-file", "t", "token.json", "File containing the auth token")
	rootCmd.PersistentFlags().BoolVarP(&BatchMode, "batch", "b", false, "Batch mode - not acquiring new auth tokens nor showing progress bars")
	rootCmd.PersistentFlags().BoolVarP(&NoBrowser, "no-browser", "w", false, "Don't open the web browser if authentication needed")
	rootCmd.PersistentFlags().BoolVarP(&NoTokenSave, "no-token-save", "s", false, "Don't save obtained token")

}
