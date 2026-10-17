-- +goose Up
-- +goose StatementBegin
ALTER TABLE profiles
ADD COLUMN staff_role TEXT,
ADD CONSTRAINT staff_role_valid CHECK (staff_role IN ('pm', 'contributor'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS staff_role_valid;
ALTER TABLE profiles DROP COLUMN IF EXISTS staff_role;
-- +goose StatementEnd