-- +goose Up
-- +goose StatementBegin
ALTER TABLE outbox_events
    ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS metadata;
-- +goose StatementEnd
