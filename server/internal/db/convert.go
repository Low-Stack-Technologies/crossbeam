package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UUIDToPg converts a uuid.UUID to pgtype.UUID for use with generated queries.
func UUIDToPg(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(u), Valid: true}
}

// PgToUUID converts a pgtype.UUID to uuid.UUID.
func PgToUUID(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}

// PgToUUIDPtr returns nil if the pgtype.UUID is not valid.
func PgToUUIDPtr(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	u := uuid.UUID(p.Bytes)
	return &u
}

// UUIDPtrToPg converts a *uuid.UUID to pgtype.UUID (invalid if nil).
func UUIDPtrToPg(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: [16]byte(*u), Valid: true}
}

// TimeFromPg extracts the time.Time from a pgtype.Timestamptz.
func TimeFromPg(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time
}

// TimeToPg converts a time.Time to pgtype.Timestamptz.
func TimeToPg(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
