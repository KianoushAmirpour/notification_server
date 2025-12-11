package domain

import (
	"context"
	"time"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type IdentityToken struct {
	UserID int
	Email  string
}

type RefreshToken struct {
	ID          string
	UserID      int
	HashedToken string
	ExpiredAt   time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

type JwtTokenRepository interface {
	CreateJWTToken(id int, email string) (*TokenPair, error)
	VerifyJWTToken(tokenString string, secretKey []byte) (*IdentityToken, error)
	VerifyRefreshToken(tokenString string, secretKey []byte) (*IdentityToken, error)
}

type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	RetrieveRefreshToken(ctx context.Context, userID int) (*RefreshToken, error)
	UpdateRefreshToken(ctx context.Context, userID int, revokedAt time.Time) error
}
