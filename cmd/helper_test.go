package cmd

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestToTimeE(t *testing.T) {
	base := time.Date(2006, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		arg       any
		expected  time.Time
		wantError bool
	}{
		// nanoseconds
		{"1136171045", base, false},
		{"1136171045.0", base, false},
		{"1136171045.1", base.Add(1e8 * time.Nanosecond), false},
		{json.Number("1136171045.1"), base.Add(1e8 * time.Nanosecond), false},
		{"1136171056.02", base.Add(11*time.Second + 2e7*time.Nanosecond), false},
		{"1136171045.000000001", base.Add(1 * time.Nanosecond), false},
		{"1136171045.123456789", base.Add(123456789 * time.Nanosecond), false},
		{"1136171045.12345678912345", base.Add(123456789 * time.Nanosecond), false},
		// cast.ToTimeE
		{1136171045, base, false},
		{"2006-01-02T03:04:05.123456789", base.Add(123456789 * time.Nanosecond), false},
		// error
		{"", time.Time{}, true},
		{".", time.Time{}, true},
		{"a.b", time.Time{}, true},
		{"1.a", time.Time{}, true},
		{"abc", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.arg), func(t *testing.T) {
			tm, err := toTimeE(tt.arg)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, but got no error")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %+v", err)
				return
			}
			if !tt.expected.Equal(tm) {
				t.Errorf("expected %v, but actual %v", tt.expected, tm.UTC())
			}
		})
	}
}
