-- +goose Up
-- +goose StatementBegin
-- Alter table to add user roles to users table
ALTER TABLE users
ADD COLUMN role TEXT NOT NULL DEFAULT 'client',
ADD CONSTRAINT role_valid CHECK (role IN ('admin', 'staff', 'client'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT IF EXISTS role_valid;
ALTER TABLE users DROP COLUMN IF EXISTS role;
-- +goose StatementEnd
