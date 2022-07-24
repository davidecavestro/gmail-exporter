package svc

import (
	"context"
	_ "embed"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidecavestro/gmail-exporter/logger"
	"github.com/davidecavestro/gmail-exporter/ui"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type LocalAttachment struct {
	Filename string
}

//go:embed credentials.json
var Creds []byte

func GetGmailSrv(TokenFile string, BatchMode bool, NoBrowser bool, NoTokenSave bool) (*gmail.Service, error) {
	ctx := context.Background()

	config, err := google.ConfigFromJSON(Creds, gmail.GmailReadonlyScope)
	if err != nil {
		logger.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := GetClient(config, TokenFile, BatchMode, NoBrowser, NoTokenSave)

	return gmail.NewService(ctx, option.WithHTTPClient(client))
}

func SaveAttachments(srv *gmail.Service, rateLimiter ratelimit.Limiter, AttachmentsDir string, AttachmentsSeed *[]int32, user string, message *gmail.Message) ([]*LocalAttachment, error) {

	var ret []*LocalAttachment
	if len(message.Payload.Parts) > 0 {

		paths := []string{AttachmentsDir}
		var last int32
		last = 0
		for _, p := range *AttachmentsSeed {
			paths = append(paths, message.Id[last:last+p])
			last = p
		}
		for _, p := range message.Payload.Parts {
			// fmt.Printf("Filename: %s\n", p.Filename)
			// fmt.Printf("Attachment ID: %s\n", p.Body.AttachmentId)

			for attachmentId := p.Body.AttachmentId; attachmentId != ""; {
				if rateLimiter != nil {
					rateLimiter.Take()
				}
				attach, err := srv.Users.Messages.Attachments.Get(user, message.Id, attachmentId).Do()
				if err != nil {
					logger.Fatalf("Unable to retrieve attachment: %v", err)
				}
				attachmentId = attach.AttachmentId
				if attachmentId != "" {
					continue
				}
				if attach.Data == "" {
					break
				}
				decoded, _ := base64.URLEncoding.DecodeString(attach.Data)

				// p.Filename = mailReceivedDate + "_" + p.Filename
				if p.Filename != "" {
					for _, ph := range p.Headers {
						if ph.Name == "Content-Disposition" {
							if !strings.HasPrefix(ph.Value, "inline;") {
								dirPath := filepath.Join(paths...)
								err := os.MkdirAll(dirPath, os.ModePerm)
								if err != nil {
									logger.Fatal("Unable to prepare attachments dir: ", zap.Error(err))
								}
								filename := filepath.Join(dirPath, p.Filename)
								err = ioutil.WriteFile(filename, decoded, 0644)
								if err != nil {
									logger.Fatal("Unable to save attachment: ", zap.Error(err))
								}
								ret = append(ret, &LocalAttachment{Filename: filename})
							}
						}
					}
				}
			}
		}
	}
	return ret, nil
}

func ListLabels(srv *gmail.Service, user string) ([]*gmail.Label, error) {
	if labels, err := srv.Users.Labels.List(user).Do(); err != nil {
		return nil, err
	} else {
		return labels.Labels, err
	}
}

func GetLabelsByIdOrName(srv *gmail.Service, user string, refs ...string) ([]*gmail.Label, error) {
	if labels, err := srv.Users.Labels.List(user).Do(); err != nil {
		return nil, err
	} else {
		ret := make([]*gmail.Label, 0)
		for _, ref := range refs {
			for _, label := range labels.Labels {
				if label.Id == ref || label.Name == ref {
					// get label details
					if label, err := srv.Users.Labels.Get(user, label.Id).Do(); err != nil {
						return nil, err
					} else {
						ret = append(ret, label)
						break
					}
				}
			}
		}
		return ret, err
	}
}

func GetMessages(srv *gmail.Service, messagesLimit int, pui *ui.ProgressUI, user string, pageSize int64, pageLimit int64, labelRefs ...string) (chan *gmail.Message, int64) {
	ret := make(chan *gmail.Message, pageSize)

	var total int64 = 0
	labels, err := GetLabelsByIdOrName(srv, user, labelRefs...)
	if err != nil {
		logger.Fatalf("Unable to retrieve labels '%s': %v", labelRefs, err)
	}
	if len(labels) == 0 {
		logger.Info("No labels found matching", labelRefs)
		os.Exit(10)
	}
	labelIds := make([]string, 0)
	for _, label := range labels {
		if label != nil {
			total += label.MessagesTotal
			labelIds = append(labelIds, label.Id)
		}
	}

	go func(ret chan *gmail.Message, srv *gmail.Service, user string, pageSize int64, pageLimit int64, labelIds ...string) {
		var pageNum int64 = 0
		defer close(ret)
		caller := func() *gmail.UsersMessagesListCall {
			logger.Debugf("Getting messages for page %d", pageNum)
			return srv.Users.Messages.List(user).MaxResults(pageSize).LabelIds(labelIds...)
		}

		limitWindow := ratelimit.Per(1 * time.Second)
		var rateLimiter ratelimit.Limiter
		if messagesLimit != 0 {
			rateLimiter = ratelimit.New(messagesLimit, limitWindow)
		}

		msgs, err := caller().Do()

		for {
			if err != nil {
				logger.Fatalf("Unable to retrieve '%s' messages: %v", labelIds, err)
				return
			}
			pageTotal := len(msgs.Messages)
			if pageTotal == 0 {
				logger.Debugf("No messages found.")
				return
			}

			pui.GmailNewPage(int64(pageTotal), pageNum)
			// pui.GmailPageTotal(int64(pageTotal))
			for _, m := range msgs.Messages {
				if rateLimiter != nil {
					rateLimiter.Take()
				}
				pui.GmailIncrement()

				msg, err := srv.Users.Messages.Get(user, m.Id).Format("full").Do()
				if err != nil {
					logger.Fatalf("Unable to retrieve %s message: %v", m.Id, err)
					return
				}
				ret <- msg
			}
			if msgs.NextPageToken == "" {
				return
			}

			pageNum++
			if pageLimit > 0 && pageNum >= pageLimit {
				logger.Debugf("Limit of '%d' message pages reached", pageNum)
				return
			}
			msgs, err = caller().PageToken(msgs.NextPageToken).Do()
		}
	}(ret, srv, user, pageSize, pageLimit, labelIds...)

	return ret, total
}
