CREATE TABLE booking_seats (
    -- CASCADE: cancelling/deleting a booking removes its seat links, freeing
    -- the seats to be booked again (seat status is reset to 'available' by
    -- the app in the same transaction).
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    -- RESTRICT: a seat that's linked to a booking can't be deleted.
    seat_id    UUID NOT NULL REFERENCES seats(id) ON DELETE RESTRICT,

    -- Composite PK: the same (booking, seat) pair can't appear twice.
    PRIMARY KEY (booking_id, seat_id),

    -- And globally, a seat can belong to at most one booking at a time.
    -- When a booking is cancelled its rows are deleted, so the seat becomes
    -- linkable again — correct, at the cost of not retaining which seats a
    -- cancelled booking held. Accepted trade-off.
    CONSTRAINT uniq_booking_seat UNIQUE (seat_id)
);
