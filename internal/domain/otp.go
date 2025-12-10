package domain

import "context"

type OTPService interface {
	SaveOTP(ctx context.Context, email string, otp, expiration int) error
	VerifyOTP(ctx context.Context, email string, sentopt int) error
}

type OTPGenerator interface {
	GenerateOTP() (int, error)
}
