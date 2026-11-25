package civil_test

import (
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"math"
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

// dateOf returns the Date of a time.
func dateOf(t *testing.T, instant time.Time) civil.Date {
	t.Helper()

	return civil.NewDateFromTime(instant)
}

// plusDays returns the date days after d.
func plusDays(t *testing.T, d civil.Date, days int) civil.Date {
	t.Helper()

	return d.AddDays(days)
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

// assertNoTextForm checks that value still prints as display, and that every
// way of encoding it fails with wantErr.
func assertNoTextForm[T textValue](t *testing.T, display, wantErr string, value T) {
	t.Helper()

	assert.Equal(t, display, value.String())

	_, err := value.MarshalText()
	require.EqualError(t, err, wantErr)

	appended, err := value.AppendText([]byte("key="))
	require.EqualError(t, err, wantErr)
	assert.Equal(t, "key=", string(appended), "a failed append must leave the buffer unchanged")

	queryArg, err := value.Value()
	require.EqualError(t, err, wantErr)
	assert.Nil(t, queryArg)

	_, err = json.Marshal(value)
	require.ErrorContains(t, err, wantErr)

	_, err = json.Marshal(map[T]int{value: 1})
	require.ErrorContains(t, err, wantErr)
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
		{name: "five-digit year", year: 10000, month: time.January, day: 1, want: "10000-01-01"},
		{name: "negative year", year: -1, month: time.December, day: 31, want: "000-1-12-31"},
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
				assert.False(t, got.IsValid())

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.year, got.Year())
			assert.Equal(t, tt.month, got.Month())
			assert.Equal(t, tt.day, got.Day())
			assert.Equal(t, tt.want, got.String())
			assert.True(t, got.IsValid())
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
		name    string
		time    time.Time
		want    civil.Date
		wantErr string
	}{
		{name: "date in UTC", time: instant, want: date(t, 2026, time.March, 9)},
		{name: "same instant in a zone ahead", time: instant.In(tokyo), want: date(t, 2026, time.March, 10)},
		{name: "midnight", time: utc(2026, time.March, 9), want: date(t, 2026, time.March, 9)},
		{name: "zero time", time: time.Time{}, want: date(t, 1, time.January, 1)},
		{name: "first day of year 0", time: utc(0, time.January, 1), want: date(t, 0, time.January, 1)},
		{name: "last day of year 9999", time: utc(9999, time.December, 31), want: date(t, 9999, time.December, 31)},
		{name: "year past four digits", time: utc(10000, time.January, 1), wantErr: "year 10000 must be in 0..9999"},
		{name: "negative year", time: utc(-1, time.December, 31), wantErr: "year -1 must be in 0..9999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := civil.NewDateFromTime(tt.time)
			if tt.wantErr != "" {
				_, err := got.MarshalText()
				require.EqualError(t, err, tt.wantErr)

				return
			}

			assert.Equal(t, tt.want, got)
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
				assert.False(t, got.IsValid())

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.text, got.String())
		})
	}
}

func TestDateIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date civil.Date
		want bool
	}{
		{name: "zero value", date: civil.Date{}, want: false},
		{name: "earliest day of year 1", date: date(t, 1, time.January, 1), want: true},
		{name: "first day of year 0", date: date(t, 0, time.January, 1), want: true},
		{name: "date of the zero time", date: dateOf(t, time.Time{}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.date.IsValid())
		})
	}
}

// TestDateAgreesWithTime walks every day of years 0 through 3199, which is
// eight full 400-year cycles from the first year of a Date, and checks each
// day against the time package.
func TestDateAgreesWithTime(t *testing.T) {
	t.Parallel()

	want := utc(0, time.January, 1)
	end := utc(3200, time.January, 1)
	first := dateOf(t, want)

	var mismatches []string

	for offset := 0; want.Before(end); offset++ {
		got := plusDays(t, first, offset)

		if got != dateOf(t, want) || got.Weekday() != want.Weekday() || got.DaysSince(first) != offset {
			mismatches = append(mismatches, fmt.Sprintf("%s: got %s %s", want.Format(time.DateOnly), got, got.Weekday()))
		}

		want = want.AddDate(0, 0, 1)
	}

	assert.Empty(t, mismatches)
}

// TestDateAddMonthsAndYearsAgreeWithTime checks month and year arithmetic
// against time.Time.AddDate from every day of a leap year and of a common
// year, over offsets that cross year, century and millennium boundaries.
func TestDateAddMonthsAndYearsAgreeWithTime(t *testing.T) {
	t.Parallel()

	offsets := []int{-2000, -1200, -25, -13, -12, -11, -1, 0, 1, 2, 11, 12, 13, 25, 1200, 2000}

	var mismatches []string

	for start := utc(2024, time.January, 1); start.Year() < 2026; start = start.AddDate(0, 0, 1) {
		d := dateOf(t, start)

		for _, n := range offsets {
			got := d.AddMonths(n)

			if want := dateOf(t, start.AddDate(0, n, 0)); got != want {
				mismatches = append(mismatches, fmt.Sprintf("%s.AddMonths(%d) = %s, want %s", d, n, got, want))
			}

			got = d.AddYears(n)

			if want := dateOf(t, start.AddDate(n, 0, 0)); got != want {
				mismatches = append(mismatches, fmt.Sprintf("%s.AddYears(%d) = %s, want %s", d, n, got, want))
			}
		}
	}

	assert.Empty(t, mismatches)
}

