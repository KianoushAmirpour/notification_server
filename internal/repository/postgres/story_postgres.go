package postgres

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/jackc/pgx/v5"
)

type PostgresStoryRepo struct {
	Db *pgx.Conn
}

func NewPostgresStoryRepo(db *pgx.Conn) *PostgresStoryRepo {
	return &PostgresStoryRepo{Db: db}
}

func (p *PostgresStoryRepo) SaveStoryInfo(ctx context.Context, i *domain.Story) error {
	query := `
	 insert into images
	 (file_name, user_id, url, status)
	 values ($1, $2, $3, $4) returning id
	 
	`
	var returnedID int

	row := p.Db.QueryRow(ctx, query, i.FileName, i.UserID, i.Url, i.Status)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil

}
