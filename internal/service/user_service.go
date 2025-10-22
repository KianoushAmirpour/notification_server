package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

type UserRegisterService struct {
	Users          repository.UserRepository
	PasswordHasher repository.PasswordHasher
	Mailer         repository.Mailer
	Otp            repository.OTPService
	DbPool         *pgxpool.Pool
}

var apiErr *domain.APIError

func NewUserRegisterService(
	users repository.UserRepository,
	passwordhasher repository.PasswordHasher,
	mailer repository.Mailer,
	otp repository.OTPService,
	dbpool *pgxpool.Pool) *UserRegisterService {
	return &UserRegisterService{Users: users, PasswordHasher: passwordhasher, Mailer: mailer, Otp: otp, DbPool: dbpool}
}

func (s *UserRegisterService) RegisterUser(ctx context.Context, req domain.RegisteredUser, cfg *config.Config, reqid string, logger *slog.Logger) (*domain.RegisterResponse, *domain.APIError) {
	start := time.Now()
	log := logger.With(slog.String("service", "register_user"), slog.String("request_id", reqid), slog.String("user_email", req.Email))

	existing, _ := s.Users.GetByEmail(ctx, req.Email)
	if existing != nil {
		log.Warn("register_user_failed_check_user_existance", slog.String("reason", "email_exists"))
		return nil, domain.NewAPIError(errors.New("email already exists"), http.StatusConflict)
	}

	hashedPassword, err := s.PasswordHasher.HashPassword(req.Password)
	if err != nil {
		log.Error("register_user_failed_hash_password", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
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
		log.Error("register_user_failed_transaction_begin", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}
	defer tx.Rollback(ctx)
	err = s.Users.CreateUserStaging(ctx, tx, user)
	if err != nil {
		log.Error("register_user_failed_create_staging_user", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	err = s.Users.SaveEmailByReqID(ctx, tx, reqid, req.Email)
	if err != nil {
		log.Error("register_user_failed_save_email_reqid", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}
	err = tx.Commit(ctx)
	if err != nil {
		log.Error("register_user_failed_transaction_end", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	otp, err := adapters.GenerateOTP(cfg.OTPLength)
	if err != nil {
		log.Error("register_user_failed_generate_otp", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	defer cancelFunc()

	eg, _ := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		return s.Otp.SaveOTP(timeoutCtx, req.Email, otp, cfg.OTPExpiration)
	})

	eg.Go(func() error {
		return s.Mailer.SendVerification(req.Email, otp)
	})

	err = eg.Wait()
	if err != nil {
		log.Error("register_user_failed_save_send_otp", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	log.Info("register_user_successful", slog.Int("duration_us", int(time.Since(start).Microseconds())))

	return &domain.RegisterResponse{Message: "The verification code was sent to your email. Please check your email"}, nil

}

func (s *UserRegisterService) VerifyUser(ctx context.Context, req domain.RegisterVerify, reqid string, logger *slog.Logger) (*domain.RegisterResponse, *domain.APIError) {

	start := time.Now()
	log := logger.With(slog.String("service", "verify_user"), slog.String("request_id", reqid))

	UserSentOtp, err := strconv.Atoi(req.SentOtpbyUser)
	if err != nil {
		log.Error("verify_user_failed_converting_otp_to_int", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	tx, err := s.DbPool.Begin(ctx)
	if err != nil {
		log.Error("verify_user_failed_transaction_begin", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}
	defer tx.Rollback(ctx)
	email, err := s.Users.GetEmailByReqID(ctx, tx, reqid)
	if err != nil {
		log.Error("verify_user_failed_get_email_reqid", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	err = s.Otp.VerifyOTP(ctx, email, UserSentOtp)
	if err != nil {
		log.Error("verify_user_failed_verufy_otp", slog.String("reason", err.Error()))
		if errors.Is(err, domain.ErrInvalidOtp) {
			return nil, domain.NewAPIError(err, http.StatusUnauthorized)
		} else if errors.Is(err, domain.ErrTooManyAttempts) {
			return nil, domain.NewAPIError(err, http.StatusTooManyRequests)
		} else {
			return nil, domain.NewAPIError(err, http.StatusUnauthorized)
		}

	}
	err = s.Users.MoveUserFromStaging(ctx, tx, email)
	if err != nil {
		log.Error("verify_user_failed_move_user_from_stage", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	err = s.Users.DeleteUserFromStaging(ctx, tx, email)
	if err != nil {
		log.Error("verify_user_failed_delete_user_from_stage", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	// err = s.Users.DeleteUserFromEmailVerification(ctx, tx, email)
	// if err != nil {
	// 	apiErr = domain.NewAPIError(err, http.StatusInternalServerError)
	// 	return nil, apiErr
	// }

	err = tx.Commit(ctx)
	if err != nil {
		log.Error("verify_user_failed_transaction_end", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	log.Info("verify_user_successful", slog.Int("duration_us", int(time.Since(start).Microseconds())))
	return &domain.RegisterResponse{Message: "You are verified. Welcome 🥰. Lets explore🚀"}, nil

}

func (s *UserRegisterService) AuthenticateUser(ctx context.Context, req domain.LoginUser, cfg *config.Config, logger *slog.Logger) (*domain.RegisterResponse, *domain.APIError) {

	start := time.Now()
	log := logger.With(slog.String("service", "authenticate"), slog.String("email", req.Email))

	u, err := s.Users.GetByEmail(ctx, req.Email)
	if err != nil {
		log.Error("authenticate_failed_get_user_by_email", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusNotFound)
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password))
	if err != nil {
		log.Error("authenticate_failed_verify_user_password", slog.String("reason", err.Error()))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, domain.NewAPIError(errors.New("invalid credentials"), http.StatusUnauthorized)
		}
		return nil, domain.NewAPIError(err, http.StatusUnauthorized)
	}

	token, err := adapters.CreateJWTToken(u.ID, u.Email, []byte(cfg.JwtSecret), cfg.JwtISS)
	if err != nil {
		log.Error("authenticate_failed_create_jwt", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	log.Info("authenticate_successful", slog.Int("duration_us", int(time.Since(start).Microseconds())))
	return &domain.RegisterResponse{Message: token}, nil

}

func (s *UserRegisterService) DeleteUser(ctx context.Context, req domain.User, logger *slog.Logger) (*domain.RegisterResponse, *domain.APIError) {
	start := time.Now()
	log := logger.With(slog.String("service", "delete_user"), slog.String("email", req.Email), slog.Int("user_id", req.ID))
	err := s.Users.DeleteByID(ctx, req.ID)
	if err != nil {
		log.Error("delete_by_id_failed", slog.String("reason", err.Error()))
		return nil, domain.NewAPIError(err, http.StatusNotFound)
	}
	log.Info("delete_user_completed", slog.Int("duration_us", int(time.Since(start).Microseconds())))
	return &domain.RegisterResponse{Message: "User deleted succussfully"}, nil
}