func TestDateIn(t *testing.T) {
	t.Parallel()

	apia, err := time.LoadLocation("Pacific/Apia")
	require.NoError(t, err)
	skipped := date(t, 2011, time.December, 30)

	// The pairing is valid even though no instant in Apia displays this date.
	day := skipped.In(apia)
	assert.True(t, day.IsValid())
	assert.Equal(t, skipped, day.Date())
	assert.Same(t, apia, day.Location())
	assert.True(t, day.Equal(locationDate(t, skipped, "Pacific/Apia")))

	data, err := json.Marshal(day)
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":"2011-12-30","timeZone":"Pacific/Apia"}`, string(data))

	missing := civil.Date{}.In(time.UTC)
	assert.False(t, missing.IsValid())
	assert.Same(t, time.UTC, missing.Location())

	withoutLocation := skipped.In(nil)
	assert.False(t, withoutLocation.IsValid())
	assert.Equal(t, skipped, withoutLocation.Date())
	assert.Nil(t, withoutLocation.Location())

	open := clock(t, 9, 30, 0, 0)
	for _, loc := range []*time.Location{time.Local, time.FixedZone("Local", 3600), time.FixedZone("", -5400)} {
		day := skipped.In(loc)
		assert.Same(t, loc, day.Location())
		want := time.Date(2011, time.December, 30, 9, 30, 0, 0, loc)
		assert.Equal(t, want, day.At(open))
		assert.Equal(t, want, skipped.At(open).In(loc))
	}
}

func TestDateAt(t *testing.T) {
	t.Parallel()

	d := date(t, 2026, time.March, 9)
	for _, tm := range []civil.Time{{}, clock(t, 14, 30, 15, 250)} {
		t.Run(tm.String(), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, dateTime(t, d, tm), d.At(tm))
			missing := civil.Date{}.At(tm)
			assert.False(t, missing.IsValid())
			assert.Equal(t, tm, missing.Time())
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
		{name: "all four-digit years", start: date(t, 0, time.January, 1), days: 3652424, want: date(t, 9999, time.December, 31)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.start.AddDays(tt.days)
			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.days, tt.want.DaysSince(tt.start))

			back := tt.want.AddDays(-tt.days)
			assert.Equal(t, tt.start, back)
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

			got := tt.start.AddMonths(tt.months)
			assert.Equal(t, tt.want, got)
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
		{name: "to the last year", start: date(t, 9998, time.June, 1), years: 1, want: date(t, 9999, time.June, 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.start.AddYears(tt.years)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDateArithmeticOutOfRange checks that a result outside the years 0 to
// 9999 cannot be encoded.
func TestDateArithmeticOutOfRange(t *testing.T) {
	t.Parallel()

	first := date(t, 0, time.January, 1)
	last := date(t, 9999, time.December, 31)
	leapDay := date(t, 9996, time.February, 29)

	tests := []struct {
		name    string
		add     func(n int) civil.Date
		n       int
		wantErr string
	}{
		{name: "day before the first", add: first.AddDays, n: -1, wantErr: "year -1 must be in 0..9999"},
		{name: "day after the last", add: last.AddDays, n: 1, wantErr: "year 10000 must be in 0..9999"},
		{name: "largest number of days", add: first.AddDays, n: math.MaxInt, wantErr: "must be in 0..9999"},
		{name: "smallest number of days", add: last.AddDays, n: math.MinInt, wantErr: "must be in 0..9999"},
		{name: "month before the first", add: first.AddMonths, n: -1, wantErr: "year -1 must be in 0..9999"},
		{name: "month after the last", add: last.AddMonths, n: 1, wantErr: "year 10000 must be in 0..9999"},
		{name: "largest number of months", add: first.AddMonths, n: math.MaxInt, wantErr: "must be in 0..9999"},
		{name: "smallest number of months", add: last.AddMonths, n: math.MinInt, wantErr: "must be in 0..9999"},
		{name: "year before the first", add: first.AddYears, n: -1, wantErr: "year -1 must be in 0..9999"},
		{name: "year after the last", add: last.AddYears, n: 1, wantErr: "year 10000 must be in 0..9999"},
		{name: "largest number of years", add: first.AddYears, n: math.MaxInt, wantErr: "must be in 0..9999"},
		{name: "smallest number of years", add: last.AddYears, n: math.MinInt, wantErr: "must be in 0..9999"},
		{name: "leap day past the last year", add: leapDay.AddYears, n: 4, wantErr: "year 10000 must be in 0..9999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.add(tt.n).MarshalText()

			// The year in the message of the largest steps depends on the
			// size of an int, so those rows give only its end.
			require.ErrorContains(t, err, tt.wantErr)
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
		{name: "year 0 against year 1", date: date(t, 0, time.December, 31), other: date(t, 1, time.January, 1), want: -1},
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
		{name: "first day of year 0", date: date(t, 0, time.January, 1), want: time.Saturday},
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
		{name: "last four-digit day", value: date(t, 9999, time.December, 31), want: "9999-12-31"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertTextForm(t, tt.want, tt.value)
		})
	}
}

func TestDateTextErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   civil.Date
		display string
		wantErr string
	}{
		{name: "zero value", value: civil.Date{}, display: "0000-00-00", wantErr: "cannot marshal the zero Date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertNoTextForm(t, tt.display, tt.wantErr, tt.value)
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
		{name: "time in year 0", src: utc(0, time.February, 29), want: date(t, 0, time.February, 29)},
		{name: "time past four-digit years", src: utc(10000, time.January, 1), wantErr: `invalid date "10000-01-01": must be YYYY-MM-DD`},
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
