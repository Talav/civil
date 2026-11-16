package civil_test

import (
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talav/civil"
)

// date builds a Date that the test knows to be valid.
func date(t *testing.T, year int, month time.Month, day int) civil.Date {
	t.Helper()

	d, err := civil.NewDate(year, month, day)
	require.NoError(t, err)

	return d
}

// utc returns midnight UTC of the given day, normalized the way time.Date does.
func utc(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// textValue is what Date, Time and DateTime have in common: one text form used
// for printing, for encoding and for query arguments.
type textValue interface {
	comparable
	fmt.Stringer
	encoding.TextAppender
	encoding.TextMarshaler
	driver.Valuer
}

// assertTextForm checks that every way of writing value gives want, and that
// want reads back as value.
func assertTextForm[T textValue, P interface {
	*T
	encoding.TextUnmarshaler
}](t *testing.T, want string, value T) {
	t.Helper()

	assert.Equal(t, want, value.String())

	text, err := value.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, want, string(text))

	appended, err := value.AppendText([]byte("key="))
	require.NoError(t, err)
	assert.Equal(t, "key="+want, string(appended))

	queryArg, err := value.Value()
	require.NoError(t, err)
	assert.Equal(t, want, queryArg)

	var parsed T

	require.NoError(t, P(&parsed).UnmarshalText(text))
	assert.Equal(t, value, parsed, "the text must read back as the same value")
}

func TestNewDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		year    int
		month   time.Month
		day     int
		want    string
		wantErr string
	}{
		{name: "ordinary day", year: 2026, month: time.November, day: 16, want: "2026-11-16"},
		{name: "leap day", year: 2024, month: time.February, day: 29, want: "2024-02-29"},
		{name: "leap day in a leap century year", year: 2000, month: time.February, day: 29, want: "2000-02-29"},
		{name: "last day of a common february", year: 2025, month: time.February, day: 28, want: "2025-02-28"},
		{name: "last day of a short month", year: 2025, month: time.June, day: 30, want: "2025-06-30"},
		{name: "last day of the year", year: 2025, month: time.December, day: 31, want: "2025-12-31"},
		{name: "three-digit year", year: 476, month: time.September, day: 4, want: "0476-09-04"},
		{name: "first day of year 1", year: 1, month: time.January, day: 1, want: "0001-01-01"},
		{name: "leap day of year 0", year: 0, month: time.February, day: 29, want: "0000-02-29"},
		{name: "last four-digit day", year: 9999, month: time.December, day: 31, want: "9999-12-31"},
		{name: "five-digit year", year: 10000, month: time.January, day: 1, want: "+10000-01-01"},
		{name: "six-digit year", year: 271821, month: time.April, day: 20, want: "+271821-04-20"},
		{name: "negative year", year: -1, month: time.December, day: 31, want: "-0001-12-31"},
		{name: "four-digit negative year", year: -4712, month: time.November, day: 24, want: "-4712-11-24"},
		{name: "all zero", wantErr: "month 0 must be in 1..12"},
		{name: "month above december", year: 2025, month: 13, day: 10, wantErr: "month 13 must be in 1..12"},
		{name: "negative month", year: 2025, month: -3, day: 10, wantErr: "month -3 must be in 1..12"},
		{name: "day zero", year: 2025, month: time.May, wantErr: "day 0 must be in 1..31 for May 2025"},
		{name: "negative day", year: 2025, month: time.May, day: -7, wantErr: "day -7 must be in 1..31 for May 2025"},
		{name: "day past a long month", year: 2025, month: time.May, day: 32, wantErr: "day 32 must be in 1..31 for May 2025"},
		{name: "day past a short month", year: 2025, month: time.June, day: 31, wantErr: "day 31 must be in 1..30 for June 2025"},
		{name: "day past a leap february", year: 2024, month: time.February, day: 30, wantErr: "day 30 must be in 1..29 for February 2024"},
		{name: "leap day in a common year", year: 2025, month: time.February, day: 29, wantErr: "day 29 must be in 1..28 for February 2025"},
		{name: "leap day in a common century year", year: 2100, month: time.February, day: 29, wantErr: "day 29 must be in 1..28 for February 2100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewDate(tt.year, tt.month, tt.day)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.True(t, got.IsZero())

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.year, got.Year())
			assert.Equal(t, tt.month, got.Month())
			assert.Equal(t, tt.day, got.Day())
			assert.Equal(t, tt.want, got.String())
			assert.False(t, got.IsZero())
		})
	}
}

