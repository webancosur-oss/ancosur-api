package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

type SMTPClient struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

func NewSMTPClient(
	host, port, username, password, from, fromName string,
) *SMTPClient {
	return &SMTPClient{
		Host:     strings.TrimSpace(host),
		Port:     strings.TrimSpace(port),
		Username: strings.TrimSpace(username),
		Password: password,
		From:     strings.TrimSpace(from),
		FromName: strings.TrimSpace(fromName),
	}
}

func (s *SMTPClient) Validate() error {
	switch {
	case s.Host == "":
		return fmt.Errorf("SMTP_HOST no está configurado")
	case s.Port == "":
		return fmt.Errorf("SMTP_PORT no está configurado")
	case s.Username == "":
		return fmt.Errorf("SMTP_USERNAME no está configurado")
	case s.Password == "":
		return fmt.Errorf("SMTP_PASSWORD no está configurado")
	case s.From == "":
		return fmt.Errorf("EMAIL_FROM no está configurado")
	default:
		return nil
	}
}

func (s *SMTPClient) Send(
	to, replyTo, subject, htmlBody, textBody string,
	attachments []Attachment,
) (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}

	to = strings.TrimSpace(to)
	if to == "" {
		return "", fmt.Errorf("el destinatario está vacío")
	}

	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	fromHeader := s.From
	if s.FromName != "" {
		fromHeader = fmt.Sprintf(
			"%s <%s>",
			mime.QEncoding.Encode("utf-8", s.FromName),
			s.From,
		)
	}

	messageID := buildMessageID(s.From)

	var raw []byte
	var err error

	if len(attachments) == 0 {
		raw = buildSimpleMessage(
			fromHeader,
			to,
			replyTo,
			subject,
			messageID,
			htmlBody,
			textBody,
		)
	} else {
		raw, err = buildMultipartMessage(
			fromHeader,
			to,
			replyTo,
			subject,
			messageID,
			htmlBody,
			textBody,
			attachments,
		)
		if err != nil {
			return "", err
		}
	}

	if err := smtp.SendMail(
		s.Host+":"+s.Port,
		auth,
		s.From,
		[]string{to},
		raw,
	); err != nil {
		return "", err
	}

	return messageID, nil
}

func buildSimpleMessage(
	fromHeader, to, replyTo, subject, messageID, htmlBody, textBody string,
) []byte {
	body := htmlBody
	contentType := "text/html; charset=UTF-8"

	if strings.TrimSpace(body) == "" {
		body = textBody
		contentType = "text/plain; charset=UTF-8"
	}

	headers := []string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"Message-ID: " + messageID,
		"MIME-Version: 1.0",
		"Content-Type: " + contentType,
		"Content-Transfer-Encoding: 8bit",
	}

	if strings.TrimSpace(replyTo) != "" {
		headers = append(headers, "Reply-To: "+replyTo)
	}

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body)
}

func buildMultipartMessage(
	fromHeader, to, replyTo, subject, messageID, htmlBody, textBody string,
	attachments []Attachment,
) ([]byte, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	headers := []string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"Message-ID: " + messageID,
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=" + writer.Boundary(),
	}

	if strings.TrimSpace(replyTo) != "" {
		headers = append(headers, "Reply-To: "+replyTo)
	}

	buffer.WriteString(strings.Join(headers, "\r\n"))
	buffer.WriteString("\r\n\r\n")

	bodyHeaders := textproto.MIMEHeader{}
	body := htmlBody
	bodyHeaders.Set("Content-Type", "text/html; charset=UTF-8")
	if strings.TrimSpace(body) == "" {
		body = textBody
		bodyHeaders.Set("Content-Type", "text/plain; charset=UTF-8")
	}
	bodyHeaders.Set("Content-Transfer-Encoding", "8bit")

	bodyPart, err := writer.CreatePart(bodyHeaders)
	if err != nil {
		return nil, err
	}

	if _, err := bodyPart.Write([]byte(body)); err != nil {
		return nil, err
	}

	for _, attachment := range attachments {
		if strings.TrimSpace(attachment.Filename) == "" {
			continue
		}

		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		filename := mime.QEncoding.Encode("utf-8", attachment.Filename)

		partHeaders := textproto.MIMEHeader{}
		partHeaders.Set("Content-Type",
			contentType+`; name="`+filename+`"`)
		partHeaders.Set("Content-Disposition",
			`attachment; filename="`+filename+`"`)
		partHeaders.Set("Content-Transfer-Encoding", "base64")

		part, err := writer.CreatePart(partHeaders)
		if err != nil {
			return nil, err
		}

		encoded := base64.StdEncoding.EncodeToString(attachment.Content)
		for len(encoded) > 76 {
			if _, err := part.Write([]byte(encoded[:76] + "\r\n")); err != nil {
				return nil, err
			}
			encoded = encoded[76:]
		}
		if encoded != "" {
			if _, err := part.Write([]byte(encoded + "\r\n")); err != nil {
				return nil, err
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func buildMessageID(from string) string {
	host := "localhost"
	if at := strings.LastIndex(from, "@"); at >= 0 && at+1 < len(from) {
		host = from[at+1:]
	}
	return fmt.Sprintf(
		"<%d.%s@%s>",
		time.Now().UnixNano(),
		strings.ReplaceAll(strings.ReplaceAll(from, " ", ""), "@", "."),
		host,
	)
}
