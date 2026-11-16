package civil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talav/civil"
)

// dateTime builds a DateTime that the test knows to be valid.
func dateTime(t *testing.T, d civil.Date, tm civil.Time) civil.DateTime {
	t.Helper()

	dt, err := civil.NewDateTime(d, tm)
	require.NoError(t, err)

	return dt
}

func TestNewDateTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		date    civil.Date
		time    civil.Time
		want    string
		wantErr string
	}{
		{name: "session open", date: date(t, 2026, time.March, 9), time: clock(t, 9, 30, 0, 0), want: "2026-03-09T09:30:00"},
		{name: "start of the day", date: date(t, 2026, time.March, 9), time: civil.Time{}, want: "2026-03-09T00:00:00"},
		{name: "fraction of a second", date: date(t, 2024, time.February, 29), time: clock(t, 23, 59, 59, 5_000_000), want: "2024-02-29T23:59:59.005000000"},
		{name: "year before 1000", date: date(t, 33, time.April, 3), time: clock(t, 15, 0, 0, 0), want: "0033-04-03T15:00:00"},
		{name: "zero date", date: civil.Date{}, time: clock(t, 9, 30, 0, 0), wantErr: "date must not be the zero Date"},
		{name: "zero date at midnight", date: civil.Date{}, time: civil.Time{}, wantErr: "date must not be the zero Date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewDateTime(tt.date, tt.time)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Equal(t, civil.DateTime{}, got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.date, got.Date())
			assert.Equal(t, tt.time, got.Time())
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestDateTimeIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dateTime civil.DateTime
		want     bool
	}{
		{name: "zero value", dateTime: civil.DateTime{}, want: false},
		{name: "date at midnight", dateTime: dateTime(t, date(t, 2026, time.March, 9), civil.Time{}), want: true},
		{name: "date and time", dateTime: dateTime(t, date(t, 2026, time.March, 9), clock(t, 9, 30, 0, 0)), want: true},
		{name: "datetime of the zero time", dateTime: civil.NewDateTimeFromTime(time.Time{}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.dateTime.IsValid())
		})
	}
}

func TestNewDateTimeFromTime(t *testing.T) {
	t.Parallel()

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	// 20:15:30.5 UTC on March 9 is 05:15:30.5 on March 10 in Tokyo.
	instant := time.Date(2026, time.March, 9, 20, 15, 30, 500_000_000, time.UTC)

	tests := []struct {
		name string
		time time.Time
		want civil.DateTime
	}{
		{
			name: "calendar and clock in UTC",
			time: instant,
			want: dateTime(t, date(t, 2026, time.March, 9), clock(t, 20, 15, 30, 500_000_000)),
		},
		{
			name: "same instant on the next day in a zone ahead",
			time: instant.In(tokyo),
			want: dateTime(t, date(t, 2026, time.March, 10), clock(t, 5, 15, 30, 500_000_000)),
		},
		{
			name: "zero time is midnight on the first day of year 1",
			time: time.Time{},
			want: dateTime(t, date(t, 1, time.January, 1), civil.Time{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := civil.NewDateTimeFromTime(tt.time)

			assert.Equal(t, tt.want, got)
			assert.True(t, tt.time.Equal(got.In(tt.time.Location())), "In must give the instant back")
		})
	}
}

func TestNewDateTimeFromString(t *testing.T) {
	t.Parallel()

	const formatErr = "must be YYYY-MM-DDTHH:MM:SS"

	day := date(t, 2026, time.March, 9)

	tests := []struct {
		name    string
		text    string
		want    civil.DateTime
		wantErr string
	}{
		{name: "with seconds", text: "2026-03-09T16:00:05", want: dateTime(t, day, clock(t, 16, 0, 5, 0))},
		{name: "without seconds", text: "2026-03-09T09:30", want: dateTime(t, day, clock(t, 9, 30, 0, 0))},
		{name: "with a fraction", text: "2026-03-09T09:30:00.25", want: dateTime(t, day, clock(t, 9, 30, 0, 250_000_000))},
		{name: "space as the separator", text: "2026-03-09 16:00:05", want: dateTime(t, day, clock(t, 16, 0, 5, 0))},
		{name: "space and a fraction", text: "2026-03-09 16:00:05.000001", want: dateTime(t, day, clock(t, 16, 0, 5, 1000))},
		{name: "empty", text: "", wantErr: formatErr},
		{name: "date only", text: "2026-03-09", wantErr: formatErr},
		{name: "time only", text: "09:30:00", wantErr: formatErr},
		{name: "lower-case separator", text: "2026-03-09t09:30:00", wantErr: formatErr},
		{name: "underscore separator", text: "2026-03-09_09:30:00", wantErr: formatErr},
		{name: "unpadded month", text: "2026-3-09T09:30:00", wantErr: formatErr},
		{
			name:    "separator without a time",
			text:    "2026-03-09T",
			wantErr: `invalid time "": must be HH:MM, HH:MM:SS or HH:MM:SS.fffffffff`,
		},
		{
			name:    "zone suffix",
			text:    "2026-03-09T09:30:00Z",
			wantErr: `invalid time "09:30:00Z": must be HH:MM, HH:MM:SS or HH:MM:SS.fffffffff`,
		},
		{
			name:    "zone offset",
			text:    "2026-03-09T09:30:00+01:00",
			wantErr: `invalid time "09:30:00+01:00": must be HH:MM, HH:MM:SS or HH:MM:SS.fffffffff`,
		},
		{
			name:    "slashes in the date",
			text:    "2026/03/09T09:30:00",
			wantErr: `invalid date "2026/03/09": must be YYYY-MM-DD`,
		},
		{
			name:    "day that does not exist",
			text:    "2026-02-30T09:30:00",
			wantErr: `invalid date "2026-02-30": day 30 must be in 1..28 for February 2026`,
		},
		{
			name:    "hour past the end of the day",
			text:    "2026-03-09T24:00:00",
			wantErr: `invalid time "24:00:00": hour 24 must be in 0..23`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewDateTimeFromString(tt.text)
			if tt.wantErr != "" {
				require.EqualError(t, err, "invalid datetime \""+tt.text+"\": "+tt.wantErr)
				assert.Equal(t, civil.DateTime{}, got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			again, err := civil.NewDateTimeFromString(got.String())
			require.NoError(t, err)
			assert.Equal(t, got, again, "String must read back as the same datetime")
		})
	}
}

func TestDateTimeIn(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	dt := dateTime(t, date(t, 2026, time.March, 9), clock(t, 8, 30, 0, 0))

	tests := []struct {
		name string
		loc  *time.Location
		want time.Time
	}{
		{name: "wall time in UTC", loc: time.UTC, want: time.Date(2026, time.March, 9, 8, 30, 0, 0, time.UTC)},
		{name: "wall time in daylight time", loc: chicago, want: time.Date(2026, time.March, 9, 13, 30, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := dt.In(tt.loc)

			assert.True(t, tt.want.Equal(got), "got %s", got)
			assert.Equal(t, tt.loc, got.Location())
		})
	}
}

func TestDateTimeCompare(t *testing.T) {
	t.Parallel()

	monday := date(t, 2026, time.March, 9)
	tuesday := date(t, 2026, time.March, 10)

	tests := []struct {
		name     string
		dateTime civil.DateTime
		other    civil.DateTime
		want     int
	}{
		{
			name:     "same day and time",
			dateTime: dateTime(t, monday, clock(t, 9, 30, 0, 0)),
			other:    dateTime(t, monday, clock(t, 9, 30, 0, 0)),
			want:     0,
		},
		{
			name:     "earlier day with a later time",
			dateTime: dateTime(t, monday, clock(t, 23, 59, 59, 0)),
			other:    dateTime(t, tuesday, civil.Time{}),
			want:     -1,
		},
		{
			name:     "later day with an earlier time",
			dateTime: dateTime(t, tuesday, clock(t, 0, 0, 1, 0)),
			other:    dateTime(t, monday, clock(t, 16, 0, 0, 0)),
			want:     1,
		},
		{
			name:     "same day, earlier time",
			dateTime: dateTime(t, monday, clock(t, 9, 30, 0, 0)),
			other:    dateTime(t, monday, clock(t, 16, 0, 0, 0)),
			want:     -1,
		},
		{
			name:     "same day, one nanosecond later",
			dateTime: dateTime(t, monday, clock(t, 9, 30, 0, 1)),
			other:    dateTime(t, monday, clock(t, 9, 30, 0, 0)),
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.dateTime.Compare(tt.other))
			assert.Equal(t, tt.want < 0, tt.dateTime.Before(tt.other))
			assert.Equal(t, tt.want > 0, tt.dateTime.After(tt.other))
			assert.Equal(t, tt.want == 0, tt.dateTime == tt.other)
		})
	}
}

