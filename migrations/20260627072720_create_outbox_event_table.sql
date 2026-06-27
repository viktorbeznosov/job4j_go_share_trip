-- +goose Up
-- +goose StatementBegin
create table outbox_event (
    id UUID primary key,
    event_name text not null,
    aggregate_id UUID not null,
    payload jsonb not null,
    created_at timestamptz not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS outbox_event;
-- +goose StatementEnd
