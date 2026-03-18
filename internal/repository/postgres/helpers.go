package postgres

import "time"

// nullTime converts a time.Time to a *time.Time that is nil when the time is zero.
func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
