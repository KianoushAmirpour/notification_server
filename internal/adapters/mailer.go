package adapters

import (
	"fmt"

	gomail "gopkg.in/mail.v2"
)

type Mailer struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
}

func (m Mailer) SendVerification(email string, otp int) error {
	message := gomail.NewMessage()
	message.SetHeader("From", m.FromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", "verificatoin code")

	message.SetBody("text/plain", fmt.Sprintf("Thanks for your register. your code is %d", otp))

	dialer := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)

	if err := dialer.DialAndSend(message); err != nil {
		return err
	} else {
		fmt.Println("HTML Email sent successfully with a plain-text alternative!")
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
		return err
	} else {
		fmt.Println("HTML Email sent successfully with a plain-text alternative!")
		return nil
	}

}
