-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS stories (
    id SERIAL PRIMARY KEY,
    file_name VARCHAR(255) NOT NULL,
    user_id INT NOT NULL,
    story TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT stories_user_file_unique UNIQUE (user_id, file_name),
    CONSTRAINT fk_stories_users
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE stories;
-- +goose StatementEnd
