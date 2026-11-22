package civil_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talav/civil"
)

// clock builds a Time that the test knows to be valid.
func clock(t *testing.T, hour, minute, second, nanosecond int) civil.Time {
	t.Helper()

	tm, err := civil.NewTime(hour, minute, second, nanosecond)
	require.NoError(t, err)

	return tm
}

func TestNewTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		hour       int
		minute     int
		second     int
		nanosecond int
		want       string
		wantErr    string
	}{
		{name: "midnight", want: "00:00:00"},
		{name: "market open", hour: 9, minute: 30, want: "09:30:00"},
		{name: "market close", hour: 16, want: "16:00:00"},
		{name: "with seconds", hour: 13, minute: 5, second: 7, want: "13:05:07"},
		{name: "with a quarter second", hour: 9, minute: 30, nanosecond: 250_000_000, want: "09:30:00.250000000"},
		{name: "with one nanosecond", hour: 9, minute: 30, nanosecond: 1, want: "09:30:00.000000001"},
		{name: "last instant of the day", hour: 23, minute: 59, second: 59, nanosecond: 999_999_999, want: "23:59:59.999999999"},
		{name: "hour 24", hour: 24, wantErr: "hour 24 must be in 0..23"},
		{name: "negative hour", hour: -1, wantErr: "hour -1 must be in 0..23"},
		{name: "minute 60", hour: 9, minute: 60, wantErr: "minute 60 must be in 0..59"},
		{name: "negative minute", hour: 9, minute: -5, wantErr: "minute -5 must be in 0..59"},
		{name: "leap second", hour: 23, minute: 59, second: 60, wantErr: "second 60 must be in 0..59"},
		{name: "negative second", hour: 9, second: -1, wantErr: "second -1 must be in 0..59"},
		{name: "a full second of nanoseconds", hour: 9, nanosecond: 1_000_000_000, wantErr: "nanosecond 1000000000 must be in 0..999999999"},
		{name: "negative nanosecond", hour: 9, nanosecond: -1, wantErr: "nanosecond -1 must be in 0..999999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewTime(tt.hour, tt.minute, tt.second, tt.nanosecond)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Equal(t, civil.Time{}, got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.hour, got.Hour())
			assert.Equal(t, tt.minute, got.Minute())
			assert.Equal(t, tt.second, got.Second())
			assert.Equal(t, tt.nanosecond, got.Nanosecond())
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestNewTimeFromTime(t *testing.T) {
	t.Parallel()

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	// 20:15:30.5 UTC is 05:15:30.5 the next morning in Tokyo.
	instant := time.Date(2026, time.March, 9, 20, 15, 30, 500_000_000, time.UTC)

	tests := []struct {
		name string
		time time.Time
		want civil.Time
	}{
		{name: "clock in UTC", time: instant, want: clock(t, 20, 15, 30, 500_000_000)},
		{name: "same instant in a zone ahead", time: instant.In(tokyo), want: clock(t, 5, 15, 30, 500_000_000)},
		{name: "zero time", time: time.Time{}, want: civil.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, civil.NewTimeFromTime(tt.time))
		})
	}
}

