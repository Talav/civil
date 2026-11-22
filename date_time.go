package civil

import (
	"cmp"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

// dateTimeLen is the length of the String form of a datetime that has no
// fraction of a second.
const dateTimeLen = dateLen + len("T") + hourMinuteSecondLen

// A DateTime is a calendar day together with a time of day, with no time
// zone: what a calendar and a wall clock show at some moment, without saying
// where.
//
// A DateTime comes only from a constructor, so its date is a day that exists
// in the years 0 to 9999 and its time is in range. DateTimes are comparable with == and usable as
// map keys.
//
// The zero value is the one exception: it has no date, so it is not a
// datetime. It stands for a missing one, IsValid reports false for it, and
// other methods are meaningless on it.
type DateTime struct {
	// date is the calendar day. It is never the zero Date.
	date Date

	// time is the time of day on that date.
	time Time
}

// NewDateTime returns the datetime that shows t on the date d.
//
// It returns an error if d is the zero Date, which is not a day.
func NewDateTime(d Date, t Time) (DateTime, error) {
	dt := DateTime{date: d, time: t}
	if err := dt.validate(); err != nil {
		return DateTime{}, err
	}

	return dt, nil
}

// NewDateTimeFromTime returns the date and the time of day that t shows in
// t's own location.
//
// It returns an error if t is in a year outside 0 to 9999, which a time.Time
// can hold and a DateTime cannot.
func NewDateTimeFromTime(t time.Time) (DateTime, error) {
	d, err := NewDateFromTime(t)
	if err != nil {
		return DateTime{}, err
	}

	return NewDateTime(d, NewTimeFromTime(t))
}

// NewDateTimeFromString returns the datetime written in s: a date as
// NewDateFromString reads it, the letter T, and a time as NewTimeFromString
// reads it, such as 2026-03-09T09:30:00. A space is accepted in place of the
// T, which is how databases write a timestamp.
func NewDateTimeFromString(s string) (DateTime, error) {
	// The date has a fixed length, so the separator is always at dateLen.
	if len(s) <= dateLen || (s[dateLen] != 'T' && s[dateLen] != ' ') {
		return DateTime{}, fmt.Errorf("invalid datetime %q: must be YYYY-MM-DD followed by T or a space and a time of day", s)
	}

	d, err := NewDateFromString(s[:dateLen])
	if err != nil {
		return DateTime{}, fmt.Errorf("invalid datetime %q: %w", s, err)
	}

	t, err := NewTimeFromString(s[dateLen+1:])
	if err != nil {
		return DateTime{}, fmt.Errorf("invalid datetime %q: %w", s, err)
	}

	return DateTime{date: d, time: t}, nil
}

// Date returns the calendar day of the datetime.
func (dt DateTime) Date() Date { return dt.date }

// Time returns the time of day of the datetime.
func (dt DateTime) Time() Time { return dt.time }

// IsValid reports whether dt is a datetime. It is false only for the zero
// value, which stands for a missing one. Every valid DateTime can be encoded.
func (dt DateTime) IsValid() bool {
	return dt.date.IsValid()
}

// String returns the datetime as the date, the letter T and the time:
// 2026-03-09T09:30:00. The two parts are written as Date.String and
// Time.String write them.
func (dt DateTime) String() string {
	return string(dt.appendTo(make([]byte, 0, dateTimeLen)))
}

// In returns the instant at which a calendar and a clock in loc show the
// datetime: the result of time.Date for that day and time.
//
// A wall time that a zone skips or repeats on that day has no single instant.
// time.Date then picks one, and In returns it unchanged. A nil loc panics, as
// it does in time.Date.
func (dt DateTime) In(loc *time.Location) time.Time {
	return dt.date.At(dt.time, loc)
}

// Before reports whether dt is earlier than dt2.
func (dt DateTime) Before(dt2 DateTime) bool {
	return dt.Compare(dt2) < 0
}

// After reports whether dt is later than dt2.
func (dt DateTime) After(dt2 DateTime) bool {
	return dt.Compare(dt2) > 0
}

// Compare orders two datetimes for sorting, as cmp.Compare does: negative
// when dt is before dt2, positive when it is after, zero when they are equal.
// Both are read as if in the same location.
func (dt DateTime) Compare(dt2 DateTime) int {
	return cmp.Or(dt.date.Compare(dt2.date), dt.time.Compare(dt2.time))
}

// AppendText appends the String form of the datetime to b and returns the
// extended slice. It makes DateTime an encoding.TextAppender.
//
// For the zero DateTime, which is not valid, it returns b unchanged and an
// error.
func (dt DateTime) AppendText(b []byte) ([]byte, error) {
	if !dt.IsValid() {
		return b, errors.New("cannot marshal the zero DateTime")
	}

	return dt.appendTo(b), nil
}

// MarshalText returns the String form of the datetime. It returns the same
// errors as AppendText. It makes DateTime an encoding.TextMarshaler, which
// encoding/json uses for values and for map keys.
func (dt DateTime) MarshalText() ([]byte, error) {
	return dt.AppendText(make([]byte, 0, dateTimeLen))
}

// UnmarshalText sets dt to the datetime that NewDateTimeFromString reads from
// text. If the text is not a datetime, it returns the error and leaves dt as
// it was. It makes *DateTime an encoding.TextUnmarshaler.
func (dt *DateTime) UnmarshalText(text []byte) error {
	parsed, err := NewDateTimeFromString(string(text))
	if err != nil {
		return err
	}

	*dt = parsed

	return nil
}

// Value returns the String form of the datetime for use as a query argument.
// It returns the same errors as AppendText. It makes DateTime a
// driver.Valuer.
func (dt DateTime) Value() (driver.Value, error) {
	text, err := dt.MarshalText()
	if err != nil {
		return nil, err
	}

	return string(text), nil
}

// Scan sets dt from a database column value. It makes *DateTime a
// sql.Scanner.
//
// Drivers deliver a timestamp column in one of three forms. From a time.Time,
// Scan takes the date and the time of day in that time's location. A string
// or []byte must be in a format that NewDateTimeFromString accepts. A time in
// a year outside 0 to 9999 is an error, and so is NULL; scan a nullable
// column into a sql.Null[DateTime]. On error dt is unchanged.
func (dt *DateTime) Scan(src any) error {
	return scan(dt, src, "datetime", "2006-01-02T15:04:05.999999999")
}

func (dt DateTime) validate() error {
	if !dt.date.IsValid() {
		return errors.New("date must not be the zero Date")
	}

	return nil
}

// appendTo appends the String form of the datetime to b.
func (dt DateTime) appendTo(b []byte) []byte {
	b = dt.date.appendTo(b)
	b = append(b, 'T')

	return dt.time.appendTo(b)
}
