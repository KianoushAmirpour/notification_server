package usecase

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type UserServiceResponse struct {
	Message string `json:"message"`
}

type UserService struct {
	Users            domain.UserRepository
	UserVerification domain.UserVerificationRepository
	PasswordHandler  domain.Password
	MailHandler      domain.Mailer
	OtpHandler       domain.OTPService
	OtpGenerator     domain.OTPGenerator
	JwtTokenHandler  domain.JwtTokenRepository
	Logger           domain.LoggingRepository
}

func NewUserRegisterService(
	users domain.UserRepository,
	userVerification domain.UserVerificationRepository,
	passwordhandler domain.Password,
	mailhandler domain.Mailer,
	otphandler domain.OTPService,
	otpgenerator domain.OTPGenerator,
	jwttoken domain.JwtTokenRepository,
	logger domain.LoggingRepository,
) *UserService {
	return &UserService{
		Users:            users,
		UserVerification: userVerification,
		PasswordHandler:  passwordhandler,
		MailHandler:      mailhandler,
		OtpHandler:       otphandler,
		OtpGenerator:     otpgenerator,
		JwtTokenHandler:  jwttoken,
		Logger:           logger}
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

	hashedPassword, err := s.PasswordHandler.HashPassword(req.Password)
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

	timeoutCtx, cancelFunc := context.WithTimeout(ctx, time.Duration(5*time.Second))
	defer cancelFunc()

	otpErrChan := make(chan error, 1)
	emailErrChan := make(chan error, 1)

	go func() {
		otperr := s.OtpHandler.SaveOTP(timeoutCtx, req.Email, otp, otpExpiration)
		if err != nil {
			otpErrChan <- otperr
		}
		close(otpErrChan)
	}()

	go func() {
		<-time.After(5 * time.Second)
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

	UserSentOtp, err := strconv.Atoi(req.SentOtpbyUser)
	if err != nil {
		log.Error("verify_user_failed_converting_otp_to_int", "reason", err.Error())
		return nil, domain.ErrTypeConvertion
	}

	email, err := s.UserVerification.RetrieveVerificationData(ctx, reqid)
	if err != nil {
		log.Error("verify_user_failed_get_email_reqid", "reason", err.Error())
		return nil, err
	}

	err = s.OtpHandler.VerifyOTP(ctx, email, UserSentOtp)
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

func (s *UserService) AuthenticateUser(ctx context.Context, req domain.LoginUser, jwtIss, jwtSecret string) (*UserServiceResponse, error) {

	start := time.Now()
	log := s.Logger.With(slog.String("service", "authentication"), slog.String("email", req.Email))

	u, err := s.Users.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Error("authentication_failed_get_user_by_email", "reason", err.Error())
		return nil, err
	}

	err = s.PasswordHandler.VerifyPassword([]byte(u.Password), []byte(req.Password))
	if err != nil {
		log.Error("authentication_failed_verify_user_password", "reason", err.Error())
		return nil, err
	}

	token, err := s.JwtTokenHandler.CreateJWTToken(u.ID, u.Email)
	if err != nil {
		log.Error("authentication_failed_create_jwt", "reason", err.Error())
		return nil, err
	}

	log.Info("authentication_successfully", "duration_us", int(time.Since(start).Microseconds()))
	return &UserServiceResponse{Message: token}, nil

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
