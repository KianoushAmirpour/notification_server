package adapters

import "golang.org/x/crypto/bcrypt"

type Hasher struct {
	Cost int
}

func (h Hasher) HashPassword(plainpassword string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainpassword), h.Cost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
