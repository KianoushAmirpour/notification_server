package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type UserServiceResponse struct {
	Message string `json:"message"`
}

type UserServiceAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserService struct {
	Users               domain.UserRepository
	UserVerification    domain.UserVerificationRepository
	HashHandler         domain.HashRepository
	MailHandler         domain.Mailer
	OtpHandler          domain.OTPService
	OtpGenerator        domain.OTPGenerator
	JwtTokenHandler     domain.JwtTokenRepository
	RefreshTokenHandler domain.RefreshTokenRepository
	Logger              domain.LoggingRepository
}

func NewUserRegisterService(
	users domain.UserRepository,
	userVerification domain.UserVerificationRepository,
	hashHandler domain.HashRepository,
	mailhandler domain.Mailer,
	otphandler domain.OTPService,
	otpgenerator domain.OTPGenerator,
	jwttoken domain.JwtTokenRepository,
	reftoken domain.RefreshTokenRepository,
	logger domain.LoggingRepository,
) *UserService {
	return &UserService{
		Users:               users,
		UserVerification:    userVerification,
		HashHandler:         hashHandler,
		MailHandler:         mailhandler,
		OtpHandler:          otphandler,
		OtpGenerator:        otpgenerator,
		JwtTokenHandler:     jwttoken,
		RefreshTokenHandler: reftoken,
		Logger:              logger}
}

func (s *UserService) RegisterUser(ctx context.Context, req domain.RegisteredUser, reqid string, otpExpiration int) (*UserServiceResponse, error) {
	var err error

	start := time.Now()
	log := s.Logger.With("service", "register_user", "request_id", reqid, "user_email", req.Email)

	existing, err := s.Users.GetUserByEmail(ctx, req.Email)
	if existing != nil && err == nil {
		log.Warn("register_user_failed",
			"step", "check_user_existance",
			"reason", "email already exists")
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "email already exists", nil)
	}

	hashedPassword, err := s.HashHandler.Hash(req.Password, false)
	if err != nil {
		log.Error("register_user_failed",
			"step", "hash_password",
			"reason", err.Error())
		return nil, err
	}

	user := &domain.User{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    hashedPassword,
		Preferences: req.Preferences,
	}
	err = s.UserVerification.CreateUser(ctx, user)
	if err != nil {
		log.Error("register_user_failed", "step", "create_user_staging", "reason", err.Error())
		return nil, err
	}

	err = s.UserVerification.SaveVerificationData(ctx, reqid, req.Email)
	if err != nil {
		log.Error("register_user_failed", "step", "save_email_reqid", "reason", err.Error())
		return nil, err
	}

	otp, err := s.OtpGenerator.GenerateOTP()
	if err != nil {
		log.Error("register_user_failed_generate_otp", "reason", err.Error())
		return nil, err
	}

	hashedOtp, err := s.HashHandler.Hash(otp, false)
	if err != nil {
		log.Error("register_user_failed_hash_otp", "reason", err.Error())
		return nil, err
	}
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, time.Duration(5*time.Second))
	defer cancelFunc()

	otpErrChan := make(chan error, 1)
	emailErrChan := make(chan error, 1)

	go func() {
		otperr := s.OtpHandler.SaveOTP(timeoutCtx, req.Email, hashedOtp, otpExpiration)
		if err != nil {
			otpErrChan <- otperr
		}
		close(otpErrChan)
	}()

	go func() {
		<-time.After(1 * time.Second)
		emailerr := s.MailHandler.SendVerificationEmail(req.Email, otp)
		if emailerr != nil {
			emailErrChan <- emailerr
		}
		close(emailErrChan)
	}()

	err = <-otpErrChan
	if err != nil {
		log.Error("register_user_failed_save_otp", "reason", err.Error())
		return nil, err
	}

	err = <-emailErrChan
	if err != nil {
		log.Error("register_user_failed_send_otp_email", "reason", err.Error())
		return nil, err
	}

	log.Info("register_user_successfully", "duration_us", int(time.Since(start).Microseconds()))

	return &UserServiceResponse{Message: "The verification code was sent to your email. Please check your email"}, nil

}

func (s *UserService) VerifyUser(ctx context.Context, req domain.RegisterVerify, reqid string) (*UserServiceResponse, error) {

	start := time.Now()
	log := s.Logger.With("service", "verify_user", "request_id", reqid)

	// UserSentOtp, err := strconv.Atoi(req.SentOtpbyUser)
	// if err != nil {
	// 	log.Error("verify_user_failed_converting_otp_to_int", "reason", err.Error())
	// 	return nil, domain.ErrTypeConvertion
	// }

	email, err := s.UserVerification.RetrieveVerificationData(ctx, reqid)
	if err != nil {
		log.Error("verify_user_failed_get_email_reqid", "reason", err.Error())
		return nil, err
	}

	err = s.OtpHandler.VerifyOTP(ctx, email, req.SentOtpbyUser)
	if err != nil {
		log.Error("verify_user_failed_verify_otp", "reason", err.Error())
		return nil, err
	}
	err = s.UserVerification.MoveUserFromStaging(ctx, email)
	if err != nil {
		log.Error("verify_user_failed_move_user_from_stage", "reason", err.Error())
		return nil, err
	}

	err = s.UserVerification.DeleteUserFromStaging(ctx, email)
	if err != nil {
		log.Error("verify_user_failed_delete_user_from_stage", "reason", err.Error())
		return nil, err
	}

	err = s.UserVerification.DeleteUserVerificationData(ctx, email)
	if err != nil {
		log.Error("verify_user_failed_delete_email_after_verification", "reason", err.Error())
		return nil, err
	}

	log.Info("verify_user_successfully", "duration_us", int(time.Since(start).Microseconds()))
	return &UserServiceResponse{Message: "You are verified. Welcome 🥰. Lets explore🚀"}, nil

}

