package adapters

import (
	"crypto/rand"
	"math/big"
)

func GenerateOTP(n int) (int, error) {

	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	otp, err := rand.Int(rand.Reader, max)

	if err != nil {
		return 0, err
	}

	return int(otp.Int64()), err
}
