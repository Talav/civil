package civil

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// dateRangeTextLen is the length of [YYYY-MM-DD,YYYY-MM-DD).
const dateRangeTextLen = 2*dateLen + len("[,)")

// A DateRange is a nonempty range of calendar dates. The start is included
// and the end is excluded. For example, June 1 to July 1 includes June 30
// and excludes July 1. DateRanges are comparable and usable as map keys.
//
// The zero value represents a missing range. IsValid reports false for it,
// and other methods are meaningless on it.
type DateRange struct {
	// start is the first included date.
	start Date

	// end is the first excluded date, strictly after start.
	end Date
}

// NewDateRange returns the range from start up to but excluding end.
// Both dates must be valid and end must be strictly after start.
func NewDateRange(start, end Date) (DateRange, error) {
	r := DateRange{start: start, end: end}
	if err := r.validate(); err != nil {
		return DateRange{}, err
	}

	return r, nil
}

// NewDateRangeFromString reads [YYYY-MM-DD,YYYY-MM-DD), with each endpoint
// in the four-digit-year format accepted by NewDateFromString.
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

// IsValid reports whether r holds a range. It is false only for the zero value.
func (r DateRange) IsValid() bool { return r.start.IsValid() }

// Days returns the number of calendar days in r, as End().DaysSince(Start()).
// The count must fit in an int.
func (r DateRange) Days() int { return r.end.DaysSince(r.start) }

// Contains reports whether d is a day in r.
func (r DateRange) Contains(d Date) bool {
	return !d.Before(r.start) && d.Before(r.end)
}

// Overlaps reports whether r and other share at least one date.
// Adjacent ranges share no dates.
func (r DateRange) Overlaps(other DateRange) bool {
	return r.start.Before(other.end) && other.start.Before(r.end)
}

// Merge returns the union of two overlapping or adjacent ranges.
// A gap between the ranges yields the zero value and false.
func (r DateRange) Merge(other DateRange) (DateRange, bool) {
	if r.end.Before(other.start) || other.end.Before(r.start) {
		return DateRange{}, false
	}

	if other.start.Before(r.start) {
		r.start = other.start
	}

	if other.end.After(r.end) {
		r.end = other.end
	}

	return r, true
}

// Compare orders ranges for sorting by start date, then by end date.
// The result is negative, zero or positive as in Date.Compare.
func (r DateRange) Compare(other DateRange) int {
	return cmp.Or(r.start.Compare(other.start), r.end.Compare(other.end))
}

// String returns [start,end), with each endpoint formatted by Date.String.
// Years outside 0000..9999 use the expanded form.
func (r DateRange) String() string {
	return string(r.appendTo(make([]byte, 0, dateRangeTextLen)))
}

// AppendText appends the text accepted by NewDateRangeFromString to b.
// An endpoint year outside 0000..9999 returns b and an error.
func (r DateRange) AppendText(b []byte) ([]byte, error) {
	if err := r.checkEncodable(); err != nil {
		return b, err
	}

	return r.appendTo(b), nil
}

// MarshalText returns the range text, also used for JSON map keys.
// It returns the same errors as AppendText.
func (r DateRange) MarshalText() ([]byte, error) {
	return r.AppendText(nil)
}

// UnmarshalText sets r to the range read by NewDateRangeFromString.
// It assigns only after both bounds and their order have been checked.
func (r *DateRange) UnmarshalText(text []byte) error {
	parsed, err := NewDateRangeFromString(string(text))
	if err != nil {
		return err
	}

	*r = parsed

	return nil
}

// MarshalJSON writes an object with start and end date strings. The start
// is inclusive and the end is exclusive. An endpoint year outside 0000..9999
// returns an error.
func (r DateRange) MarshalJSON() ([]byte, error) {
	if err := r.checkEncodable(); err != nil {
		return nil, err
	}

	return r.appendJSON(nil), nil
}

// UnmarshalJSON reads an object with two required date strings, start and end.
// JSON null preserves r. A malformed or incomplete object returns an error
// before assigning to r.
func (r *DateRange) UnmarshalJSON(data []byte) error {
	var bounds *dateRangeJSON
	if err := json.Unmarshal(data, &bounds); err != nil {
		return fmt.Errorf("invalid date range: %w", err)
	}

	if bounds == nil { // JSON null preserves the receiver.
		return nil
	}

	if bounds.Start == nil {
		return errors.New("invalid date range: start is required")
	}

	if bounds.End == nil {
		return errors.New("invalid date range: end is required")
	}

	parsed, err := NewDateRange(*bounds.Start, *bounds.End)
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

func (r DateRange) checkEncodable() error {
	// The endpoints are ordered, so these bounds cover both years.
	if r.start.Year() < 0 || r.end.Year() > 9999 {
		return errors.New("cannot marshal date range: endpoint years must be in 0000..9999")
	}

	return nil
}

// appendTo uses the endpoint formatters directly so AppendText can reuse its buffer.
func (r DateRange) appendTo(b []byte) []byte {
	b = slices.Grow(b, dateRangeTextLen)
	b = append(b, '[')
	b = r.start.appendTo(b)
	b = append(b, ',')
	b = r.end.appendTo(b)

	return append(b, ')')
}

// appendJSON can quote each date directly because its text contains only digits and hyphens.
func (r DateRange) appendJSON(b []byte) []byte {
	b = slices.Grow(b, 2*dateLen+len(`{"start":"","end":""}`))
	b = append(b, `{"start":"`...)
	b = r.start.appendTo(b)
	b = append(b, `","end":"`...)
	b = r.end.appendTo(b)

	return append(b, '"', '}')
}

type dateRangeJSON struct {
	// Start is the inclusive start. Nil detects an absent or null JSON field.
	Start *Date `json:"start"`

	// End is the exclusive end. Nil detects an absent or null JSON field.
	End *Date `json:"end"`
}
