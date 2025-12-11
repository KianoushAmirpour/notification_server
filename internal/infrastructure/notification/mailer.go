package notification

import (
	"fmt"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	gomail "gopkg.in/mail.v2"
)

type Mailer struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	Logger    domain.LoggingRepository
}

func (m Mailer) SendVerificationEmail(email string, otp string) error {
	message := gomail.NewMessage()
	message.SetHeader("From", m.FromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", "verificatoin code")

	message.SetBody("text/plain", fmt.Sprintf("Thanks for your register. your code is %s", otp))

	dialer := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)

	if err := dialer.DialAndSend(message); err != nil {
		m.Logger.Error("verification_email_failed", "to", email, "reason", err.Error())
		return domain.NewDomainError(domain.ErrCodeExternal, "failed to send email", err)
	} else {
		// fmt.Println("HTML Email sent successfully with a plain-text alternative!")
		m.Logger.Info("verification_email_sent_successfully", "to", email)
		return nil
	}

}

func (m Mailer) SendNotificationEmail(email string) error {
	message := gomail.NewMessage()
	message.SetHeader("From", m.FromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", "Notification")

	message.SetBody("text/plain", "Your story is ready.🎉🎂")

	dialer := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)

	if err := dialer.DialAndSend(message); err != nil {
		m.Logger.Error("notification_email_failed", "to", email, "reason", err.Error())
		return domain.NewDomainError(domain.ErrCodeExternal, "failed to send email", err)
	} else {
		// fmt.Println("HTML Email sent successfully with a plain-text alternative!")
		m.Logger.Info("notification_email_sent_successfully", "to", email)
		return nil
	}

}
