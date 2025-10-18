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

type PostgresPoolUserRepo struct {
	Db *pgxpool.Pool
}

// type User struct {
// 	ID          int
// 	FirstName   string
// 	LastName    string
// 	Email       string
// 	Password    string
// 	Preferences []string
// }

func NewPostgresUserRepo(db *pgx.Conn) *PostgresUserRepo {
	return &PostgresUserRepo{Db: db}
}

func NewPostgresPoolUserRepo(db *pgxpool.Pool) *PostgresPoolUserRepo {
	return &PostgresPoolUserRepo{db}
}

func OpenDatabaseConnPool(dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return pool, nil

}

func OpenDatabaseConn(dsn string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (r *PostgresPoolUserRepo) Create(ctx context.Context, u *domain.User) error {

	var returnedID int

	query := `insert into users 
			   (first_name, last_name, email, password, preferences) 
			   values ($1, $2, $3, $4, $5) returning id
	`

	row := r.Db.QueryRow(ctx, query, u.FirstName, u.LastName, u.Email, u.Password, u.Preferences)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresPoolUserRepo) CreateUserStaging(ctx context.Context, tx pgx.Tx, u *domain.User) error {

	var returnedID int

	query := `insert into staging_users 
			   (first_name, last_name, email, password, preferences) 
			   values ($1, $2, $3, $4, $5) returning id
	`
	row := tx.QueryRow(ctx, query, u.FirstName, u.LastName, u.Email, u.Password, u.Preferences)
	// row := r.Db.QueryRow(ctx, query, u.FirstName, u.LastName, u.Email, u.Password, u.Preferences)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresPoolUserRepo) DeleteByID(ctx context.Context, id int) error {

	var returnedID int

	query := `delete from users where id = $1 returning id`
	row := r.Db.QueryRow(ctx, query, id)
	err := row.Scan(&returnedID)
	if err != nil {
		return errors.New("user not found")
	}

	return nil

}

func (r *PostgresPoolUserRepo) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	var u domain.User

	query := `select id, first_name, last_name, email, preferences from users where id=$1`
	row := r.Db.QueryRow(ctx, query, id)
	err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Preferences)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &u, nil

}

func (r *PostgresPoolUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User

	query := `select id, first_name, last_name, email, password, preferences from users where email=$1`
	row := r.Db.QueryRow(ctx, query, email)
	err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.Preferences)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &u, nil
}

func (r *PostgresPoolUserRepo) MoveUserFromStaging(ctx context.Context, tx pgx.Tx, email string) error {
	query := `Insert into users select * from staging_users where email=$1 returning id`

	var returnedID int
	row := tx.QueryRow(ctx, query, email)
	// row := r.Db.QueryRow(ctx, query, email)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil

}

func (r *PostgresPoolUserRepo) DeleteUserFromStaging(ctx context.Context, tx pgx.Tx, email string) error {

	var returnedID int

	query := `delete from staging_users where email = $1 returning id`
	row := tx.QueryRow(ctx, query, email)
	// row := r.Db.QueryRow(ctx, query, email)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil

}

func (r *PostgresPoolUserRepo) SaveEmailByReqID(ctx context.Context, tx pgx.Tx, reqid, email string) error {
	query := `insert into email_verification 
			   (request_id, email) 
			   values ($1, $2) returning id`

	var returnedID int
	row := tx.QueryRow(ctx, query, reqid, email)
	// row := r.Db.QueryRow(ctx, query, reqid, email)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresPoolUserRepo) GetEmailByReqID(ctx context.Context, tx pgx.Tx, reqid string) (string, error) {
	var e string

	query := `select email from email_verification where request_id=$1`
	row := tx.QueryRow(ctx, query, reqid)
	// row := r.Db.QueryRow(ctx, query, reqid)
	err := row.Scan(&e)
	if err != nil {
		return "", errors.New("invalid request")
	}
	return e, nil
}

func (r *PostgresPoolUserRepo) DeleteUserFromEmailVerification(ctx context.Context, tx pgx.Tx, email string) error {

	var returnedID int

	query := `delete from email_verification where email = $1 returning id`
	row := tx.QueryRow(ctx, query, email)
	// row := r.Db.QueryRow(ctx, query, email)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil

}
