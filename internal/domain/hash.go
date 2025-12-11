package domain

type HashRepository interface {
	Hash(plaintext string, preHash bool) (string, error)
	VerifyHash(hashedtext []byte, plaintext string, preHash bool) error
}
