package svc

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidecavestro/gmail-exporter/ui"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type LocalAttachment struct {
	Filename string
}

func GetGmailSrv(CredsFile string, TokenFile string, BatchMode bool, NoBrowser bool, NoTokenSave bool) (*gmail.Service, error) {
	ctx := context.Background()
	b, err := ioutil.ReadFile(CredsFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
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
					log.Fatalf("Unable to retrieve attachment: %v", err)
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
								log.Debugf("File to write: %s - Type: %s - %#v\n", p.Filename, p.MimeType, p.Headers)
								err := os.MkdirAll(dirPath, os.ModePerm)
								if err != nil {
									log.Fatalf("Unable to prepare attachments dir: %v", err)
								}
								filename := filepath.Join(dirPath, p.Filename)
								err = ioutil.WriteFile(filename, decoded, 0644)
								if err != nil {
									log.Fatalf("Unable to save attachment: %v", err)
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

func GetMessages(srv *gmail.Service, messagesLimit int, pui *ui.ProgressUI, user string, pageSize int64, pageLimit int64, labelIds ...string) (chan *gmail.Message, int64) {
	ret := make(chan *gmail.Message, pageSize)

	var total int64 = 0
	for _, labelId := range labelIds {
		label, err := srv.Users.Labels.Get(user, labelId).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve label '%s': %v", labelId, err)
		}
		total += label.MessagesTotal
	}
	go func(ret chan *gmail.Message, srv *gmail.Service, user string, pageSize int64, pageLimit int64, labelIds ...string) error {
		var pageNum int64 = 0
		defer close(ret)
		caller := func() *gmail.UsersMessagesListCall {
			log.Debugf("Getting messages for page %d", pageNum)
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
				log.Fatalf("Unable to retrieve '%s' messages: %v", labelIds, err)
				return err
			}
			pageTotal := len(msgs.Messages)
			if pageTotal == 0 {
				log.Debugf("No messages found.")
				return nil
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
					log.Fatalf("Unable to retrieve %s message: %v", m.Id, err)
					return err
				}
				ret <- msg
			}
			if msgs.NextPageToken == "" {
				return nil
			}

			pageNum++
			if pageLimit > 0 && pageNum >= pageLimit {
				log.Debugf("Limit of '%d' message pages reached", pageNum)
				return nil
			}
			msgs, err = caller().PageToken(msgs.NextPageToken).Do()
		}
	}(ret, srv, user, pageSize, pageLimit, labelIds...)

	return ret, total
}
