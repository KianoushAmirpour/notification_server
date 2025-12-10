package security

import (
	"crypto/rand"
	"math/big"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type Otpgen struct {
	OTPLength int
}

func (o Otpgen) GenerateOTP() (int, error) {

	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(o.OTPLength)), nil)
	otp, err := rand.Int(rand.Reader, max)

	if err != nil {
		return 0, domain.NewDomainError(domain.ErrCodeInternal, "failed to generate opt code", err)
	}

	return int(otp.Int64()), nil
}
