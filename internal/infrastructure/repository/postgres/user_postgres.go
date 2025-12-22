package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepo struct {
	Db *pgx.Conn
}

// type PostgresPoolUserRepo struct {
// 	Db *pgxpool.Pool
// }

type UserRepo struct {
	Db *pgxpool.Pool
}

type UserVerificationRepo struct {
	Db *pgxpool.Pool
}

type StoryRepo struct {
	Db *pgxpool.Pool
}

type RefreshTokenRepo struct {
	Db *pgxpool.Pool
}

// func NewPostgresUserRepo(db *pgx.Conn) *PostgresUserRepo {
// 	return &PostgresUserRepo{Db: db}
// }

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db}
}

func NewUserVerificationRepo(db *pgxpool.Pool) *UserVerificationRepo {
	return &UserVerificationRepo{db}
}

func NewStoryRepo(db *pgxpool.Pool) *StoryRepo {
	return &StoryRepo{db}
}

func NewRefreshTokenRepo(db *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{db}
}

func OpenDatabaseConnPool(dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, domain.ErrDbConnection
	}
	return pool, nil

}

func OpenDatabaseConn(dsn string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, domain.ErrDbConnection
	}
	return conn, nil
}

func (u *UserRepo) DeleteUserByID(ctx context.Context, id int) error {

	var returnedID int

	query := `delete from users where id = $1 returning id`
	row := u.Db.QueryRow(ctx, query, id)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return nil

}

func (u *UserRepo) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	var user domain.User

	query := `select id, first_name, last_name, email from users where id=$1`
	row := u.Db.QueryRow(ctx, query, id)
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return &user, nil

}

func (u *UserRepo) GetUserPreferencesByID(ctx context.Context, id int) (*domain.Preferences, error) {
	var user domain.Preferences

	query := `select id, user_id, preferences from users_preferences where user_id=$1`
	row := u.Db.QueryRow(ctx, query, id)
	err := row.Scan(&user.ID, &user.UserID, &user.UserPreferences)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return &user, nil

}

func (u *UserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	query := `select id, first_name, last_name, email, password from users where email=$1`
	row := u.Db.QueryRow(ctx, query, email)
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrEmailNotFound
	}
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return &user, nil
}

func (v *UserVerificationRepo) CreateUser(ctx context.Context, u *domain.User) error {

	var returnedID int

	query := `insert into staging_users 
			   (first_name, last_name, email, password, preferences) 
			   values ($1, $2, $3, $4, $5) returning id
	`

	row := v.Db.QueryRow(ctx, query, u.FirstName, u.LastName, u.Email, u.Password, u.Preferences)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPersistUser
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}

	return nil
}

func (v *UserVerificationRepo) PersistUserInfo(ctx context.Context, email string) (int, []string, error) {
	query := `
	WITH selected AS (
    SELECT first_name, last_name, email, password, preferences
    FROM staging_users
    WHERE email = $1
)
	INSERT INTO users (first_name, last_name, email, password)
	SELECT first_name, last_name, email, password
	FROM selected
	RETURNING id,
		(SELECT preferences FROM selected);
`

	var returnedID int
	var preferences []string
	row := v.Db.QueryRow(ctx, query, email)
	err := row.Scan(&returnedID, &preferences)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, []string{}, domain.ErrPersistUser
	}
	if err != nil {
		return 0, []string{}, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}

	return returnedID, preferences, nil

}

func (v *UserVerificationRepo) PersistUserPreferenes(ctx context.Context, user_id int, preferences []string) error {
	query := `Insert into users_preferences (user_id, preferences) values ($1, $2) returning id`

	var returnedID int
	row := v.Db.QueryRow(ctx, query, user_id, preferences)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPersistUser
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}

	return nil
}

func (v *UserVerificationRepo) DeleteUserFromStaging(ctx context.Context, email string) error {

	var returnedID int

	query := `delete from staging_users where email = $1 returning id`
	row := v.Db.QueryRow(ctx, query, email)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrEmailNotFound
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}

	return nil

}

func (v *UserVerificationRepo) SaveVerificationData(ctx context.Context, reqid, email string) error {
	query := `insert into email_verification 
			   (request_id, email) 
			   values ($1, $2) returning id`

	var returnedID int
	row := v.Db.QueryRow(ctx, query, reqid, email)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPersistVerification
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}

	return nil
}

func (v *UserVerificationRepo) RetrieveVerificationData(ctx context.Context, reqid string) (string, error) {
	var e string

	query := `select email from email_verification where request_id=$1`
	row := v.Db.QueryRow(ctx, query, reqid)
	err := row.Scan(&e)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrRequestIDNotFound
	}
	if err != nil {
		return "", domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}

	return e, nil
}

