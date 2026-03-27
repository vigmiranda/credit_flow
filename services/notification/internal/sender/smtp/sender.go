package smtp

import (
	"fmt"
	"net/smtp"
	"strings"

	"creditflow/services/notification/internal/domain"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Sender struct {
	address string
	auth    smtp.Auth
	from    string
}

func New(config Config) *Sender {
	var auth smtp.Auth
	if strings.TrimSpace(config.Username) != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	return &Sender{
		address: fmt.Sprintf("%s:%s", config.Host, config.Port),
		auth:    auth,
		from:    config.From,
	}
}

func (s *Sender) Send(notification domain.Notification) error {
	body := strings.Join([]string{
		fmt.Sprintf("To: %s", notification.Recipient),
		fmt.Sprintf("Subject: Credit Flow - %s", notification.TriggerStatus),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		notification.Message,
	}, "\r\n")

	return smtp.SendMail(s.address, s.auth, s.from, []string{notification.Recipient}, []byte(body))
}
