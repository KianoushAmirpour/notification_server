package domain

type Mailer interface {
	SendVerificationEmail(email string, otp string) error
	SendNotificationEmail(email string) error
}
