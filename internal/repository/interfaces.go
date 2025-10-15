package repository

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	DeleteByID(ctx context.Context, id int) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateUserStaging(ctx context.Context, u *domain.User) error
	MoveUserFromStaging(ctx context.Context, email string) error
	DeleteUserFromStaging(ctx context.Context, email string) error
	SaveEmailByReqID(ctx context.Context, reqid, email string) error
	GetEmailByReqID(ctx context.Context, reqid string) (string, error)
	DeleteUserFromEmailVerification(ctx context.Context, email string) error
}

type PasswordHasher interface {
	HashPassword(plainpassword string) (string, error)
}

type OTPService interface {
	SaveOTP(ctx context.Context, email string, otp, expiration int) error
	VerifyOTP(ctx context.Context, email string, sentopt int) error
}

type Mailer interface {
	SendVerification(email string, otp int) error
}

type ImageGeneration interface {
	GenerateImage(ctx context.Context)
}
