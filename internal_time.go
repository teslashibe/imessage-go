package imessage

import "time"

// appleEpoch is 2001-01-01 UTC. iMessage stores message dates as
// nanoseconds since this epoch (older versions: seconds).
var appleEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// fromAppleDate converts an Apple Cocoa absolute time value to time.Time.
// The chat.db `date`, `date_read`, `date_delivered` columns store either
// seconds or nanoseconds since 2001-01-01 depending on macOS version. We
// detect which by magnitude.
func fromAppleDate(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	// Nanoseconds-since-2001 are ~10^17 for current dates; seconds are ~10^8.
	if v > 1_000_000_000_000 {
		return appleEpoch.Add(time.Duration(v))
	}
	return appleEpoch.Add(time.Duration(v) * time.Second)
}

// toAppleDate is the inverse, used when constructing Since/Until WHERE
// clauses. We always emit nanoseconds (the modern format); chat.db's
// implicit cast treats both representations as numbers in WHERE comparisons
// against the stored value, so we coerce by detecting later.
func toAppleDate(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return int64(t.Sub(appleEpoch))
}
