-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS email_verification (
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(256) NOT NULL,
    email VARCHAR(256) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_verification_staging_users
        FOREIGN KEY (email)
        REFERENCES staging_users(email)
        ON DELETE CASCADE
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE email_verification;
-- +goose StatementEnd

