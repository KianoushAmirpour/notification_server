package domain

type Password interface {
	HashPassword(plainpassword string) (string, error)
	VerifyPassword(hashedpassword, password []byte) error
}
