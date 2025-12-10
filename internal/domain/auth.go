package domain

type IdentityToken struct {
	UserID int
	Email  string
}

type JwtTokenRepository interface {
	CreateJWTToken(id int, email string) (string, error)
	VerifyJWTToken(tokenString string, secretKey []byte) (*IdentityToken, error)
}
