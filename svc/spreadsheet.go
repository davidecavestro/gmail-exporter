package svc

import (
	"encoding/base64"
	"strings"

	"github.com/davidecavestro/gmail-exporter/logger"
	"github.com/davidecavestro/gmail-exporter/ui"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/gmail/v1"
)

type SaveMsgAttachments func(*gmail.Message) ([]*LocalAttachment, error)
type SaveEml func(*gmail.Message) (string, error)

func ExportMessages(
	msgs chan *gmail.Message, total int64, pui *ui.ProgressUI,
	saveMsgAttachments SaveMsgAttachments,
	saveEml SaveEml, NoHtmlBody bool, NoTextBody bool) *excelize.File {

	pui.SpreadsheetTotal(total)

	file := excelize.NewFile()
	streamWriter, err := file.NewStreamWriter("Sheet1")
	if err != nil {
		logger.Fatalf("Unable to prepare stream writer: %v", err)
	}
	styleID, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "#777777"}})
	if err != nil {
		logger.Fatalf("Unable to prepare header style: %v", err)
	}
	if err := streamWriter.SetRow("A1", []interface{}{
		excelize.Cell{StyleID: styleID, Value: "FROM"},
		excelize.Cell{StyleID: styleID, Value: "TO"},
		excelize.Cell{StyleID: styleID, Value: "SIZE"},
		excelize.Cell{StyleID: styleID, Value: "DATE"},
		excelize.Cell{StyleID: styleID, Value: "DATE INTERNAL"},
		excelize.Cell{StyleID: styleID, Value: "THREAD"},
		excelize.Cell{StyleID: styleID, Value: "SUBJECT"},
		excelize.Cell{StyleID: styleID, Value: "SNIPPET"},
		excelize.Cell{StyleID: styleID, Value: "TEXT BODY"},
		excelize.Cell{StyleID: styleID, Value: "HTML BODY"},
		excelize.Cell{StyleID: styleID, Value: "EML"},
		excelize.Cell{StyleID: styleID, Value: "ATTACHMENT LIST"},
		excelize.Cell{StyleID: styleID, Value: "ATTACHMENT1"},
		excelize.Cell{StyleID: styleID, Value: "ATTACHMENT2"},
		excelize.Cell{StyleID: styleID, Value: "ATTACHMENT3"},
		excelize.Cell{StyleID: styleID, Value: "ATTACHMENT4"}},
		excelize.RowOpts{Height: 25, Hidden: false}); err != nil {
		logger.Fatalf("Cannot write headers to prepare stream, writer: %v", err)
	}
	err = file.AddTable("Sheet1", "A1", "O1", `{
		"table_name": "table",
		"table_style": "TableStyleLight1",
		"show_first_column": true,
		"show_last_column": true,
		"show_row_stripes": true,
		"show_column_stripes": true
	}`)
	if err != nil {
		logger.Fatalf("Unable to decorate xls table: %v", err)
	}
	// err = file.AutoFilter("Sheet1", "A1", "H1", "")
	err = file.SetPanes("Sheet1", `{
		"freeze": true,
		"split": false,
		"x_split": 0,
		"y_split": 1,
		"top_left_cell": "B2",
		"active_pane": "topRight",
		"panes": [
		{
			"sqref": "A2",
			"active_cell": "A2",
			"pane": "topRight"
		}]
	}`)
	if err != nil {
		logger.Fatalf("Unable to decorate xls table: %v", err)
	}

	rowID := 2
	for msg := range msgs {
		var attachments []*LocalAttachment = nil
		var err error = nil
		if saveMsgAttachments != nil {
			attachments, err = saveMsgAttachments(msg)
		}
		if err != nil {
			logger.Fatalf("Cannot save attachments: %v", err)
		}

		var emlFile string = ""
		if saveEml != nil {
			emlFile, err = saveEml(msg)
			if err != nil {
				logger.Fatalf("Cannot save message: %v", err)
			}
		}
		// wg.Done()
		row := make([]interface{}, 50)
		// go func(msg gmail.Message) {

		for _, h := range msg.Payload.Headers {
			// fmt.Printf("- %s -", h)
			if h.Name == "From" {
				row[0] = h.Value
			}
			if h.Name == "To" {
				row[1] = h.Value
			}
			if h.Name == "Date" {
				row[3] = h.Value
			}
			if h.Name == "Subject" {
				row[6] = h.Value
			}
		}
		row[2] = msg.SizeEstimate
		row[4] = msg.InternalDate
		row[5] = msg.ThreadId
		row[7] = msg.Snippet
		var getBody func(parts []*gmail.MessagePart) (string, string, error)
		getBody = func(parts []*gmail.MessagePart) (string, string, error) {
			text := ""
			html := ""
			for _, part := range parts {
				if part.MimeType == "text/plain" {
					data, _ := base64.StdEncoding.DecodeString(part.Body.Data)
					text = concat("\n", text, string(data))
				} else if part.MimeType == "text/html" {
					data, _ := base64.StdEncoding.DecodeString(part.Body.Data)
					html = concat("\n", html, string(data))
				} else if part.Parts != nil {
					innerText, innerHtml, err := getBody(part.Parts)
					if err != nil {
						logger.Fatalf("Unable to process inner part: %v", err)
					}
					if innerText != "" {
						text = concat("\n", text, innerText)
					}
					if innerHtml != "" {
						html = concat("\n", html, innerHtml)
					}
				}
			}
			return text, html, nil
		}

		textBody, htmlBody, err := getBody(msg.Payload.Parts)
		if err != nil {
			logger.Fatalf("Unable to get message body for msg %s: %v", msg.Id, err)
		}

		if !NoTextBody {
			row[8] = textBody
		}
		if !NoHtmlBody {
			row[9] = htmlBody
		}
		row[10] = emlFile

		attachmentCsv := []string{}
		col := 12
		for attPos, attachment := range attachments {
			if attachment != nil {
				attachmentCsv = append(attachmentCsv, attachment.Filename)

				if attPos < 4 {
					row[col] = attachment.Filename
					col++
				}
			}
		}
		row[11] = strings.Join(attachmentCsv, ",")
		// }(*msg)

		cell, _ := excelize.CoordinatesToCellName(1, rowID)
		rowID++
		if err := streamWriter.SetRow(cell, row); err != nil {
			logger.Fatalf("Unable to set xls row: %v", err)
		}
		pui.SpreadsheetIncrement()
	}

	if err := streamWriter.Flush(); err != nil {
		logger.Fatalf("Unable to save xls file: %v", err)
	}

	return file
}
