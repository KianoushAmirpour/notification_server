package security

import (
	"errors"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type Hasher struct {
	Cost int
}

func (h Hasher) HashPassword(plainpassword string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainpassword), h.Cost)
	if err != nil {
		return "", domain.NewDomainError(domain.ErrCodeInternal, "failed to hash password", err)
	}
	return string(hashedPassword), nil
}

func (h Hasher) VerifyPassword(hashedpassword, password []byte) error {
	err := bcrypt.CompareHashAndPassword(hashedpassword, password)
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return domain.NewDomainError(domain.ErrCodeUnauthorized, "invalid credentials", err)
		}
		return domain.NewDomainError(domain.ErrCodeValidation, "invalid credentials", err)
	}
	return nil
}
