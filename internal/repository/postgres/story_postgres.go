package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStoryRepo struct {
	Db *pgxpool.Pool
}

func NewPostgresStoryRepo(db *pgxpool.Pool) *PostgresStoryRepo {
	return &PostgresStoryRepo{Db: db}
}

// func (p *PostgresStoryRepo) Upload(ctx context.Context, story *domain.UploadStory) error {
// 	var returnedID int

// 	query := `UPDATE stories
// 			SET
// 				story = $1,
// 				updated_at = NOW()
// 				status='completed'
// 			WHERE user_id = $2
// 			RETURNING id;
// 	`

// 	row := p.Db.QueryRow(ctx, query, story.UserID, story.Story)
// 	err := row.Scan(&returnedID)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
