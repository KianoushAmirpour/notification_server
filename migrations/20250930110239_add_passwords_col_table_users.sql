-- +goose Up
-- +goose StatementBegin
alter table users 
add column password VARCHAR(256);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table users 
drop column password;
-- +goose StatementEnd
