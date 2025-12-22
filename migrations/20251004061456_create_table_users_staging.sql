-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS staging_users (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(256) NOT NULL,
    last_name VARCHAR(256) NOT NULL,
    email VARCHAR(256) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    password VARCHAR(256) NOT NULL, 
    preferences JSONB NOT NULL

)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE staging_users;
-- +goose StatementEnd
