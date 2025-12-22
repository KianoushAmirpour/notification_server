-- +goose Up
-- +goose StatementBegin
CREATE TYPE email_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE IF NOT EXISTS email_jobs (
    id SERIAL PRIMARY KEY,
    story_id INT NOT NULL,
    user_id INT NOT NULL,
    status email_status DEFAULT 'pending' NOT NULL,
    CONSTRAINT fk_email_jobs_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_email_jobs_story
        FOREIGN KEY (story_id)
        REFERENCES stories(id)
        ON DELETE CASCADE
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE email_status;
DROP TABLE email_jobs;
-- +goose StatementEnd
