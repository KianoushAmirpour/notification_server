package adapters

import (
	"fmt"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

func CreateJWTToken(id int, email string, secret []byte, issuer string) (string, error) {

	accessTokenClaims := domain.CustomClaims{
		UserID: id,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "access-token",
			ExpiresAt: jwt.NewNumericDate((time.Now().Add(time.Hour))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil

}

func VerifyJWTToken(tokenString string, secretKey []byte) (*domain.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &domain.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*domain.CustomClaims)
	if !token.Valid || !ok {
		return nil, jwt.ErrInvalidKey
	}

	return claims, nil
}
