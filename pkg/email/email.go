package email

import (
	"fmt"
	"net/smtp"
)

type Sender interface {
	SendVerificationCode(to, code string) error
}

type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTPSender(host string, port int, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) SendVerificationCode(to, code string) error {
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	msg := fmt.Sprintf(
		"From: Zync <%s>\r\nTo: %s\r\nSubject: Код подтверждения Zync\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nВаш код подтверждения: %s\r\n\r\nКод действителен 10 минут.",
		s.from, to, code,
	)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
