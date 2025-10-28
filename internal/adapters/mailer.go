package adapters

import (
	"fmt"
	"log/slog"

	gomail "gopkg.in/mail.v2"
)

type Mailer struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	Logger    *slog.Logger
}

func (m Mailer) SendVerification(email string, otp int) error {
	message := gomail.NewMessage()
	message.SetHeader("From", m.FromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", "verificatoin code")

	message.SetBody("text/plain", fmt.Sprintf("Thanks for your register. your code is %d", otp))

	dialer := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)

	if err := dialer.DialAndSend(message); err != nil {
		m.Logger.Error("verification_email_failed", slog.String("to", email), slog.String("reason", err.Error()))
		return err
	} else {
		// fmt.Println("HTML Email sent successfully with a plain-text alternative!")
		m.Logger.Info("verification_email_sent_successfully", slog.String("to", email))
		return nil
	}

}

func (m Mailer) SendNotification(email string) error {
	message := gomail.NewMessage()
	message.SetHeader("From", m.FromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", "Notification")

	message.SetBody("text/plain", "Your story is ready.🎉🎂")

	dialer := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)

	if err := dialer.DialAndSend(message); err != nil {
		m.Logger.Error("notification_email_failed", slog.String("to", email), slog.String("reason", err.Error()))
		return err
	} else {
		// fmt.Println("HTML Email sent successfully with a plain-text alternative!")
		m.Logger.Info("notification_email_sent_successfully", slog.String("to", email))
		return nil
	}

}
