-- +goose Up
-- +goose StatementBegin
ALTER TYPE trip_status ADD VALUE IF NOT EXISTS 'preapproved';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
