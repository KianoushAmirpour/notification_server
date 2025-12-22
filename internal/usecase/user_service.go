package usecase

import (
	"context"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/observability"
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

func (s *UserService) RegisterUser(ctx context.Context, req domain.RegisteredUser, otpExpiration int) (*UserServiceResponse, error) {
	reqID := observability.GetRequestID(ctx)
	log := s.Logger.With("service.name", "register", "http.request.id", reqID, "event.category", []string{"iam"})

	log.Info("user registration started", "event.type", []string{"start"})

	existing, err := s.Users.GetUserByEmail(ctx, req.Email)
	if existing != nil && err == nil {
		log.Warn(
			"user already exists",
			"event.action", "check_existing_user",
			"event.outcome", "failed",
			"event.type", []string{"error", "end"},
			"reason", "email already exists")
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "email already exists", nil)
	}

	hashedPassword, err := s.HashHandler.Hash(req.Password, false)
	if err != nil {
		log.Error(
			"failed to hash user password",
			"event.action", "hash_password",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
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
		log.Error(
			"failed to save user",
			"event.action", "create_user",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.UserVerification.SaveVerificationData(ctx, reqID, req.Email)
	if err != nil {
		log.Error(
			"failed to save verification data",
			"event.action", "save_verification_data",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	otp, err := s.OtpGenerator.GenerateOTP()
	if err != nil {
		log.Error(
			"failed to generate otp code",
			"event.action", "generate_otp",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	hashedOtp, err := s.HashHandler.Hash(otp, false)
	if err != nil {
		log.Error("failed to hash otp code",
			"event.action", "hash_otp",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
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

	otperr := <-otpErrChan
	if otperr != nil {
		log.Error(
			"failed to save hashed otp code",
			"event.action", "save_otp",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	emailerr := <-emailErrChan
	if emailerr != nil {
		log.Error(
			"failed to send verification email to user",
			"event.action", "send_verification_email",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	log.Info(
		"user successfully registered",
		"event.type", []string{"end", "creation"},
		"event.outcome", "success")

	return &UserServiceResponse{Message: "The verification code was sent to your email. Please check your email"}, nil

}

func (s *UserService) VerifyUser(ctx context.Context, req domain.RegisterVerify, regsterReqID string) (*UserServiceResponse, error) {

	reqID := observability.GetRequestID(ctx)
	log := s.Logger.With("service.name", "verification", "http.request.id", reqID, "event.category", []string{"iam"})
	log.Info("user verification started", "event.type", []string{"start"})

	email, err := s.UserVerification.RetrieveVerificationData(ctx, regsterReqID)
	if err != nil {
		log.Error(
			"failed to retrieve verification data",
			"event.action", "retrieve_verification_data",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.OtpHandler.VerifyOTP(ctx, email, req.SentOtpbyUser)
	if err != nil {
		log.Error(
			"failed to verify otp code",
			"event.action", "verify_otp",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}
	userID, preferences, err := s.UserVerification.PersistUserInfo(ctx, email)
	if err != nil {
		log.Error(
			"failed to save user information",
			"event.action", "save_user_info",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.UserVerification.PersistUserPreferenes(ctx, userID, preferences)
	if err != nil {
		log.Error(
			"failed to save user preferences",
			"event.action", "save_user_preferences",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.UserVerification.DeleteUserFromStaging(ctx, email)
	if err != nil {
		log.Error(
			"failed to delete user information",
			"event.action", "delete_user_from_staging",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	log.Info(
		"user successfully verified",
		"user.id", userID,
		"event.type", []string{"end", "allowed"},
		"event.outcome", "success")

	return &UserServiceResponse{Message: "You are verified. Welcome 🥰. Lets explore🚀"}, nil

}

func (s *UserService) AuthenticateUser(ctx context.Context, req domain.LoginUser) (*UserServiceAuthResponse, error) {

	reqID := observability.GetRequestID(ctx)
	log := s.Logger.With("service.name", "login", "http.request.id", reqID, "event.category", []string{"authentication"})
	log.Info("user authentication started", "event.type", []string{"start"})

	u, err := s.Users.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Error(
			"failed to find user by email",
			"event.action", "get_user_by_email",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.HashHandler.VerifyHash([]byte(u.Password), req.Password, false)
	if err != nil {
		log.Error(
			"failed to verify user password",
			"user.id", u.ID,
			"event.action", "verify_user-password",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	tokenPair, err := s.JwtTokenHandler.CreateJWTToken(u.ID, u.Email)
	if err != nil {
		log.Error(
			"failed to create access and refresh token",
			"user.id", u.ID,
			"event.action", "create_jwt_token",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	refreshTokenHash, err := s.HashHandler.Hash(tokenPair.RefreshToken, true)
	if err != nil {
		log.Error(
			"failed to hash refresh token",
			"user.id", u.ID,
			"event.action", "hash_refresh_token",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.RefreshTokenHandler.StoreRefreshToken(ctx, u.ID, refreshTokenHash, time.Now().Add(time.Hour*24*7))
	if err != nil {
		log.Error(
			"failed to save refresh token",
			"user.id", u.ID,
			"event.action", "save_refresh_token",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	log.Info(
		"user successfully logged in",
		"user.id", u.ID,
		"event.type", []string{"end", "allowed"},
		"event.outcome", "success")

	return &UserServiceAuthResponse{AccessToken: tokenPair.AccessToken, RefreshToken: tokenPair.RefreshToken}, nil

}

func (s *UserService) DeleteUser(ctx context.Context, req domain.DeleteUser) (*UserServiceResponse, error) {
	reqID := observability.GetRequestID(ctx)
	log := s.Logger.With("service.name", "deletion", "http.request.id", reqID, "user.id", req.ID, "event.category", []string{"iam"})
	log.Info("user deletion started", "event.type", []string{"start"})

	err := s.Users.DeleteUserByID(ctx, req.ID)
	if err != nil {
		log.Error(
			"failed to delete user by id",
			"event.action", "delete_user_by_id",
			"event.type", []string{"error", "denied"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}
	log.Info(
		"user deleted successfully",
		"event.type", []string{"end", "deletion"},
		"event.outcome", "success")

	return &UserServiceResponse{Message: "User deleted successfully"}, nil
}

func (s *UserService) RefreshJwtToken(ctx context.Context, refreshToken string, JwtRefreshSecret string) (*UserServiceAuthResponse, error) {
	reqID := observability.GetRequestID(ctx)
	log := s.Logger.With("service.name", "jwt-refresh", "http.request.id", reqID, "event.category", []string{"authentication"})
	log.Info("refreshing jwt token started", "event.type", []string{"start"})

	token, err := s.JwtTokenHandler.VerifyRefreshToken(refreshToken, []byte(JwtRefreshSecret))
	if err != nil {
		log.Error(
			"failed to verify jwt refresh token",
			"event.action", "verify_refresh_token",
			"event.type", []string{"error", "denied"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}
	user, err := s.Users.GetUserByID(ctx, token.UserID)
	if err != nil {
		log.Error("failed to find user by id",
			"event.action", "get_user_by_id",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}
	dbReftoken, err := s.RefreshTokenHandler.RetrieveRefreshToken(ctx, user.ID)
	if err != nil {
		log.Error(
			"failed to retrieve refresh token",
			"event.action", "retrive_jwt_refresh_token",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.HashHandler.VerifyHash([]byte(dbReftoken.HashedToken), refreshToken, true)
	if err != nil {
		log.Error(
			"failed to verify hashed refresh token",
			"event.action", "verify_jwt_hashed_refresh_token",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	if dbReftoken.ExpiredAt.Before(time.Now()) {
		log.Error(
			"refresh token expired",
			"event.action", "refresh_token_expiration",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", "refresh token expired")
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "refresh token expired", nil)
	}

	if dbReftoken.RevokedAt != nil {
		log.Error(
			"refresh token already revoked",
			"event.action", "refresh_token_revoke",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", "refresh token revoked")
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "refresh token revoked", nil)
	}

	tokenPair, err := s.JwtTokenHandler.CreateJWTToken(token.UserID, token.Email)
	if err != nil {
		log.Error(
			"failed to create access and refresh token",
			"event.action", "create_jwt_token",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	refreshTokenHash, err := s.HashHandler.Hash(tokenPair.RefreshToken, true)
	if err != nil {
		log.Error(
			"failed to hash refresh token",
			"event.action", "hash_refresh_token",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.RefreshTokenHandler.UpdateRefreshToken(ctx, user.ID, time.Now())
	if err != nil {
		log.Error(
			"failed to update refresh token",
			"event.action", "update_refresh_token",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	err = s.RefreshTokenHandler.StoreRefreshToken(ctx, user.ID, refreshTokenHash, time.Now().Add(time.Hour*24*7))
	if err != nil {
		log.Error(
			"failed to save new refresh token",
			"event.action", "save_new_refresh_token",
			"user.id", user.ID,
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	log.Info(
		"update refresh token successfuly",
		"user.id", user.ID,
		"event.type", []string{"end", "creation"},
		"event.outcome", "success")
	return &UserServiceAuthResponse{AccessToken: tokenPair.AccessToken, RefreshToken: tokenPair.RefreshToken}, nil
}
