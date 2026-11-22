package civil

import (
	"cmp"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

const (
	monthsPerYear = 12

	// The day arithmetic works in 400-year eras that start on March 1, so the
	// leap day is the last day of a year and of an era.
	yearsPerEra = 400
	daysPerEra  = 146097

	// marchOffset is the number of days from March 1 of year 0, where era 0
	// starts, to January 1 of year 1, which is day number 0.
	marchOffset = 306
)

// daysInMonth holds the length of each month, January first, in a non-leap
// year.
var daysInMonth = [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// A Date is a calendar day (year, month, day).
//
// A Date comes only from a constructor or from arithmetic on another Date, so
// it always names a day that exists; there is no February 30. Dates are
// comparable with == and usable as map keys.
//
// The zero value is the one exception: it is not a day. It stands for a
// missing date, IsValid reports false for it, and other methods are
// meaningless on it.
type Date struct {
	// year is the calendar year. Year 0 comes before year 1, and negative
	// years before that.
	year int

	// month is the month of the year, January through December.
	month time.Month

	// day is the day of the month, from 1 to the length of that month.
	day int
}

// NewDate returns the date with the given year, month and day.
//
// It returns an error if the month is not January through December or the
// day does not exist in that month. It never normalizes: February 30 is an
// error, not March 1 or 2.
func NewDate(year int, month time.Month, day int) (Date, error) {
	d := Date{year: year, month: month, day: day}
	if err := d.validate(); err != nil {
		return Date{}, err
	}

	return d, nil
}

// NewDateFromTime returns the date on which t falls in t's own location. Any
// time.Time falls on an existing day, so it cannot fail.
func NewDateFromTime(t time.Time) Date {
	year, month, day := t.Date()

	return Date{year: year, month: month, day: day}
}

// NewDateFromString returns the date written in s as YYYY-MM-DD, the RFC 3339
// full-date format and the form that String produces for years 0 to 9999.
func NewDateFromString(s string) (Date, error) {
	if len(s) != len(time.DateOnly) || s[4] != '-' || s[7] != '-' {
		return Date{}, fmt.Errorf("invalid date %q: must be YYYY-MM-DD", s)
	}

	year, yearOK := parseDigits(s[0:4])
	month, monthOK := parseDigits(s[5:7])
	day, dayOK := parseDigits(s[8:10])

	if !yearOK || !monthOK || !dayOK {
		return Date{}, fmt.Errorf("invalid date %q: must be YYYY-MM-DD", s)
	}

	d, err := NewDate(year, time.Month(month), day)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: %w", s, err)
	}

	return d, nil
}

// Year returns the year of the date.
func (d Date) Year() int { return d.year }

// Month returns the month of the date.
func (d Date) Month() time.Month { return d.month }

// Day returns the day of the month, starting at 1.
func (d Date) Day() int { return d.day }

// Weekday returns the day of the week of the date.
func (d Date) Weekday() time.Weekday {
	// Day number 0 is a Monday. The second modulo keeps the result
	// non-negative for dates before it.
	return time.Weekday(((d.dayNumber()+1)%7 + 7) % 7)
}

// IsValid reports whether d is a day. It is false only for the zero value,
// which stands for a missing date; every Date from a constructor is valid.
func (d Date) IsValid() bool {
	return d != Date{}
}

// String returns the date as YYYY-MM-DD.
//
// A year outside 0 to 9999 is written with a sign and all its digits, as in
// ISO 8601: +10000-01-01. NewDateFromString does not read that form.
func (d Date) String() string {
	return string(d.appendTo(make([]byte, 0, len(time.DateOnly))))
}

// In returns the instant at which the date starts in loc: the result of
// time.Date for midnight of that day.
//
// A zone that skips midnight on that day has no such instant. time.Date then
// picks a nearby one, which can fall on the previous day, and In returns it
// unchanged. A nil loc panics, as it does in time.Date.
func (d Date) In(loc *time.Location) time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, loc)
}

// At returns the instant at which a clock in loc shows t on the date: the
// result of time.Date for that day and time.
//
// A wall time that a zone skips or repeats on that day has no single instant.
// time.Date then picks one, and At returns it unchanged. A nil loc panics, as
// it does in time.Date.
func (d Date) At(t Time, loc *time.Location) time.Time {
	return time.Date(d.year, d.month, d.day, t.hour, t.minute, t.second, t.nanosecond, loc)
}

// AddDays returns the date n days after d. A negative n goes into the past.
func (d Date) AddDays(n int) Date {
	return newDateFromDayNumber(d.dayNumber() + int64(n))
}

// AddMonths returns the date n calendar months after d. A negative n goes
// into the past.
//
// The day of the month is kept. When the target month is too short for it,
// the surplus days run on into the following month: January 31 plus one month
// is March 3, or March 2 in a leap year. time.Time.AddDate does the same.
func (d Date) AddMonths(n int) Date {
	// Count months from January of year 0, so one division splits the sum
	// back into a year and a month for any sign of n.
	months := int64(d.year)*monthsPerYear + int64(d.month-time.January) + int64(n)
	year := floorDiv(months, monthsPerYear)
	month := time.January + time.Month(months-year*monthsPerYear)

	return Date{year: int(year), month: month, day: 1}.AddDays(d.day - 1)
}

// AddYears returns the date n calendar years after d. A negative n goes into
// the past.
//
// The month and day are kept. February 29 moved to a year without one becomes
// March 1, as with time.Time.AddDate.
func (d Date) AddYears(n int) Date {
	return Date{year: d.year + n, month: d.month, day: 1}.AddDays(d.day - 1)
}

// DaysSince returns how many days d is after d2; the result is negative when
// d is before d2. For any two dates, d2.AddDays(d.DaysSince(d2)) == d.
func (d Date) DaysSince(d2 Date) int {
	return int(d.dayNumber() - d2.dayNumber())
}

