package security

import (
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type JwtAuth struct {
	AccessSecret  []byte
	RefreshSecret []byte
	Issuer        string
}

type CustomClaims struct {
	UserID int
	Email  string
	jwt.RegisteredClaims
}

func (j JwtAuth) CreateJWTToken(id int, email string) (*domain.TokenPair, error) {

	accessTokenClaims := CustomClaims{
		UserID: id,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.Issuer,
			Subject:   "access-token",
			ExpiresAt: jwt.NewNumericDate((time.Now().Add(time.Minute * 15))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accesstoken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)

	accesstokenString, err := accesstoken.SignedString(j.AccessSecret)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to create jwt access token", err)
	}

	RefreshTokenClaims := CustomClaims{
		UserID: id,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.Issuer,
			Subject:   "refresh-token",
			ExpiresAt: jwt.NewNumericDate((time.Now().Add(time.Hour * 24 * 7))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	reftoken := jwt.NewWithClaims(jwt.SigningMethodHS256, RefreshTokenClaims)

	reftokenString, err := reftoken.SignedString(j.RefreshSecret)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to create jwt refresh token", err)
	}

	return &domain.TokenPair{AccessToken: accesstokenString, RefreshToken: reftokenString}, nil

}

func (j JwtAuth) VerifyJWTToken(tokenString string, secretKey []byte) (*domain.IdentityToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidJWTMethod
		}
		return secretKey, nil
	})

	if err != nil || token == nil {
		return nil, domain.ErrInvalidJWTToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !token.Valid || !ok || claims.Subject != "access-token" {
		return nil, domain.ErrInvalidJWTToken
	}

	return &domain.IdentityToken{UserID: claims.UserID, Email: claims.Email}, nil
}

func (j JwtAuth) VerifyRefreshToken(tokenString string, secretKey []byte) (*domain.IdentityToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidJWTMethod
		}
		return secretKey, nil
	})

	if err != nil || token == nil {
		return nil, domain.ErrInvalidJWTToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !token.Valid || !ok || claims.Subject != "refresh-token" {
		return nil, domain.ErrInvalidJWTToken
	}

	return &domain.IdentityToken{UserID: claims.UserID, Email: claims.Email}, nil
}
