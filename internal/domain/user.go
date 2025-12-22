package domain

import (
	"context"
)

type User struct {
	ID          int
	FirstName   string
	LastName    string
	Email       string
	Password    string
	Preferences []string
}

type RegisteredUser struct {
	FirstName   string
	LastName    string
	Email       string
	Password    string
	Preferences []string
}

type Preferences struct {
	ID              int
	UserID          int
	UserPreferences []string
}

type RegisterVerify struct {
	SentOtpbyUser string
}

type LoginUser struct {
	Email    string
	Password string
}

type DeleteUser struct {
	ID int
}

type UserRepository interface {
	GetUserByID(ctx context.Context, id int) (*User, error)
	DeleteUserByID(ctx context.Context, id int) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserPreferencesByID(ctx context.Context, id int) (*Preferences, error)
}

type UserVerificationRepository interface {
	CreateUser(ctx context.Context, u *User) error
	SaveVerificationData(ctx context.Context, reqid, email string) error
	RetrieveVerificationData(ctx context.Context, reqid string) (string, error)
	PersistUserInfo(ctx context.Context, email string) (int, []string, error)
	PersistUserPreferenes(ctx context.Context, user_id int, preferences []string) error
	DeleteUserFromStaging(ctx context.Context, email string) error
	DeleteUserVerificationData(ctx context.Context, email string) error
}