func TestNewDateFromTime(t *testing.T) {
	t.Parallel()

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	// 20:15 UTC on March 9 is 05:15 on March 10 in Tokyo.
	instant := time.Date(2026, time.March, 9, 20, 15, 0, 0, time.UTC)

	tests := []struct {
		name string
		time time.Time
		want civil.Date
	}{
		{name: "date in UTC", time: instant, want: date(t, 2026, time.March, 9)},
		{name: "same instant in a zone ahead", time: instant.In(tokyo), want: date(t, 2026, time.March, 10)},
		{name: "midnight", time: utc(2026, time.March, 9), want: date(t, 2026, time.March, 9)},
		{name: "zero time", time: time.Time{}, want: date(t, 1, time.January, 1)},
		{name: "year past four digits", time: utc(10000, time.January, 1), want: date(t, 10000, time.January, 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := civil.NewDateFromTime(tt.time)

			assert.Equal(t, tt.want, got)
			assert.False(t, got.IsZero())
		})
	}
}

func TestNewDateFromString(t *testing.T) {
	t.Parallel()

	const formatErr = "must be YYYY-MM-DD"

	tests := []struct {
		name    string
		text    string
		want    civil.Date
		wantErr string
	}{
		{name: "ordinary day", text: "2026-11-16", want: date(t, 2026, time.November, 16)},
		{name: "first day of the year", text: "2025-01-01", want: date(t, 2025, time.January, 1)},
		{name: "last day of the year", text: "2025-12-31", want: date(t, 2025, time.December, 31)},
		{name: "leap day", text: "2024-02-29", want: date(t, 2024, time.February, 29)},
		{name: "padded year", text: "0476-09-04", want: date(t, 476, time.September, 4)},
		{name: "year 0", text: "0000-01-01", want: date(t, 0, time.January, 1)},
		{name: "last four-digit day", text: "9999-12-31", want: date(t, 9999, time.December, 31)},
		{name: "empty", text: "", wantErr: formatErr},
		{name: "word", text: "today", wantErr: formatErr},
		{name: "unpadded year", text: "476-09-04", wantErr: formatErr},
		{name: "trailing space", text: "2025-06-15 ", wantErr: formatErr},
		{name: "leading space", text: " 2025-06-15", wantErr: formatErr},
		{name: "unpadded month", text: "2025-6-15", wantErr: formatErr},
		{name: "unpadded day", text: "2025-06-5", wantErr: formatErr},
		{name: "slashes", text: "2025/06/15", wantErr: formatErr},
		{name: "dots", text: "2025.06.15", wantErr: formatErr},
		{name: "no separators", text: "20250615", wantErr: formatErr},
		{name: "day first", text: "15-06-2025", wantErr: formatErr},
		{name: "letter in the year", text: "2o25-06-15", wantErr: formatErr},
		{name: "letter in the month", text: "2025-o6-15", wantErr: formatErr},
		{name: "letter in the day", text: "2025-06-1S", wantErr: formatErr},
		{name: "sign in the year", text: "-025-06-15", wantErr: formatErr},
		{name: "sign in the month", text: "2025-+6-15", wantErr: formatErr},
		{name: "signed year", text: "+2025-06-15", wantErr: formatErr},
		{name: "five-digit year", text: "+10000-01-01", wantErr: formatErr},
		{name: "timestamp", text: "2025-06-15T09:30:00Z", wantErr: formatErr},
		{name: "zero year, month and day", text: "0000-00-00", wantErr: "month 0 must be in 1..12"},
		{name: "month above december", text: "2025-13-15", wantErr: "month 13 must be in 1..12"},
		{name: "day zero", text: "2025-06-00", wantErr: "day 0 must be in 1..30 for June 2025"},
		{name: "day past a short month", text: "2025-06-31", wantErr: "day 31 must be in 1..30 for June 2025"},
		{name: "leap day in a common year", text: "2025-02-29", wantErr: "day 29 must be in 1..28 for February 2025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewDateFromString(tt.text)
			if tt.wantErr != "" {
				require.EqualError(t, err, fmt.Sprintf("invalid date %q: %s", tt.text, tt.wantErr))
				assert.True(t, got.IsZero())

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.text, got.String())
		})
	}
}

func TestDateIsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date civil.Date
		want bool
	}{
		{name: "zero value", date: civil.Date{}, want: true},
		{name: "earliest day of year 1", date: date(t, 1, time.January, 1)},
		{name: "first day of year 0", date: date(t, 0, time.January, 1)},
		{name: "date of the zero time", date: civil.NewDateFromTime(time.Time{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.date.IsZero())
		})
	}
}

