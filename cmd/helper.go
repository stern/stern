package cmd

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cast"
)

func toTime(a any) time.Time {
	t, _ := toTimeE(a)
	return t
}

func toTimeE(a any) (time.Time, error) {
	switch v := a.(type) {
	case string:
		if t, ok := parseUnixTimeNanoString(v); ok {
			return t, nil
		}
	case json.Number:
		if t, ok := parseUnixTimeNanoString(v.String()); ok {
			return t, nil
		}
	}
	return cast.ToTimeE(a)
}

func parseUnixTimeNanoString(num string) (time.Time, bool) {
	parts := strings.Split(num, ".")
	if len(parts) > 2 {
		return time.Time{}, false
	}

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	var nsec int64
	if len(parts) == 2 {
		// convert fraction part to nanoseconds
		const digits = 9
		frac := parts[1]
		if len(frac) > digits {
			frac = frac[:digits]
		} else if len(frac) < digits {
			frac = frac + strings.Repeat("0", digits-len(frac))
		}
		nsec, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return time.Time{}, false
		}
	}
	return time.Unix(sec, nsec), true
}
