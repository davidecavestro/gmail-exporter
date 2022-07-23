package cmd

import (
	"math"
	"time"

	"github.com/davidecavestro/gmail-exporter/logger"
	"github.com/davidecavestro/gmail-exporter/svc"
	"github.com/davidecavestro/gmail-exporter/ui"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	"go.uber.org/ratelimit"
	"google.golang.org/api/gmail/v1"
)

var PageLimit int64
var PageSize int64
var OutputFile string
var MessagesPerSec int
var AttachmentsPerSec int
var ProgressBarWidth int
var NoProgressBar bool
var NoAttachments bool
var AttachmentsDir string
var AttachmentsSeed *[]int32

func init() {
	exportCmd.Flags().Int64VarP(&PageLimit, "pages-limit", "l", 0, "Max message pages fetched (default 0, so unlimited)")
	exportCmd.Flags().Int64VarP(&PageSize, "page-size", "p", 25, "Messages per page")
	exportCmd.Flags().StringVarP(&OutputFile, "out-file", "f", "messages.xlsx", "Output file")

	exportCmd.Flags().IntVarP(&MessagesPerSec, "messages-per-sec", "m", 0, "Limit download of messages per second (default 0, so unlimited)")
	exportCmd.Flags().IntVarP(&AttachmentsPerSec, "attachments-per-sec", "e", 0, "Limit download of attachments per second (default 0, so unlimited)")

	exportCmd.Flags().BoolVarP(&NoProgressBar, "no-progressbar", "n", false, "Hide progress bars")
	exportCmd.Flags().IntVarP(&ProgressBarWidth, "progressbar-width", "i", 64, "Progressbar width")

	exportCmd.Flags().BoolVarP(&NoAttachments, "no-attachments", "a", false, "Don't export attachments")
	defaultAttachmentsSeed := []int32{2, 2}
	exportCmd.Flags().StringVarP(&AttachmentsDir, "attachments-dir", "d", "attachments", "Attachments output directory")
	AttachmentsSeed = &[]int32{}
	exportCmd.Flags().Int32SliceVarP(AttachmentsSeed, "attachments-seed", "x", defaultAttachmentsSeed, "Attachments subfolder naming strategy")

	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:              "export [msg labels...]",
	Short:            "Export mail messages",
	Long:             `Export mail messages, optionally filtered by specified labels.`,
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		srv, err := svc.GetGmailSrv(TokenFile, BatchMode, NoBrowser, NoTokenSave)
		if err != nil {
			logger.Fatalf("Unable to retrieve Gmail client: %v", err)
		}

		user := User
		limitWindow := ratelimit.Per(1 * time.Second)

		messagesLimit := MessagesPerSec
		attachmentsLimit := AttachmentsPerSec

		// messageLimiter := ratelimit.New(messagesLimit, limitWindow)

		// initialize progress container, with custom width
		pui := ui.ProgressUI{Hide: NoProgressBar || BatchMode, BarContainer: mpb.New(mpb.WithWidth(ProgressBarWidth))}

		var pageLimit int64 = int64(PageLimit)
		var pageSize int64 = int64(PageSize)
		outputFile := OutputFile
		labels := args

		msgs, totalMessages := svc.GetMessages(srv, messagesLimit, &pui, user, pageSize, pageLimit, labels...)

		var attachmentLimiter ratelimit.Limiter
		if attachmentsLimit != 0 {
			attachmentLimiter = ratelimit.New(attachmentsLimit, limitWindow)
		}
		var saveMsgFiles svc.SaveMsgAttachments = nil
		if !NoAttachments {
			saveMsgFiles = func(msg *gmail.Message) ([]*svc.LocalAttachment, error) {
				return svc.SaveAttachments(srv, attachmentLimiter, AttachmentsDir, AttachmentsSeed, user, msg)
			}
		}

		var messageCount int64
		if pageSize > 0 && pageLimit > 0 {
			messageCount = (int64)(math.Min((float64)(totalMessages), (float64)(pageSize*pageLimit)))
		} else {
			messageCount = totalMessages
		}
		file := svc.ExportMessages(msgs, messageCount, &pui, saveMsgFiles)

		if err := file.SaveAs(outputFile); err != nil {
			logger.Fatalf("Unable to save xls file: %v", err)
		}
	},
}
