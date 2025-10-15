package postgres

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/jackc/pgx/v5"
)

// save metadata of generated image in postgres

type PostgresImageRepo struct {
	Db *pgx.Conn
}

func NewPostgresImageRepo(db *pgx.Conn) *PostgresImageRepo {
	return &PostgresImageRepo{Db: db}
}

func (p *PostgresImageRepo) SaveImageInfo(ctx context.Context, i *domain.Image) error {
	query := `
	 insert into images
	 (file_name, user_id, url, ratio, size, status)
	 values ($1, $2, $3, $4, $5, $6) returning id
	 
	`
	var returnedID int

	row := p.Db.QueryRow(ctx, query, i.FileName, i.UserID, i.Url, i.Ratio, i.Size, i.Status)
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	return nil

}
