-- +goose Up
-- +goose StatementBegin

CREATE TABLE trip_publish_notified (
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    passenger_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (trip_id, passenger_id)
);

CREATE INDEX idx_trip_publish_notified_passenger_id
    ON trip_publish_notified (passenger_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS trip_publish_notified;

-- +goose StatementEnd