func TestNewTimeFromString(t *testing.T) {
	t.Parallel()

	const formatErr = "must be HH:MM, HH:MM:SS or HH:MM:SS.fffffffff"

	tests := []struct {
		name    string
		text    string
		want    civil.Time
		wantErr string
	}{
		{name: "hour and minute", text: "09:30", want: clock(t, 9, 30, 0, 0)},
		{name: "midnight without seconds", text: "00:00", want: civil.Time{}},
		{name: "with seconds", text: "16:00:05", want: clock(t, 16, 0, 5, 0)},
		{name: "one fraction digit", text: "09:30:00.5", want: clock(t, 9, 30, 0, 500_000_000)},
		{name: "three fraction digits", text: "09:30:00.025", want: clock(t, 9, 30, 0, 25_000_000)},
		{name: "nine fraction digits", text: "23:59:59.999999999", want: clock(t, 23, 59, 59, 999_999_999)},
		{name: "zero fraction", text: "09:30:00.000", want: clock(t, 9, 30, 0, 0)},
		{name: "empty", text: "", wantErr: formatErr},
		{name: "plain word", text: "close", wantErr: formatErr},
		{name: "hour only", text: "09", wantErr: formatErr},
		{name: "unpadded hour", text: "9:30", wantErr: formatErr},
		{name: "unpadded minute", text: "09:5", wantErr: formatErr},
		{name: "unpadded second", text: "09:30:5", wantErr: formatErr},
		{name: "no separators", text: "0930", wantErr: formatErr},
		{name: "dot between hour and minute", text: "09.30", wantErr: formatErr},
		{name: "dot before the seconds", text: "09:30.05", wantErr: formatErr},
		{name: "comma before the fraction", text: "09:30:00,5", wantErr: formatErr},
		{name: "point without digits", text: "09:30:00.", wantErr: formatErr},
		{name: "ten fraction digits", text: "09:30:00.1234567890", wantErr: formatErr},
		{name: "letter in the hour", text: "o9:30", wantErr: formatErr},
		{name: "letter in the fraction", text: "09:30:00.5s", wantErr: formatErr},
		{name: "sign in the hour", text: "-9:30", wantErr: formatErr},
		{name: "trailing space", text: "09:30 ", wantErr: formatErr},
		{name: "twelve-hour suffix", text: "09:30 AM", wantErr: formatErr},
		{name: "zone suffix", text: "09:30:00Z", wantErr: formatErr},
		{name: "hour 24", text: "24:00", wantErr: "hour 24 must be in 0..23"},
		{name: "minute 60", text: "09:60", wantErr: "minute 60 must be in 0..59"},
		{name: "leap second", text: "23:59:60", wantErr: "second 60 must be in 0..59"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewTimeFromString(tt.text)
			if tt.wantErr != "" {
				require.EqualError(t, err, fmt.Sprintf("invalid time %q: %s", tt.text, tt.wantErr))
				assert.Equal(t, civil.Time{}, got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			again, err := civil.NewTimeFromString(got.String())
			require.NoError(t, err)
			assert.Equal(t, got, again, "String must read back as the same time")
		})
	}
}

func TestTimeCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		time  civil.Time
		other civil.Time
		want  int
	}{
		{name: "same time", time: clock(t, 9, 30, 0, 0), other: clock(t, 9, 30, 0, 0), want: 0},
		{name: "earlier hour with a later minute", time: clock(t, 9, 59, 0, 0), other: clock(t, 10, 0, 0, 0), want: -1},
		{name: "later hour with an earlier minute", time: clock(t, 16, 0, 0, 0), other: clock(t, 15, 59, 59, 0), want: 1},
		{name: "earlier minute with a later second", time: clock(t, 9, 29, 59, 0), other: clock(t, 9, 30, 0, 0), want: -1},
		{name: "earlier second with a later nanosecond", time: clock(t, 9, 30, 0, 999), other: clock(t, 9, 30, 1, 0), want: -1},
		{name: "one nanosecond later", time: clock(t, 9, 30, 0, 1), other: clock(t, 9, 30, 0, 0), want: 1},
		{name: "midnight against the end of the day", time: civil.Time{}, other: clock(t, 23, 59, 59, 999_999_999), want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.time.Compare(tt.other))
			assert.Equal(t, tt.want < 0, tt.time.Before(tt.other))
			assert.Equal(t, tt.want > 0, tt.time.After(tt.other))
			assert.Equal(t, tt.want == 0, tt.time == tt.other)
		})
	}
}

func TestTimeText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value civil.Time
		want  string
	}{
		{name: "start of the session", value: clock(t, 9, 45, 0, 0), want: "09:45:00"},
		{name: "zero value", value: civil.Time{}, want: "00:00:00"},
		{name: "seconds past the minute", value: clock(t, 16, 0, 7, 0), want: "16:00:07"},
		{name: "half a second", value: clock(t, 12, 0, 0, 500_000_000), want: "12:00:00.500000000"},
		{name: "end of the day", value: clock(t, 23, 59, 59, 999_999_998), want: "23:59:59.999999998"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertTextForm(t, tt.want, tt.value)
		})
	}
}

func TestTimeScan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     any
		want    civil.Time
		wantErr string
	}{
		{name: "time", src: time.Date(2026, time.March, 9, 14, 30, 0, 0, time.UTC), want: clock(t, 14, 30, 0, 0)},
		{name: "time with a fraction", src: time.Date(2026, time.March, 9, 14, 30, 0, 1, time.UTC), want: clock(t, 14, 30, 0, 1)},
		{name: "string", src: "14:30:00", want: clock(t, 14, 30, 0, 0)},
		{name: "bytes", src: []byte("14:30:00"), want: clock(t, 14, 30, 0, 0)},
		{name: "null", src: nil, wantErr: "invalid time: NULL"},
		{name: "unsupported type", src: int64(1430), wantErr: "invalid time: unsupported type int64"},
		{name: "invalid string", src: "25:00", wantErr: `invalid time "25:00": hour 25 must be in 0..23`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := clock(t, 8, 15, 0, 0)
			got := before

			err := got.Scan(tt.src)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Equal(t, before, got, "a failed scan must leave the time unchanged")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