// TestDateAgreesWithTime walks every day of years -400 through 2800, which is
// eight full 400-year cycles and spans year 0, and checks each day against
// the time package.
func TestDateAgreesWithTime(t *testing.T) {
	t.Parallel()

	want := utc(-400, time.January, 1)
	end := utc(2801, time.January, 1)
	first := civil.NewDateFromTime(want)

	var mismatches []string

	for offset := 0; want.Before(end); offset++ {
		got := first.AddDays(offset)

		if got != civil.NewDateFromTime(want) || got.Weekday() != want.Weekday() || got.DaysSince(first) != offset {
			mismatches = append(mismatches, fmt.Sprintf("%s: got %s %s", want.Format(time.DateOnly), got, got.Weekday()))
		}

		want = want.AddDate(0, 0, 1)
	}

	assert.Empty(t, mismatches)
}

// TestDateAddMonthsAndYearsAgreeWithTime checks month and year arithmetic
// against time.Time.AddDate from every day of a leap year and of a common
// year, over offsets that cross year, century and year-0 boundaries.
func TestDateAddMonthsAndYearsAgreeWithTime(t *testing.T) {
	t.Parallel()

	offsets := []int{-24301, -2025, -1200, -25, -13, -12, -11, -1, 0, 1, 2, 11, 12, 13, 25, 1200, 24301}

	var mismatches []string

	for start := utc(2024, time.January, 1); start.Year() < 2026; start = start.AddDate(0, 0, 1) {
		d := civil.NewDateFromTime(start)

		for _, n := range offsets {
			if got, want := d.AddMonths(n), civil.NewDateFromTime(start.AddDate(0, n, 0)); got != want {
				mismatches = append(mismatches, fmt.Sprintf("%s.AddMonths(%d) = %s, want %s", d, n, got, want))
			}

			if got, want := d.AddYears(n), civil.NewDateFromTime(start.AddDate(n, 0, 0)); got != want {
				mismatches = append(mismatches, fmt.Sprintf("%s.AddYears(%d) = %s, want %s", d, n, got, want))
			}
		}
	}

	assert.Empty(t, mismatches)
}

func TestDateIn(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	tests := []struct {
		name string
		date civil.Date
		loc  *time.Location
		want time.Time
	}{
		{name: "UTC", date: date(t, 2026, time.March, 9), loc: time.UTC, want: utc(2026, time.March, 9)},
		{name: "zone behind UTC in summer time", date: date(t, 2026, time.March, 9), loc: chicago, want: time.Date(2026, time.March, 9, 5, 0, 0, 0, time.UTC)},
		{name: "day of a daylight saving change", date: date(t, 2026, time.March, 8), loc: chicago, want: time.Date(2026, time.March, 8, 6, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.date.In(tt.loc)

			assert.True(t, tt.want.Equal(got), "got %s", got)
			assert.Equal(t, tt.loc, got.Location())
			assert.Equal(t, tt.date, civil.NewDateFromTime(got))
		})
	}
}

func TestDateAt(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	tests := []struct {
		name string
		date civil.Date
		time civil.Time
		loc  *time.Location
		want time.Time
	}{
		{name: "midnight is the start of the day", date: date(t, 2026, time.March, 9), time: civil.Time{}, loc: time.UTC, want: utc(2026, time.March, 9)},
		{name: "wall time in UTC", date: date(t, 2026, time.March, 9), time: clock(t, 14, 30, 15, 250), loc: time.UTC, want: time.Date(2026, time.March, 9, 14, 30, 15, 250, time.UTC)},
		{name: "session open in winter time", date: date(t, 2026, time.March, 6), time: clock(t, 8, 30, 0, 0), loc: chicago, want: time.Date(2026, time.March, 6, 14, 30, 0, 0, time.UTC)},
		{name: "session open in summer time", date: date(t, 2026, time.March, 9), time: clock(t, 8, 30, 0, 0), loc: chicago, want: time.Date(2026, time.March, 9, 13, 30, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.date.At(tt.time, tt.loc)

			assert.True(t, tt.want.Equal(got), "got %s", got)
			assert.Equal(t, tt.loc, got.Location())
			assert.Equal(t, tt.date, civil.NewDateFromTime(got))
			assert.Equal(t, tt.time, civil.NewTimeFromTime(got))
		})
	}
}

