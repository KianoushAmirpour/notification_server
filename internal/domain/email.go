package domain

type Mailer interface {
	SendVerificationEmail(email string, otp int) error
	SendNotificationEmail(email string) error
}