// Before reports whether d is earlier than d2.
func (d Date) Before(d2 Date) bool {
	return d.Compare(d2) < 0
}

// After reports whether d is later than d2.
func (d Date) After(d2 Date) bool {
	return d.Compare(d2) > 0
}

// Compare orders two dates for sorting, as cmp.Compare does: negative when d
// is before d2, positive when it is after, zero for the same day.
func (d Date) Compare(d2 Date) int {
	return cmp.Or(
		cmp.Compare(d.year, d2.year),
		cmp.Compare(d.month, d2.month),
		cmp.Compare(d.day, d2.day),
	)
}

// AppendText appends the String form of the date to b and returns the
// extended slice. It never fails. It makes Date an encoding.TextAppender.
func (d Date) AppendText(b []byte) ([]byte, error) {
	return d.appendTo(b), nil
}

// MarshalText returns the String form of the date. It never fails. It makes
// Date an encoding.TextMarshaler, which encoding/json uses for values and for
// map keys.
func (d Date) MarshalText() ([]byte, error) {
	return d.appendTo(make([]byte, 0, len(time.DateOnly))), nil
}

// UnmarshalText sets d to the date that NewDateFromString reads from text. If
// the text is not a date, it returns the error and leaves d as it was. It
// makes *Date an encoding.TextUnmarshaler.
func (d *Date) UnmarshalText(text []byte) error {
	parsed, err := NewDateFromString(string(text))
	if err != nil {
		return err
	}

	*d = parsed

	return nil
}

// Value returns the String form of the date for use as a query argument. It
// makes Date a driver.Valuer.
func (d Date) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan sets d from a database column value. It makes *Date a sql.Scanner.
//
// Drivers deliver a DATE column in one of three forms. From a time.Time, Scan
// takes the date in that time's location. A string or []byte must be in the
// format that NewDateFromString accepts. NULL is an error; scan a nullable
// column into a sql.Null[Date]. On error d is unchanged.
func (d *Date) Scan(src any) error {
	switch value := src.(type) {
	case time.Time:
		*d = NewDateFromTime(value)

		return nil
	case string:
		return d.UnmarshalText([]byte(value))
	case []byte:
		return d.UnmarshalText(value)
	case nil:
		return errors.New("invalid date: NULL")
	default:
		return fmt.Errorf("invalid date: unsupported type %T", src)
	}
}

func (d Date) validate() error {
	if d.month < time.January || d.month > time.December {
		return fmt.Errorf("month %d must be in 1..12", int(d.month))
	}

	last := daysInMonth[d.month-time.January]
	if d.month == time.February && isLeap(d.year) {
		last++
	}

	if d.day < 1 || d.day > last {
		return fmt.Errorf("day %d must be in 1..%d for %s %d", d.day, last, d.month, d.year)
	}

	return nil
}

// dayNumber returns the number of days from January 1 of year 1 to the date,
// negative for earlier dates. Arithmetic on dates happens on this number.
//
// The count works in 400-year eras that start on March 1, which puts the leap
// day at the end of a year; plain integer division then gives the days before
// any year and any month.
func (d Date) dayNumber() int64 {
	marchYear := int64(d.year)
	monthFromMarch := int64(d.month - time.March)

	// January and February belong to the year that started the previous March.
	if monthFromMarch < 0 {
		monthFromMarch += monthsPerYear
		marchYear--
	}

	era := floorDiv(marchYear, yearsPerEra)
	yearOfEra := marchYear - era*yearsPerEra
	dayOfYear := (153*monthFromMarch+2)/5 + int64(d.day) - 1
	dayOfEra := yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear

	return era*daysPerEra + dayOfEra - marchOffset
}

// appendTo appends the String form of the date to b.
func (d Date) appendTo(b []byte) []byte {
	b = appendYear(b, d.year)
	b = append(b, '-')
	b = appendPadded(b, int(d.month), 2)
	b = append(b, '-')

	return appendPadded(b, d.day, 2)
}

// newDateFromDayNumber returns the date with the given day number. It is the
// inverse of Date.dayNumber, and every day number is an existing day.
func newDateFromDayNumber(dayNumber int64) Date {
	shifted := dayNumber + marchOffset
	era := floorDiv(shifted, daysPerEra)
	dayOfEra := shifted - era*daysPerEra
	yearOfEra := (dayOfEra - dayOfEra/1460 + dayOfEra/36524 - dayOfEra/146096) / 365
	dayOfYear := dayOfEra - (365*yearOfEra + yearOfEra/4 - yearOfEra/100)
	// monthFromMarch counts months from March: March is 0, February is 11.
	monthFromMarch := (5*dayOfYear + 2) / 153
	day := dayOfYear - (153*monthFromMarch+2)/5 + 1

	year := yearOfEra + era*yearsPerEra
	month := time.March + time.Month(monthFromMarch)

	if month > time.December {
		month -= monthsPerYear
		year++
	}

	return Date{year: int(year), month: month, day: int(day)}
}

// isLeap reports whether year is a leap year of the Gregorian calendar.
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// floorDiv returns a divided by b, rounded toward negative infinity. Go's own
// division rounds toward zero, which is wrong for dates before era 0.
func floorDiv(a, b int64) int64 {
	quotient := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		quotient--
	}

	return quotient
}

// appendYear appends year with at least four digits. A year outside 0 to 9999
// gets an explicit sign, the ISO 8601 expanded form.
func appendYear(b []byte, year int) []byte {
	if year < 0 {
		b = append(b, '-')
		year = -year
	} else if year > 9999 {
		b = append(b, '+')
	}

	return appendPadded(b, year, 4)
}
