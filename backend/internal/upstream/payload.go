package upstream

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"os"

	"smoi/internal/database"
)

func writeJSON(writer io.Writer, value any) error {
	return json.NewEncoder(writer).Encode(value)
}

func writeLiteral(writer io.Writer, value string) error {
	_, err := io.WriteString(writer, value)
	return err
}

func (c *Client) writePayload(
	writer io.Writer, model string, messages []database.Message,
) error {
	if err := writeLiteral(writer, `{"model":`); err != nil {
		return err
	}
	if err := writeJSON(writer, model); err != nil {
		return err
	}
	if err := writeLiteral(writer, `,"stream":true,"messages":[`); err != nil {
		return err
	}

	for index, message := range messages {
		if index > 0 {
			if err := writeLiteral(writer, ","); err != nil {
				return err
			}
		}
		if err := c.writeMessage(writer, message); err != nil {
			return err
		}
	}

	return writeLiteral(writer, "]}")
}

func (c *Client) writeMessage(writer io.Writer, message database.Message) error {
	if err := writeLiteral(writer, `{"role":`); err != nil {
		return err
	}
	if err := writeJSON(writer, message.Role); err != nil {
		return err
	}
	if err := writeLiteral(writer, `,"content":`); err != nil {
		return err
	}

	if len(message.Attachments) == 0 {
		if err := writeJSON(writer, message.Content); err != nil {
			return err
		}
	} else {
		if err := c.writeParts(writer, message); err != nil {
			return err
		}
	}

	return writeLiteral(writer, "}")
}

func (c *Client) writeParts(writer io.Writer, message database.Message) error {
	if err := writeLiteral(writer, `[{"type":"text","text":`); err != nil {
		return err
	}
	if err := writeJSON(writer, message.Content); err != nil {
		return err
	}
	if err := writeLiteral(writer, "}"); err != nil {
		return err
	}

	for _, attachment := range message.Attachments {
		if err := writeLiteral(writer, ","); err != nil {
			return err
		}
		if err := c.writeFilePart(writer, attachment); err != nil {
			return err
		}
	}

	return writeLiteral(writer, "]")
}

func (c *Client) writeFilePart(writer io.Writer, attachment database.Attachment) error {
	absolute, err := c.Store.Path(attachment.StorageReference)
	if err != nil {
		return err
	}

	file, err := os.Open(absolute)
	if err != nil {
		return err
	}
	defer file.Close()

	isImage := attachment.MIMEType == "image/png" ||
		attachment.MIMEType == "image/jpeg" ||
		attachment.MIMEType == "image/gif" ||
		attachment.MIMEType == "image/webp"

	if isImage {
		if err := writeLiteral(writer, `{"type":"image_url","image_url":{"url":"data:`+
			attachment.MIMEType+`;base64,`); err != nil {
			return err
		}
	} else {
		if err := writeLiteral(writer, `{"type":"file","file":{"filename":`); err != nil {
			return err
		}
		if err := writeJSON(writer, attachment.Filename); err != nil {
			return err
		}
		if err := writeLiteral(writer, `,"file_data":"data:`+
			attachment.MIMEType+`;base64,`); err != nil {
			return err
		}
	}

	encoder := base64.NewEncoder(base64.StdEncoding, writer)
	_, copyErr := io.Copy(encoder, file)
	closeErr := encoder.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return writeLiteral(writer, `"}}`)
}