func TestDateTimeText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value civil.DateTime
		want  string
	}{
		{
			name:  "morning of a trading day",
			value: dateTime(t, date(t, 2026, time.March, 6), clock(t, 9, 30, 0, 0)),
			want:  "2026-03-06T09:30:00",
		},
		{
			name:  "midnight of a leap day",
			value: dateTime(t, date(t, 2024, time.February, 29), civil.Time{}),
			want:  "2024-02-29T00:00:00",
		},
		{
			name:  "one nanosecond into a second",
			value: dateTime(t, date(t, 1999, time.December, 31), clock(t, 23, 59, 59, 1)),
			want:  "1999-12-31T23:59:59.000000001",
		},
		{
			name:  "three-digit year in the afternoon",
			value: dateTime(t, date(t, 476, time.September, 4), clock(t, 15, 4, 5, 0)),
			want:  "0476-09-04T15:04:05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertTextForm(t, tt.want, tt.value)
		})
	}
}

func TestDateTimeScan(t *testing.T) {
	t.Parallel()

	want := dateTime(t, date(t, 2026, time.March, 9), clock(t, 14, 30, 0, 0))

	tests := []struct {
		name    string
		src     any
		want    civil.DateTime
		wantErr string
	}{
		{name: "time", src: time.Date(2026, time.March, 9, 14, 30, 0, 0, time.UTC), want: want},
		{name: "string", src: "2026-03-09T14:30:00", want: want},
		{name: "string with a space", src: "2026-03-09 14:30:00", want: want},
		{name: "bytes", src: []byte("2026-03-09 14:30:00"), want: want},
		{name: "null", src: nil, wantErr: "invalid datetime: NULL"},
		{name: "unsupported type", src: int64(1_773_066_600), wantErr: "invalid datetime: unsupported type int64"},
		{
			name:    "invalid string",
			src:     "2026-13-09 14:30:00",
			wantErr: `invalid datetime "2026-13-09 14:30:00": invalid date "2026-13-09": month 13 must be in 1..12`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := dateTime(t, date(t, 2025, time.December, 31), clock(t, 8, 15, 0, 0))
			got := before

			err := got.Scan(tt.src)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Equal(t, before, got, "a failed scan must leave the datetime unchanged")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
