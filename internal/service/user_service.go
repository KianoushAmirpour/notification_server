package service

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserRegisterService struct {
	Users          repository.UserRepository
	PasswordHasher repository.PasswordHasher
	Mailer         repository.Mailer
	Otp            repository.OTPService
}

var apiErr *domain.APIError

func NewUserRegisterService(
	users repository.UserRepository,
	passwordhasher repository.PasswordHasher,
	mailer repository.Mailer,
	otp repository.OTPService) *UserRegisterService {
	return &UserRegisterService{Users: users, PasswordHasher: passwordhasher, Mailer: mailer, Otp: otp}
}

func (s *UserRegisterService) RegisterUser(ctx context.Context, req domain.RegisteredUser, cfg *config.Config, reqid string) (*domain.RegisterResponse, *domain.APIError) {

	hashedPassword, err := s.PasswordHasher.HashPassword(req.Password)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	user := &domain.User{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    hashedPassword,
		Preferences: req.Preferences,
	}

	err = s.Users.CreateUserStaging(ctx, user)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	otp, err := adapters.GenerateOTP(cfg.OTPLength)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	err = s.Otp.SaveOTP(ctx, req.Email, otp, cfg.OTPExpiration)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	err = s.Mailer.SendVerification(req.Email, otp)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	err = s.Users.SaveEmailByReqID(ctx, reqid, req.Email)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	return &domain.RegisterResponse{Message: "The verification code was sent to your email. Please check your email"}, nil

}

func (s *UserRegisterService) VerifyUser(ctx context.Context, req domain.RegisterVerify, reqid string) (*domain.RegisterResponse, *domain.APIError) {

	UserSentOtp, err := strconv.Atoi(req.SentOtpbyUser)
	if err != nil {
		apiErr := domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	email, err := s.Users.GetEmailByReqID(ctx, reqid)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	err = s.Otp.VerifyOTP(ctx, email, UserSentOtp)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidOtp) {
			apiErr = domain.NewAPIError(err, http.StatusUnauthorized)
			return nil, apiErr
		} else if errors.Is(err, domain.ErrTooManyAttempts) {
			apiErr = domain.NewAPIError(err, http.StatusTooManyRequests)
			return nil, apiErr
		} else {
			apiErr = domain.NewAPIError(err, http.StatusUnauthorized)
			return nil, apiErr
		}

	}
	err = s.Users.MoveUserFromStaging(ctx, email)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	err = s.Users.DeleteUserFromStaging(ctx, email)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	err = s.Users.DeleteUserFromEmailVerification(ctx, email)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	return &domain.RegisterResponse{Message: "You are verified. Welcome 🥰. Lets explore🚀"}, nil

}

func (s *UserRegisterService) AuthenticateUser(ctx context.Context, req domain.LoginUser, cfg *config.Config) (*domain.RegisterResponse, *domain.APIError) {

	u, err := s.Users.GetByEmail(ctx, req.Email)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusNotFound)
		return nil, apiErr
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		apiErr = domain.NewAPIError(errors.New("invalid credentials"), http.StatusUnauthorized)
		return nil, apiErr
	} else if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusUnauthorized)
		return nil, apiErr
	}

	token, err := adapters.CreateJWTToken(u.ID, []byte(cfg.JwtSecret))
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
		return nil, apiErr
	}

	return &domain.RegisterResponse{Message: token}, nil

}

func (s *UserRegisterService) DeleteUser(ctx context.Context, req domain.User) (*domain.RegisterResponse, *domain.APIError) {
	err := s.Users.DeleteByID(ctx, req.ID)
	if err != nil {
		apiErr = domain.NewAPIError(err, http.StatusNotFound)
		return nil, apiErr
	}
	return &domain.RegisterResponse{Message: "User deleted succussfully"}, nil
}
