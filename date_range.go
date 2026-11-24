package civil

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// dateRangeTextLen is the length of [YYYY-MM-DD,YYYY-MM-DD).
	dateRangeTextLen = 2*dateLen + len("[,)")

	// dateRangeJSONLen is the length of the JSON object form of a range.
	dateRangeJSONLen = 2*dateLen + len(`{"start":"","end":""}`)
)

// A DateRange is a nonempty range of calendar dates. The start is included
// and the end is excluded. For example, June 1 to July 1 includes June 30
// and excludes July 1.
//
// A DateRange comes only from a constructor, so both dates are days that
// exist and the end is after the start. DateRanges are comparable with == and
// usable as map keys.
//
// The zero value is the one exception: it has no dates, so it is not a range.
// It stands for a missing one, IsValid reports false for it, and other
// methods are meaningless on it.
type DateRange struct {
	// start is the first included date.
	start Date

	// end is the first excluded date, strictly after start.
	end Date
}

// NewDateRange returns the range from start up to but excluding end.
//
// It returns an error if either date is the zero Date or end is not strictly
// after start.
func NewDateRange(start, end Date) (DateRange, error) {
	r := DateRange{start: start, end: end}
	if err := r.validate(); err != nil {
		return DateRange{}, err
	}

	return r, nil
}

// NewDateRangeFromString returns the range written in s as
// [YYYY-MM-DD,YYYY-MM-DD), each endpoint a date as NewDateFromString reads
// it.
func NewDateRangeFromString(s string) (DateRange, error) {
	const comma = 1 + dateLen

	if len(s) != dateRangeTextLen || s[0] != '[' || s[comma] != ',' || s[len(s)-1] != ')' {
		return DateRange{}, fmt.Errorf("invalid date range %q: must be [YYYY-MM-DD,YYYY-MM-DD)", s)
	}

	start, err := NewDateFromString(s[1:comma])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid date range %q: start: %w", s, err)
	}

	end, err := NewDateFromString(s[comma+1 : len(s)-1])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid date range %q: end: %w", s, err)
	}

	r, err := NewDateRange(start, end)
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid date range %q: %w", s, err)
	}

	return r, nil
}

// Start returns the first date included in r.
func (r DateRange) Start() Date { return r.start }

// End returns the first date excluded after the range.
func (r DateRange) End() Date { return r.end }

// IsValid reports whether r is a range. It is false only for the zero value,
// which stands for a missing one.
func (r DateRange) IsValid() bool {
	return r.start.IsValid()
}

// Days returns the number of calendar days in r, as End().DaysSince(Start()).
func (r DateRange) Days() int {
	return r.end.DaysSince(r.start)
}

// ContainsDate reports whether d is a day in r.
func (r DateRange) ContainsDate(d Date) bool {
	return !d.Before(r.start) && d.Before(r.end)
}

// Contains reports whether every date in r2 is in r.
// The ranges may share either or both endpoints.
func (r DateRange) Contains(r2 DateRange) bool {
	return !r2.start.Before(r.start) && !r2.end.After(r.end)
}

// Overlaps reports whether r and r2 share at least one date.
// Adjacent ranges share no dates.
func (r DateRange) Overlaps(r2 DateRange) bool {
	return r.start.Before(r2.end) && r2.start.Before(r.end)
}

// Merge returns the union of two overlapping or adjacent ranges.
// A gap between the ranges yields the zero value and false.
func (r DateRange) Merge(r2 DateRange) (DateRange, bool) {
	if r.end.Before(r2.start) || r2.end.Before(r.start) {
		return DateRange{}, false
	}

	if r2.start.Before(r.start) {
		r.start = r2.start
	}

	if r2.end.After(r.end) {
		r.end = r2.end
	}

	return r, true
}

// Intersect returns the range of the dates that r and r2 share. Ranges that
// share none, adjacent ones included, yield the zero value and false.
func (r DateRange) Intersect(r2 DateRange) (DateRange, bool) {
	if !r.Overlaps(r2) {
		return DateRange{}, false
	}

	if r2.start.After(r.start) {
		r.start = r2.start
	}

	if r2.end.Before(r.end) {
		r.end = r2.end
	}

	return r, true
}

