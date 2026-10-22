-- +goose Up
-- +goose StatementBegin

CREATE TYPE request_status AS ENUM (
    'draft',
    'published',
    'pre_agreed',
    'agreed',
    'started',
    'canceled',
    'completed'
);

CREATE TABLE trip_requests (
    id UUID PRIMARY KEY,
    passenger_id UUID NOT NULL,
    from_point TEXT NOT NULL,
    to_point TEXT NOT NULL,
    departure_time TIMESTAMPTZ NOT NULL,
    seats INT NOT NULL CHECK (seats > 0),
    status request_status NOT NULL DEFAULT 'draft',
    trip_id UUID REFERENCES trips(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trip_requests_passenger_id ON trip_requests (passenger_id);
CREATE INDEX idx_trip_requests_trip_id ON trip_requests (trip_id);
CREATE INDEX idx_trip_requests_status ON trip_requests (status);

CREATE TABLE request_history (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES trip_requests(id) ON DELETE CASCADE,
    from_status request_status,
    to_status request_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS request_history;
DROP TABLE IF EXISTS trip_requests;
DROP TYPE IF EXISTS request_status;

-- +goose StatementEnd
