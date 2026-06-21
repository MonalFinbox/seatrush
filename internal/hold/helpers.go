package hold

import (
	"strconv"
	"time"
)

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

func parseUnix(s string) time.Time {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(n, 0)
}
