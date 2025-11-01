package utils

import "time"

func NowUTC() time.Time {
	return time.Now().UTC()
}

func RFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
