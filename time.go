package civil

import (
	"cmp"
	"database/sql/driver"
	"fmt"
	"time"
)

// The text of a time has one of three shapes: without seconds, with seconds,
// or with seconds, a point and one to nine fraction digits.
const (
	hourMinuteLen       = len("15:04")
	hourMinuteSecondLen = len("15:04:05")
	fractionDigits      = 9
	shortestFractionLen = hourMinuteSecondLen + len(".0")
	longestFractionLen  = hourMinuteSecondLen + len(".") + fractionDigits
)

// A Time is a time of day on a 24-hour clock (hour, minute, second,
// nanosecond), with no date and no time zone.
//
// A Time comes only from a constructor, so every part is in range; there is
// no 24:00 and no 09:60. The zero value is midnight, 00:00:00. Times are
// comparable with == and usable as map keys.
type Time struct {
	// hour is the hour of the day, 0 to 23.
	hour int

	// minute is the minute of the hour, 0 to 59.
	minute int

	// second is the second of the minute, 0 to 59. A leap second, 60, cannot
	// be represented.
	second int

	// nanosecond is the nanosecond of the second, 0 to 999999999.
	nanosecond int
}

// NewTime returns the time of day with the given hour, minute, second and
// nanosecond.
//
// It returns an error if any part is out of range. It never normalizes:
// minute 60 is an error, not the next hour.
func NewTime(hour, minute, second, nanosecond int) (Time, error) {
	t := Time{hour: hour, minute: minute, second: second, nanosecond: nanosecond}
	if err := t.validate(); err != nil {
		return Time{}, err
	}

	return t, nil
}

// NewTimeFromTime returns the time of day that t shows in t's own location.
// Any time.Time shows a time of day in range, so it cannot fail.
func NewTimeFromTime(t time.Time) Time {
	hour, minute, second := t.Clock()

	return Time{hour: hour, minute: minute, second: second, nanosecond: t.Nanosecond()}
}

// NewTimeFromString returns the time of day written in s as HH:MM, HH:MM:SS
// or HH:MM:SS followed by a point and one to nine fraction digits. Parts that
// s leaves out are zero.
func NewTimeFromString(s string) (Time, error) {
	t, ok := parseTime(s)
	if !ok {
		return Time{}, fmt.Errorf("invalid time %q: must be HH:MM, HH:MM:SS or HH:MM:SS.fffffffff", s)
	}

	if err := t.validate(); err != nil {
		return Time{}, fmt.Errorf("invalid time %q: %w", s, err)
	}

	return t, nil
}

// Hour returns the hour of the day, 0 to 23.
func (t Time) Hour() int { return t.hour }

// Minute returns the minute of the hour, 0 to 59.
func (t Time) Minute() int { return t.minute }

// Second returns the second of the minute, 0 to 59.
func (t Time) Second() int { return t.second }

// Nanosecond returns the nanosecond of the second, 0 to 999999999.
func (t Time) Nanosecond() int { return t.nanosecond }

// String returns the time as HH:MM:SS. A time with a non-zero nanosecond gets
// a point and all nine fraction digits: 09:30:00.250000000.
func (t Time) String() string {
	return string(t.appendTo(make([]byte, 0, hourMinuteSecondLen)))
}

// Before reports whether t is earlier in the day than t2.
func (t Time) Before(t2 Time) bool {
	return t.Compare(t2) < 0
}

// After reports whether t is later in the day than t2.
func (t Time) After(t2 Time) bool {
	return t.Compare(t2) > 0
}

// Compare orders two times of day for sorting, as cmp.Compare does: negative
// when t is before t2, positive when it is after, zero when they are equal.
func (t Time) Compare(t2 Time) int {
	return cmp.Or(
		cmp.Compare(t.hour, t2.hour),
		cmp.Compare(t.minute, t2.minute),
		cmp.Compare(t.second, t2.second),
		cmp.Compare(t.nanosecond, t2.nanosecond),
	)
}

// AppendText appends the String form of the time to b and returns the
// extended slice. It never fails. It makes Time an encoding.TextAppender.
func (t Time) AppendText(b []byte) ([]byte, error) {
	return t.appendTo(b), nil
}

