package mail

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

const timeout = 30 * time.Second

type Config struct {
	Host string
	Port int
	User string
	Password string
	From string
}

func (c Config) Enabled() bool { return c.Host != "" && c.From != "" }

type Message struct {
	To string
	Subject string
	Body string
}

func headerSafe(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r", " ", "\n", " ").Replace(s)), " ")
}

func (c Config) build(m Message) ([]byte, error) {
	from, err := mail.ParseAddress(c.From)
	if err != nil {
		return nil, fmt.Errorf("SMTP_FROM no es una dirección válida: %w", err)
	}
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return nil, fmt.Errorf("destinatario inválido: %w", err)
	}
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	domain := from.Address[strings.LastIndex(from.Address, "@")+1:]
	var b bytes.Buffer
	for _, h := range [][2]string{
		{"From", from.String()},
		{"To", to.String()},
		{"Subject", mime.QEncoding.Encode("utf-8", headerSafe(m.Subject))},
		{"Date", time.Now().Format(time.RFC1123Z)},
		{"Message-ID", "<" + hex.EncodeToString(id) + "@" + domain + ">"},
		{"MIME-Version", "1.0"},
		{"Content-Type", "text/plain; charset=utf-8"},
		{"Content-Transfer-Encoding", "quoted-printable"},
		{"Auto-Submitted", "auto-generated"},
	} {
		b.WriteString(h[0] + ": " + h[1] + "\r\n")
	}
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c Config) dial() (*smtp.Client, error) {
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	dialer := &net.Dialer{Timeout: timeout}
	var conn net.Conn
	var err error
	if c.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: c.Host})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(timeout))
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if c.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: c.Host}); err != nil {
				client.Close()
				return nil, err
			}
		}
	}
	if c.User != "" {
		if err := client.Auth(smtp.PlainAuth("", c.User, c.Password, c.Host)); err != nil {
			client.Close()
			return nil, err
		}
	}
	return client, nil
}

func (c Config) Send(m Message) error {
	data, err := c.build(m)
	if err != nil {
		return err
	}
	from, _ := mail.ParseAddress(c.From)
	to, _ := mail.ParseAddress(m.To)
	client, err := c.dial()
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func Permanent(err error) bool {
	var proto *textproto.Error
	return errors.As(err, &proto) && proto.Code >= 500
}
