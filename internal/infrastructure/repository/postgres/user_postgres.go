package postgres

import (
	"context"
	"errors"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
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

	query := `select id, first_name, last_name, email, preferences from users where id=$1`
	row := u.Db.QueryRow(ctx, query, id)
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Preferences)
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

	query := `select id, first_name, last_name, email, password, preferences from users where email=$1`
	row := u.Db.QueryRow(ctx, query, email)
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password, &user.Preferences)
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

func (v *UserVerificationRepo) MoveUserFromStaging(ctx context.Context, email string) error {
	query := `Insert into users select * from staging_users where email=$1 returning id`

	var returnedID int
	row := v.Db.QueryRow(ctx, query, email)
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

func (s *StoryRepo) Save(ctx context.Context, i *domain.Story) error {
	query := `
	insert into stories
	(file_name, user_id, story, status)
	values ($1, $2, $3, $4)
	returning id
	`
	var returnedID int

	row := s.Db.QueryRow(ctx, query, i.FileName, i.UserID, i.Story, i.Status)
	err := row.Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPersistStory
	}
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, "query failed", err)
	}
	return nil

}

func (s *StoryRepo) Upload(ctx context.Context, story *domain.UploadStory) error {

	query := `UPDATE stories
			SET 
				story = $1,
				updated_at = NOW(),
				status='completed'
			WHERE user_id = $2 RETURNING id;
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
