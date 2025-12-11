package security

import (
	"crypto/sha256"
	"errors"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type Hasher struct {
	Cost int
}

func (h Hasher) Hash(plaintext string, preHash bool) (string, error) {
	data := []byte(plaintext)
	if preHash {
		sum := sha256.Sum256(data)
		data = sum[:]
	}
	hashedtext, err := bcrypt.GenerateFromPassword(data, h.Cost)
	if err != nil {
		return "", domain.NewDomainError(domain.ErrCodeInternal, "failed to hash", err)
	}
	return string(hashedtext), nil
}

func (h Hasher) VerifyHash(hashedtext []byte, plaintext string, preHash bool) error {
	data := []byte(plaintext)
	if preHash {
		sum := sha256.Sum256(data)
		data = sum[:]
	}
	err := bcrypt.CompareHashAndPassword(hashedtext, data)
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return domain.NewDomainError(domain.ErrCodeUnauthorized, "invalid credentials", err)
		}
		return domain.NewDomainError(domain.ErrCodeValidation, "invalid credentials", err)
	}
	return nil
}