// Compare orders two ranges for sorting by start date, then by end date.
// The result is negative, zero or positive as in Date.Compare.
func (r DateRange) Compare(r2 DateRange) int {
	return cmp.Or(r.start.Compare(r2.start), r.end.Compare(r2.end))
}

// String returns [start,end), with each endpoint formatted by Date.String.
func (r DateRange) String() string {
	return string(r.appendTo(make([]byte, 0, dateRangeTextLen)))
}

// AppendText appends the String form of the range to b and returns the
// extended slice. It makes DateRange an encoding.TextAppender.
//
// For the zero DateRange, which is not valid, it returns b unchanged and an
// error.
func (r DateRange) AppendText(b []byte) ([]byte, error) {
	if !r.IsValid() {
		return b, errors.New("cannot marshal the zero DateRange")
	}

	if err := errors.Join(r.start.checkYear(), r.end.checkYear()); err != nil {
		return b, err
	}

	return r.appendTo(b), nil
}

// MarshalText returns the String form of the range. It returns the same
// errors as AppendText. It makes DateRange an encoding.TextMarshaler, which
// encoding/json uses for map keys.
func (r DateRange) MarshalText() ([]byte, error) {
	return r.AppendText(make([]byte, 0, dateRangeTextLen))
}

// UnmarshalText sets r to the range that NewDateRangeFromString reads from
// text. If the text is not a range, it returns the error and leaves r as it
// was. It makes *DateRange an encoding.TextUnmarshaler.
func (r *DateRange) UnmarshalText(text []byte) error {
	parsed, err := NewDateRangeFromString(string(text))
	if err != nil {
		return err
	}

	*r = parsed

	return nil
}

// MarshalJSON returns the range as an object with the two dates:
// {"start":"2026-06-01","end":"2026-07-01"}. The start is inclusive and the
// end is exclusive. It returns the same errors as AppendText. It makes
// DateRange a json.Marshaler.
func (r DateRange) MarshalJSON() ([]byte, error) {
	if !r.IsValid() {
		return nil, errors.New("cannot marshal the zero DateRange")
	}

	if err := errors.Join(r.start.checkYear(), r.end.checkYear()); err != nil {
		return nil, err
	}

	return r.appendJSON(make([]byte, 0, dateRangeJSONLen)), nil
}

// UnmarshalJSON sets r from an object with a start and an end field, both
// required, as NewDateRange would build it. On error r is unchanged. A JSON
// null leaves r unchanged and is not an error. It makes *DateRange a
// json.Unmarshaler.
func (r *DateRange) UnmarshalJSON(data []byte) error {
	// A pointer target leaves the decoder to recognize null.
	var raw *dateRangeJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid date range: %w", err)
	}

	if raw == nil {
		return nil
	}

	if raw.Start == nil {
		return errors.New("invalid date range: start is required")
	}

	if raw.End == nil {
		return errors.New("invalid date range: end is required")
	}

	parsed, err := NewDateRange(*raw.Start, *raw.End)
	if err != nil {
		return fmt.Errorf("invalid date range: %w", err)
	}

	*r = parsed

	return nil
}

func (r DateRange) validate() error {
	if !r.start.IsValid() {
		return errors.New("start must not be the zero Date")
	}

	if !r.end.IsValid() {
		return errors.New("end must not be the zero Date")
	}

	if !r.start.Before(r.end) {
		return errors.New("end must be after start")
	}

	return nil
}

// appendTo appends the String form of the range to b.
func (r DateRange) appendTo(b []byte) []byte {
	b = append(b, '[')
	b = r.start.appendTo(b)
	b = append(b, ',')
	b = r.end.appendTo(b)

	return append(b, ')')
}

// appendJSON appends the JSON object form of the range to b. It quotes each
// date as it is, because the text of a date holds only digits and hyphens.
func (r DateRange) appendJSON(b []byte) []byte {
	b = append(b, `{"start":"`...)
	b = r.start.appendTo(b)
	b = append(b, `","end":"`...)
	b = r.end.appendTo(b)

	return append(b, '"', '}')
}

// dateRangeJSON is the JSON object form of a DateRange. The fields are
// pointers so that decoding can tell a missing field from an empty one.
type dateRangeJSON struct {
	// Start is the inclusive start. Nil detects an absent or null JSON field.
	Start *Date `json:"start"`

	// End is the exclusive end. Nil detects an absent or null JSON field.
	End *Date `json:"end"`
}
