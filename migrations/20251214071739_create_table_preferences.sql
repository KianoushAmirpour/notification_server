-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users_preferences (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    preferences JSONB NOT NULL,
    CONSTRAINT fk_preferences_users
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users_preferences;
-- +goose StatementEnd


   
