package repository

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetUserByID(ctx context.Context, id int) (*domain.User, error)
	DeleteByID(ctx context.Context, id int) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateUserStaging(ctx context.Context, tx pgx.Tx, u *domain.User) error
	MoveUserFromStaging(ctx context.Context, tx pgx.Tx, email string) error
	DeleteUserFromStaging(ctx context.Context, tx pgx.Tx, email string) error
	SaveEmailByReqID(ctx context.Context, tx pgx.Tx, reqid, email string) error
	GetEmailByReqID(ctx context.Context, tx pgx.Tx, reqid string) (string, error)
	DeleteUserFromEmailVerification(ctx context.Context, tx pgx.Tx, email string) error
	SaveStoryMetaData(ctx context.Context, i *domain.Story) error
	Upload(ctx context.Context, story *domain.UploadStory) error
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

type Job interface {
	Run(ctx context.Context) (context.Context, error)
}

type WorkerPool interface {
	Submit(job Job)
	Start(resultchan chan string)
	Stop()
}

type StoryRepository interface {
	Upload(ctx context.Context, story *domain.UploadStory) error
}