func (s *UserService) AuthenticateUser(ctx context.Context, req domain.LoginUser) (*UserServiceAuthResponse, error) {

	start := time.Now()
	log := s.Logger.With(slog.String("service", "authentication"), slog.String("email", req.Email))

	u, err := s.Users.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Error("authentication_failed_get_user_by_email", "reason", err.Error())
		return nil, err
	}

	err = s.HashHandler.VerifyHash([]byte(u.Password), req.Password, false)
	if err != nil {
		log.Error("authentication_failed_verify_user_password", "reason", err.Error())
		return nil, err
	}

	tokenPair, err := s.JwtTokenHandler.CreateJWTToken(u.ID, u.Email)
	if err != nil {
		log.Error("authentication_failed_create_jwt", "reason", err.Error())
		return nil, err
	}

	refreshTokenHash, err := s.HashHandler.Hash(tokenPair.RefreshToken, true)
	if err != nil {
		log.Error("authentication_failed_hash_refresh_token",
			"reason", err.Error())
		return nil, err
	}

	err = s.RefreshTokenHandler.StoreRefreshToken(ctx, u.ID, refreshTokenHash, time.Now().Add(time.Hour*24*7))
	if err != nil {
		log.Error("authentication_failed_persist_refresh_token", "reason", err.Error())
		return nil, err
	}

	log.Info("authentication_successfully", "duration_us", int(time.Since(start).Microseconds()))
	return &UserServiceAuthResponse{AccessToken: tokenPair.AccessToken, RefreshToken: tokenPair.RefreshToken}, nil

}

func (s *UserService) DeleteUser(ctx context.Context, req domain.User) (*UserServiceResponse, error) {
	start := time.Now()
	log := s.Logger.With("service", "delete_user", "email", req.Email, "user_id", req.ID)
	err := s.Users.DeleteUserByID(ctx, req.ID)
	if err != nil {
		log.Error("delete_by_id_failed", "reason", err.Error())
		return nil, err
	}
	log.Info("delete_user_completed", "duration_us", int(time.Since(start).Microseconds()))
	return &UserServiceResponse{Message: "User deleted successfully"}, nil
}

func (s *UserService) RefreshJwtToken(ctx context.Context, refreshToken string, JwtRefreshSecret string) (*UserServiceAuthResponse, error) {
	start := time.Now()
	log := s.Logger.With("service", "refresh_jwt")
	token, err := s.JwtTokenHandler.VerifyRefreshToken(refreshToken, []byte(JwtRefreshSecret))
	if err != nil {
		log.Error("verifying_refresh_jwt_failed", "reason", err.Error())
		return nil, err
	}
	user, err := s.Users.GetUserByID(ctx, token.UserID)
	if err != nil {
		log.Error("finding_user_by_id_failed", "reason", err.Error())
		return nil, err
	}
	dbReftoken, err := s.RefreshTokenHandler.RetrieveRefreshToken(ctx, user.ID)
	if err != nil {
		log.Error("finding_refresh_token_by_user_id_failed", "reason", err.Error())
		return nil, err
	}

	err = s.HashHandler.VerifyHash([]byte(dbReftoken.HashedToken), refreshToken, true)
	if err != nil {
		log.Error("verifying_hashed_token_failed", "reason", err.Error())
		return nil, err
	}

	if dbReftoken.ExpiredAt.Before(time.Now()) {
		log.Error("verifying_refrsh_token_failed", "reason", "refresh token expired")
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "refresh token expired", nil)
	}

	if dbReftoken.RevokedAt != nil {
		log.Error("verifying_refrsh_token_failed", "reason", "refresh token revoked")
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "refresh token revoked", nil)
	}

	tokenPair, err := s.JwtTokenHandler.CreateJWTToken(token.UserID, token.Email)
	if err != nil {
		log.Error("refresh_jwt_failed_create_jwt", "reason", err.Error())
		return nil, err
	}

	refreshTokenHash, err := s.HashHandler.Hash(tokenPair.RefreshToken, true)
	if err != nil {
		log.Error("hashing_refresh_token_failed",
			"reason", err.Error())
		return nil, err
	}

	err = s.RefreshTokenHandler.UpdateRefreshToken(ctx, user.ID, time.Now())
	if err != nil {
		log.Error("revoking_old_refresh_token_failed", "reason", err.Error())
		return nil, err
	}

	err = s.RefreshTokenHandler.StoreRefreshToken(ctx, user.ID, refreshTokenHash, time.Now().Add(time.Hour*24*7))
	if err != nil {
		log.Error("authentication_failed_persist_refresh_token", "reason", err.Error())
		return nil, err
	}

	log.Info("refresh_jwt_token_completed", "duration_us", int(time.Since(start).Microseconds()))
	return &UserServiceAuthResponse{AccessToken: tokenPair.AccessToken, RefreshToken: tokenPair.RefreshToken}, nil
}