func TestDateAddDays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start civil.Date
		days  int
		want  civil.Date
	}{
		{name: "no days", start: date(t, 2026, time.March, 9), days: 0, want: date(t, 2026, time.March, 9)},
		{name: "next day", start: date(t, 2026, time.March, 9), days: 1, want: date(t, 2026, time.March, 10)},
		{name: "into the next month", start: date(t, 2026, time.April, 30), days: 1, want: date(t, 2026, time.May, 1)},
		{name: "into the next year", start: date(t, 2025, time.December, 31), days: 1, want: date(t, 2026, time.January, 1)},
		{name: "back into the previous year", start: date(t, 2026, time.January, 1), days: -1, want: date(t, 2025, time.December, 31)},
		{name: "over a leap day", start: date(t, 2024, time.February, 28), days: 2, want: date(t, 2024, time.March, 1)},
		{name: "over the end of a common february", start: date(t, 2025, time.February, 28), days: 1, want: date(t, 2025, time.March, 1)},
		{name: "a common year", start: date(t, 2024, time.July, 4), days: 365, want: date(t, 2025, time.July, 4)},
		{name: "a year holding a leap day", start: date(t, 2023, time.July, 4), days: 366, want: date(t, 2024, time.July, 4)},
		{name: "a week across a weekend", start: date(t, 2026, time.November, 13), days: 7, want: date(t, 2026, time.November, 20)},
		{name: "before 1970", start: date(t, 1929, time.October, 24), days: 5, want: date(t, 1929, time.October, 29)},
		{name: "into year 1", start: date(t, 0, time.December, 31), days: 1, want: date(t, 1, time.January, 1)},
		{name: "into a negative year", start: date(t, 0, time.January, 1), days: -1, want: date(t, -1, time.December, 31)},
		{name: "all four-digit years", start: date(t, 0, time.January, 1), days: 3652424, want: date(t, 9999, time.December, 31)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.start.AddDays(tt.days))
			assert.Equal(t, tt.days, tt.want.DaysSince(tt.start))
			assert.Equal(t, tt.start, tt.want.AddDays(-tt.days))
		})
	}
}

func TestDateAddMonths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		start  civil.Date
		months int
		want   civil.Date
	}{
		{name: "no months", start: date(t, 2026, time.March, 9), months: 0, want: date(t, 2026, time.March, 9)},
		{name: "next month", start: date(t, 2026, time.March, 9), months: 1, want: date(t, 2026, time.April, 9)},
		{name: "into the next year", start: date(t, 2025, time.November, 20), months: 3, want: date(t, 2026, time.February, 20)},
		{name: "back into the previous year", start: date(t, 2026, time.February, 20), months: -3, want: date(t, 2025, time.November, 20)},
		{name: "back from january", start: date(t, 2026, time.January, 15), months: -1, want: date(t, 2025, time.December, 15)},
		{name: "two years as months", start: date(t, 2024, time.May, 31), months: 24, want: date(t, 2026, time.May, 31)},
		{name: "day kept in a long month", start: date(t, 2026, time.July, 31), months: 1, want: date(t, 2026, time.August, 31)},
		{name: "surplus day runs on from a short month", start: date(t, 2026, time.August, 31), months: 1, want: date(t, 2026, time.October, 1)},
		{name: "surplus days run on from a common february", start: date(t, 2025, time.January, 31), months: 1, want: date(t, 2025, time.March, 3)},
		{name: "surplus days run on from a leap february", start: date(t, 2024, time.January, 31), months: 1, want: date(t, 2024, time.March, 2)},
		{name: "back into year 0", start: date(t, 1, time.February, 10), months: -3, want: date(t, 0, time.November, 10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.start.AddMonths(tt.months))
		})
	}
}

func TestDateAddYears(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start civil.Date
		years int
		want  civil.Date
	}{
		{name: "no years", start: date(t, 2026, time.March, 9), years: 0, want: date(t, 2026, time.March, 9)},
		{name: "next year", start: date(t, 2026, time.March, 9), years: 1, want: date(t, 2027, time.March, 9)},
		{name: "a decade back", start: date(t, 2026, time.March, 9), years: -10, want: date(t, 2016, time.March, 9)},
		{name: "leap day to another leap year", start: date(t, 2024, time.February, 29), years: 4, want: date(t, 2028, time.February, 29)},
		{name: "leap day to a common year", start: date(t, 2024, time.February, 29), years: 1, want: date(t, 2025, time.March, 1)},
		{name: "leap day back to a common year", start: date(t, 2024, time.February, 29), years: -1, want: date(t, 2023, time.March, 1)},
		{name: "past four-digit years", start: date(t, 9999, time.June, 1), years: 2, want: date(t, 10001, time.June, 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.start.AddYears(tt.years))
		})
	}
}

func TestDateCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		date  civil.Date
		other civil.Date
		want  int
	}{
		{name: "same day", date: date(t, 2026, time.March, 9), other: date(t, 2026, time.March, 9), want: 0},
		{name: "day before", date: date(t, 2026, time.March, 8), other: date(t, 2026, time.March, 9), want: -1},
		{name: "day after", date: date(t, 2026, time.March, 10), other: date(t, 2026, time.March, 9), want: 1},
		{name: "earlier month with a later day", date: date(t, 2026, time.February, 28), other: date(t, 2026, time.March, 1), want: -1},
		{name: "earlier year with a later month", date: date(t, 2025, time.December, 31), other: date(t, 2026, time.January, 1), want: -1},
		{name: "later year with an earlier month", date: date(t, 2027, time.January, 1), other: date(t, 2026, time.December, 31), want: 1},
		{name: "negative year against year 0", date: date(t, -1, time.December, 31), other: date(t, 0, time.January, 1), want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.date.Compare(tt.other))
			assert.Equal(t, tt.want < 0, tt.date.Before(tt.other))
			assert.Equal(t, tt.want > 0, tt.date.After(tt.other))
			assert.Equal(t, tt.want == 0, tt.date == tt.other)
		})
	}
}

func TestDateWeekday(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date civil.Date
		want time.Weekday
	}{
		{name: "monday", date: date(t, 2026, time.November, 16), want: time.Monday},
		{name: "friday", date: date(t, 2026, time.November, 20), want: time.Friday},
		{name: "saturday", date: date(t, 2026, time.November, 21), want: time.Saturday},
		{name: "sunday", date: date(t, 2026, time.November, 22), want: time.Sunday},
		{name: "before 1970", date: date(t, 1929, time.October, 29), want: time.Tuesday},
		{name: "new year of year 1", date: date(t, 1, time.January, 1), want: time.Monday},
		{name: "last day of year 0", date: date(t, 0, time.December, 31), want: time.Sunday},
		{name: "negative year", date: date(t, -1, time.December, 31), want: time.Friday},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.date.Weekday())
		})
	}
}

func TestDateText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value civil.Date
		want  string
	}{
		{name: "trading day", value: date(t, 2026, time.March, 6), want: "2026-03-06"},
		{name: "leap day in February", value: date(t, 2028, time.February, 29), want: "2028-02-29"},
		{name: "last day of a year", value: date(t, 1999, time.December, 31), want: "1999-12-31"},
		{name: "three-digit year", value: date(t, 812, time.September, 4), want: "0812-09-04"},
		{name: "first day of year 0", value: date(t, 0, time.January, 1), want: "0000-01-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertTextForm(t, tt.want, tt.value)
		})
	}
}

func TestDateJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Trade  civil.Date            `json:"trade"`
		Settle civil.Date            `json:"settle,omitzero"`
		Closes map[civil.Date]string `json:"closes"`
	}

	in := payload{
		Trade:  date(t, 2026, time.March, 9),
		Closes: map[civil.Date]string{date(t, 2026, time.March, 6): "671.25"},
	}

	data, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `{"trade":"2026-03-09","closes":{"2026-03-06":"671.25"}}`, string(data))

	var out payload

	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in, out)
}

func TestDateScan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     any
		want    civil.Date
		wantErr string
	}{
		{name: "time", src: time.Date(2026, time.March, 9, 14, 30, 0, 0, time.UTC), want: date(t, 2026, time.March, 9)},
		{name: "string", src: "2026-03-09", want: date(t, 2026, time.March, 9)},
		{name: "bytes", src: []byte("2026-03-09"), want: date(t, 2026, time.March, 9)},
		{name: "null", src: nil, wantErr: "invalid date: NULL"},
		{name: "unsupported type", src: int64(20260309), wantErr: "invalid date: unsupported type int64"},
		{name: "invalid string", src: "2026-02-30", wantErr: `invalid date "2026-02-30": day 30 must be in 1..28 for February 2026`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := date(t, 2025, time.August, 1)
			got := before

			err := got.Scan(tt.src)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Equal(t, before, got, "a failed scan must leave the date unchanged")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
