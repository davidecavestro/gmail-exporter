package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"time"

	"github.com/davidecavestro/gmail-exporter/svc"
	"github.com/davidecavestro/gmail-exporter/ui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	"go.uber.org/ratelimit"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var rootCmd = &cobra.Command{
	Use:   "gmail-exporter [message labels...]",
	Short: "gmail-exporter is a mail message and attachments export utility",
	Long: `A simple CLI for exporting messages from Gmail filtering by labels.
	Valid credentials needs to be configured for relevant account. 
	Full docs available at https://github.com/davidecavestro/gmail-exporter`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		log.SetFormatter(&log.JSONFormatter{})
		ctx := context.Background()
		b, err := ioutil.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}
		client := svc.GetClient(config)

		srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("Unable to retrieve Gmail client: %v", err)
		}
		limitWindow := ratelimit.Per(1 * time.Second)

		messagesLimit := MessagesPerSec
		attachmentsLimit := AttachmentsPerSec

		// messageLimiter := ratelimit.New(messagesLimit, limitWindow)

		// initialize progress container, with custom width
		pui := ui.ProgressUI{Hide: HideProgressUI, BarContainer: mpb.New(mpb.WithWidth(ProgressBarWidth))}

		var pageLimit int64 = int64(PageLimit)
		var pageSize int64 = int64(PageSize)
		user := User
		outputFile := OutputFile
		labels := args

		msgs, totalMessages := svc.GetMessages(srv, messagesLimit, &pui, user, pageSize, pageLimit, labels...)

		var attachmentLimiter ratelimit.Limiter
		if attachmentsLimit != 0 {
			attachmentLimiter = ratelimit.New(attachmentsLimit, limitWindow)
		}
		saveMsgFiles := func(msg *gmail.Message) ([]*svc.LocalAttachment, error) {
			return svc.SaveAttachments(srv, attachmentLimiter, AttachmentsDir, AttachmentsSeed, user, msg)
		}

		var messageCount int64
		if pageSize > 0 && pageLimit > 0 {
			messageCount = (int64)(math.Min((float64)(totalMessages), (float64)(pageSize*pageLimit)))
		} else {
			messageCount = totalMessages
		}
		file := svc.ExportMessages(msgs, messageCount, &pui, saveMsgFiles)

		if err := file.SaveAs(outputFile); err != nil {
			log.Fatalf("Unable to save xls file: %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var User string
var PageLimit int64
var PageSize int64
var OutputFile string
var MessagesPerSec int
var AttachmentsPerSec int
var ProgressBarWidth int
var HideProgressUI bool
var AttachmentsDir string
var AttachmentsSeed *[]int32

func init() {
	rootCmd.Flags().StringVarP(&User, "user", "u", "me", "User ('me' is a shortcut to credentials account)")
	rootCmd.Flags().Int64VarP(&PageLimit, "pages-limit", "l", 0, "Max message pages fetched")
	rootCmd.Flags().Int64VarP(&PageSize, "page-size", "p", 100, "Messages per page")
	rootCmd.Flags().StringVarP(&OutputFile, "out-file", "f", "messages.xlsx", "Output file")

	rootCmd.Flags().IntVarP(&MessagesPerSec, "messages-per-sec", "m", 0, "Limit download of messages per second")
	rootCmd.Flags().IntVarP(&AttachmentsPerSec, "attachments-per-sec", "a", 0, "Limit download of attachments per second")

	rootCmd.Flags().BoolVarP(&HideProgressUI, "hide-progress", "n", false, "Hide progress bars")
	rootCmd.Flags().IntVarP(&ProgressBarWidth, "progressbar-width", "w", 64, "Progressbar width")
	// rootCmd.MarkFlagRequired("region")

	defaultAttachmentsSeed := []int32{2, 2}
	rootCmd.Flags().StringVarP(&AttachmentsDir, "attachments-dir", "d", "./", "Attachments output directory")
	AttachmentsSeed = &[]int32{}
	rootCmd.Flags().Int32SliceVarP(AttachmentsSeed, "attachments-seed", "s", defaultAttachmentsSeed, "Attachments output directory")

}
