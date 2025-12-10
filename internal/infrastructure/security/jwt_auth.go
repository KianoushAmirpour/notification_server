package security

import (
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type JwtAuth struct {
	Secret []byte
	Issuer string
}

type CustomClaims struct {
	UserID int
	Email  string
	jwt.RegisteredClaims
}

func (j JwtAuth) CreateJWTToken(id int, email string) (string, error) {

	accessTokenClaims := CustomClaims{
		UserID: id,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.Issuer,
			Subject:   "access-token",
			ExpiresAt: jwt.NewNumericDate((time.Now().Add(time.Hour))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)

	tokenString, err := token.SignedString(j.Secret)
	if err != nil {
		return "", domain.NewDomainError(domain.ErrCodeInternal, "failed to create jwt token", err)
	}

	return tokenString, nil

}

func (j JwtAuth) VerifyJWTToken(tokenString string, secretKey []byte) (*domain.IdentityToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return nil, domain.ErrInvalidJWTToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !token.Valid || !ok {
		return nil, domain.ErrInvalidJWTToken
	}

	return &domain.IdentityToken{UserID: claims.UserID, Email: claims.Email}, nil
}
