-- +goose Up
-- +goose StatementBegin
alter table users 
add column preferences JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table users 
drop column preferences;
-- +goose StatementEnd