// MarshalText returns the String form of the time. It never fails. It makes
// Time an encoding.TextMarshaler, which encoding/json uses for values and for
// map keys.
func (t Time) MarshalText() ([]byte, error) {
	return t.appendTo(make([]byte, 0, hourMinuteSecondLen)), nil
}

// UnmarshalText sets t to the time that NewTimeFromString reads from text. If
// the text is not a time, it returns the error and leaves t as it was. It
// makes *Time an encoding.TextUnmarshaler.
func (t *Time) UnmarshalText(text []byte) error {
	parsed, err := NewTimeFromString(string(text))
	if err != nil {
		return err
	}

	*t = parsed

	return nil
}

// Value returns the String form of the time for use as a query argument. It
// makes Time a driver.Valuer.
func (t Time) Value() (driver.Value, error) {
	return t.String(), nil
}

// Scan sets t from a database column value. It makes *Time a sql.Scanner.
//
// Drivers deliver a TIME column in one of three forms. From a time.Time, Scan
// takes the time of day in that time's location. A string or []byte must be
// in a format that NewTimeFromString accepts. NULL is an error; scan a
// nullable column into a sql.Null[Time]. On error t is unchanged.
func (t *Time) Scan(src any) error {
	return scan(t, src, "time", "15:04:05.999999999")
}

func (t Time) validate() error {
	if t.hour < 0 || t.hour > 23 {
		return fmt.Errorf("hour %d must be in 0..23", t.hour)
	}

	if t.minute < 0 || t.minute > 59 {
		return fmt.Errorf("minute %d must be in 0..59", t.minute)
	}

	if t.second < 0 || t.second > 59 {
		return fmt.Errorf("second %d must be in 0..59", t.second)
	}

	if t.nanosecond < 0 || t.nanosecond > 999_999_999 {
		return fmt.Errorf("nanosecond %d must be in 0..999999999", t.nanosecond)
	}

	return nil
}

// appendTo appends the String form of the time to b.
func (t Time) appendTo(b []byte) []byte {
	b = appendPadded(b, t.hour, 2)
	b = append(b, ':')
	b = appendPadded(b, t.minute, 2)
	b = append(b, ':')
	b = appendPadded(b, t.second, 2)

	if t.nanosecond == 0 {
		return b
	}

	b = append(b, '.')

	return appendPadded(b, t.nanosecond, fractionDigits)
}

// parseTime reads the parts of a time of day from s without checking their
// ranges. It reports false if s does not have one of the accepted shapes.
func parseTime(s string) (Time, bool) {
	n := len(s)
	if !isTimeLen(n) {
		return Time{}, false
	}

	hour, hourOK := parseDigits(s[0:2])
	minute, minuteOK := parseDigits(s[3:hourMinuteLen])
	t := Time{hour: hour, minute: minute}
	ok := s[2] == ':' && hourOK && minuteOK

	if n >= hourMinuteSecondLen {
		second, secondOK := parseDigits(s[hourMinuteLen+1 : hourMinuteSecondLen])
		t.second = second
		ok = ok && s[hourMinuteLen] == ':' && secondOK
	}

	if n >= shortestFractionLen {
		nanosecond, fractionOK := parseFraction(s[hourMinuteSecondLen:])
		t.nanosecond = nanosecond
		ok = ok && fractionOK
	}

	return t, ok
}

// parseFraction reads the fraction of a second, a point and one to nine
// digits, and returns it in nanoseconds.
func parseFraction(s string) (int, bool) {
	nanosecond, ok := parseDigits(s[1:])
	if s[0] != '.' || !ok {
		return 0, false
	}

	// Fewer than nine fraction digits leave out trailing zeros.
	for range len(".") + fractionDigits - len(s) {
		nanosecond *= 10
	}

	return nanosecond, true
}

// isTimeLen reports whether n is the length of one of the accepted shapes.
func isTimeLen(n int) bool {
	return n == hourMinuteLen || n == hourMinuteSecondLen || (n >= shortestFractionLen && n <= longestFractionLen)
}
