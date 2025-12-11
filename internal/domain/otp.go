package domain

import "context"

type OTPService interface {
	SaveOTP(ctx context.Context, email string, otp string, expiration int) error
	VerifyOTP(ctx context.Context, email string, sentopt string) error
}

type OTPGenerator interface {
	GenerateOTP() (string, error)
}
