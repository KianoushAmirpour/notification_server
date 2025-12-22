-- +goose Up
-- +goose StatementBegin
CREATE TYPE story_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE IF NOT EXISTS story_jobs (
    id SERIAL PRIMARY KEY,
    story_id INT NOT NULL,
    status story_status DEFAULT 'pending' NOT NULL,
    CONSTRAINT fk_story_jobs_stories
        FOREIGN KEY (story_id)
        REFERENCES stories(id)
        ON DELETE CASCADE
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE story_status;
DROP TABLE story_jobs;
-- +goose StatementEnd
