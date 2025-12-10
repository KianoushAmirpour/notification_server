-- +goose Up
-- +goose StatementBegin
ALTER TABLE stories
ADD CONSTRAINT stories_user_file_unique
UNIQUE (user_id, file_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE stories
DROP CONSTRAINT stories_user_file_unique;
-- +goose StatementEnd