func (v *UserVerificationRepo) DeleteUserVerificationData(ctx context.Context, email string) error {

	var returnedID int

	query := `delete from email_verification where email = $1 returning id`
	row := v.Db.QueryRow(ctx, query, email)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrEmailNotFound
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}

	return nil

}

func (s *StoryRepo) SaveStoryInfo(ctx context.Context, i *domain.Story) (int, error) {
	query := `
	insert into stories
	(file_name, user_id, story)
	values ($1, $2, $3)
	returning id
	`
	var returnedID int

	row := s.Db.QueryRow(ctx, query, i.FileName, i.UserID, i.Story)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrPersistStory
	}
	if err != nil {
		return 0, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return returnedID, nil

}

func (s *StoryRepo) UploadStory(ctx context.Context, story *domain.UploadStory) error {

	query := `
	UPDATE stories
	SET
    	story = $1,
    	updated_at = NOW()
	WHERE user_id = $2
	RETURNING id;
	`
	var returnedID int
	row := s.Db.QueryRow(ctx, query, story.Story, story.UserID)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return nil
}

func (s *StoryRepo) SaveStoryJob(ctx context.Context, storyID int, status string) (int, error) {
	query := `
	insert into story_jobs
	(story_id, status)
	values ($1, $2)
	returning id`

	var returnedID int
	row := s.Db.QueryRow(ctx, query, storyID, status)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrStoryNotFound
	}
	if err != nil {
		return 0, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return returnedID, nil

}

func (s *StoryRepo) UpdateStoryJob(ctx context.Context, storyID int, status string) error {
	query := `
	UPDATE story_jobs
			SET 
				status=$1
			WHERE story_id = $2 RETURNING id;
	`

	var returnedID int
	row := s.Db.QueryRow(ctx, query, status, storyID)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrStoryNotFound
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return nil

}

func (s *StoryRepo) UpdateEmailJob(ctx context.Context, storyID int, userID int, status string) error {
	query := `
	UPDATE email_jobs
			SET 
				status=$1
			WHERE story_id = $2 AND user_id = $3 RETURNING id;
	`

	var returnedID int
	row := s.Db.QueryRow(ctx, query, status, storyID, userID)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrStoryNotFound
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return nil

}

func (s *StoryRepo) SaveEmailJob(ctx context.Context, storyID, userID int, status string) (int, error) {
	query := `
	insert into email_jobs
	(story_id, user_id, status)
	values ($1, $2, $3)
	returning id`

	var returnedID int
	row := s.Db.QueryRow(ctx, query, storyID, userID, status)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrStoryNotFound
	}
	if err != nil {
		return 0, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return returnedID, nil

}

func (r *RefreshTokenRepo) StoreRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	query := `
	insert into refresh_tokens
	(user_id, token_hash, expires_at)
	values ($1, $2, $3)
	returning id
	`
	var returnedID uuid.UUID

	row := r.Db.QueryRow(ctx, query, userID, tokenHash, expiresAt)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPersistRefreshToken
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RetrieveRefreshToken(ctx context.Context, userID int) (*domain.RefreshToken, error) {
	query := `
	SELECT 
        id::text,
        user_id,
        token_hash,
        expires_at,
        revoked_at,
        created_at
    FROM refresh_tokens
    WHERE user_id = $1
	AND revoked_at IS NULL
	ORDER BY created_at DESC
	LIMIT 1;
	`
	var refreshToken domain.RefreshToken

	row := r.Db.QueryRow(ctx, query, userID)
	err := row.Scan(&refreshToken.ID, &refreshToken.UserID, &refreshToken.HashedToken, &refreshToken.ExpiredAt, &refreshToken.RevokedAt, &refreshToken.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return &refreshToken, nil
}

func (r *RefreshTokenRepo) UpdateRefreshToken(ctx context.Context, userID int, revokedAt time.Time) error {
	query := `
	UPDATE refresh_tokens
	SET revoked_at = $2
	WHERE id = (
    SELECT id
    FROM refresh_tokens
    WHERE user_id = $1
      AND revoked_at IS NULL
    ORDER BY created_at DESC
    LIMIT 1
)
	RETURNING id;
	`
	var returnedID uuid.UUID

	row := r.Db.QueryRow(ctx, query, userID, revokedAt)
	err := row.Scan(&returnedID)
	fmt.Println(returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPersistRefreshToken
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return nil
}
