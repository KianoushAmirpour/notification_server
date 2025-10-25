package domain

import "github.com/golang-jwt/jwt/v5"

type User struct {
	ID          int      `json:"id" validate:"gte=0"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Preferences []string `json:"preferences"`
}

type RegisteredUser struct {
	FirstName   string   `json:"first_name" validate:"required"`
	LastName    string   `json:"last_name" validate:"required"`
	Email       string   `json:"email" validate:"required,email"`
	Password    string   `json:"password" validate:"required,min=8,max=64,passwod_strength"`
	Preferences []string `json:"preferences" validate:"required,unique,user_preferences_check"`
}

type RegisterResponse struct {
	Message string `json:"message"`
}

type RegisterVerify struct {
	SentOtpbyUser string `json:"otp" validate:"required"`
}

type LoginUser struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type CustomClaims struct {
	UserID int
	Email  string
	jwt.RegisteredClaims
}
