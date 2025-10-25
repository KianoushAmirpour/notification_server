package service

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserRegisterService struct {
	Users          repository.UserRepository
	PasswordHasher repository.PasswordHasher
	Mailer         repository.Mailer
	Otp            repository.OTPService
	DbPool         *pgxpool.Pool
}

func NewUserRegisterService(
	users repository.UserRepository,
	passwordhasher repository.PasswordHasher,
	mailer repository.Mailer,
	otp repository.OTPService,
	dbpool *pgxpool.Pool) *UserRegisterService {
	return &UserRegisterService{Users: users, PasswordHasher: passwordhasher, Mailer: mailer, Otp: otp, DbPool: dbpool}
}

func (s *UserRegisterService) RegisterUser(ctx context.Context, req domain.RegisteredUser, cfg *config.Config, reqid string, logger *slog.Logger) (*domain.RegisterResponse, *domain.DomainError) {
	start := time.Now()
	log := logger.With(slog.String("service", "register_user"), slog.String("request_id", reqid), slog.String("user_email", req.Email))

	existing, err := s.Users.GetByEmail(ctx, req.Email)
	if existing != nil && err == nil {
		log.Warn("register_user_failed",
			slog.String("step", "check_user_existance"),
			slog.String("reason", "email already exists"))
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "email already exists", nil)
	}

	hashedPassword, err := s.PasswordHasher.HashPassword(req.Password)
	if err != nil {
		log.Error("register_user_failed",
			slog.String("step", "hash_password"),
			slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "registration failed", err)
	}

	user := &domain.User{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    hashedPassword,
		Preferences: req.Preferences,
	}

	tx, err := s.DbPool.Begin(ctx)
	if err != nil {
		log.Error("register_user_failed", slog.String("step", "transaction_begin"), slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "registration failed", err)
	}
	defer tx.Rollback(ctx)
	err = s.Users.CreateUserStaging(ctx, tx, user)
	if err != nil {
		log.Error("register_user_failed", slog.String("step", "create_user_staging"), slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to create user", err)
	}

	err = s.Users.SaveEmailByReqID(ctx, tx, reqid, req.Email)
	if err != nil {
		log.Error("register_user_failed", slog.String("step", "save_email_reqid"), slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to save email-request_id pair", err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		log.Error("register_user_failed_transaction_end", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "registration failed", err)
	}

	otp, err := adapters.GenerateOTP(cfg.OTPLength)
	if err != nil {
		log.Error("register_user_failed_generate_otp", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to generate otp code", err)
	}

	timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	defer cancelFunc()

	otpErrChan := make(chan error, 1)
	emailErrChan := make(chan error, 1)

	go func() {
		otperr := s.Otp.SaveOTP(timeoutCtx, req.Email, otp, cfg.OTPExpiration)
		if err != nil {
			otpErrChan <- otperr
		}
		close(otpErrChan)
	}()

	go func() {
		emailerr := s.Mailer.SendVerification(req.Email, otp)
		if emailerr != nil {
			emailErrChan <- emailerr
		}
		close(emailErrChan)
	}()

	err = <-otpErrChan
	if err != nil {
		log.Error("register_user_failed_save_otp", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to save otp code", err)
	}

	err = <-emailErrChan
	if err != nil {
		log.Error("register_user_failed_send_otp_email", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeExternal, "failed to send otp code", err)
	}

	log.Info("register_user_successful", slog.Int("duration_us", int(time.Since(start).Microseconds())))

	return &domain.RegisterResponse{Message: "The verification code was sent to your email. Please check your email"}, nil

}

func (s *UserRegisterService) VerifyUser(ctx context.Context, req domain.RegisterVerify, reqid string, logger *slog.Logger) (*domain.RegisterResponse, *domain.DomainError) {

	start := time.Now()
	log := logger.With(slog.String("service", "verify_user"), slog.String("request_id", reqid))

	UserSentOtp, err := strconv.Atoi(req.SentOtpbyUser)
	if err != nil {
		log.Error("verify_user_failed_converting_otp_to_int", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to retrieve user otp", err)
	}

	tx, err := s.DbPool.Begin(ctx)
	if err != nil {
		log.Error("verify_user_failed_transaction_begin", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to verify user", err)
	}
	defer tx.Rollback(ctx)
	email, err := s.Users.GetEmailByReqID(ctx, tx, reqid)
	if err != nil {
		log.Error("verify_user_failed_get_email_reqid", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to retrive user email by request id", err)
	}

	err = s.Otp.VerifyOTP(ctx, email, UserSentOtp)
	if err != nil {
		log.Error("verify_user_failed_verufy_otp", slog.String("reason", err.Error()))
		if errors.Is(err, domain.ErrInvalidOtp) {
			return nil, domain.ErrInvalidOtp
		} else if errors.Is(err, domain.ErrTooManyAttempts) {
			return nil, domain.ErrTooManyAttempts
		} else {
			return nil, domain.NewDomainError(domain.ErrCodeUnauthorized, "failed to verify the user", err)
		}

	}
	err = s.Users.MoveUserFromStaging(ctx, tx, email)
	if err != nil {
		log.Error("verify_user_failed_move_user_from_stage", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to move user from staging table to users table", err)
	}

	err = s.Users.DeleteUserFromStaging(ctx, tx, email)
	if err != nil {
		log.Error("verify_user_failed_delete_user_from_stage", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to remove user from staging table", err)
	}

	// err = s.Users.DeleteUserFromEmailVerification(ctx, tx, email)
	// if err != nil {
	// 	apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
	// 	return nil, apiErr
	// }

	err = tx.Commit(ctx)
	if err != nil {
		log.Error("verify_user_failed_transaction_end", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to verify user", err)
	}

	log.Info("verify_user_successful", slog.Int("duration_us", int(time.Since(start).Microseconds())))
	return &domain.RegisterResponse{Message: "You are verified. Welcome 🥰. Lets explore🚀"}, nil

}

func (s *UserRegisterService) AuthenticateUser(ctx context.Context, req domain.LoginUser, cfg *config.Config, logger *slog.Logger) (*domain.RegisterResponse, *domain.DomainError) {

	start := time.Now()
	log := logger.With(slog.String("service", "authenticate"), slog.String("email", req.Email))

	u, err := s.Users.GetByEmail(ctx, req.Email)
	if err != nil {
		log.Error("authenticate_failed_get_user_by_email", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "user not found", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password))
	if err != nil {
		log.Error("authenticate_failed_verify_user_password", slog.String("reason", err.Error()))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, domain.NewDomainError(domain.ErrCodeValidation, "invalid credentials", err)
		}
		return nil, domain.NewDomainError(domain.ErrCodeUnauthorized, "invalid credentials", err)
	}

	token, err := adapters.CreateJWTToken(u.ID, u.Email, []byte(cfg.JwtSecret), cfg.JwtISS)
	if err != nil {
		log.Error("authenticate_failed_create_jwt", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to generate jwt code", err)
	}

	log.Info("authenticate_successful", slog.Int("duration_us", int(time.Since(start).Microseconds())))
	return &domain.RegisterResponse{Message: token}, nil

}

func (s *UserRegisterService) DeleteUser(ctx context.Context, req domain.User, logger *slog.Logger) (*domain.RegisterResponse, *domain.DomainError) {
	start := time.Now()
	log := logger.With(slog.String("service", "delete_user"), slog.String("email", req.Email), slog.Int("user_id", req.ID))
	err := s.Users.DeleteByID(ctx, req.ID)
	if err != nil {
		log.Error("delete_by_id_failed", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "user not found", err)
	}
	log.Info("delete_user_completed", slog.Int("duration_us", int(time.Since(start).Microseconds())))
	return &domain.RegisterResponse{Message: "User deleted succussfully"}, nil
}
