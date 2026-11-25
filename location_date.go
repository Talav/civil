package civil

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// localZone names the machine's zone. Its meaning can change across machines.
const localZone = "Local"

// A LocationDate is a calendar day in a time zone.
//
// Date.In takes a Go location. NewLocationDate loads a zone by name.
// JSON records the zone name and uses the receiving machine's zone database
// when decoded.
//
// Compare LocationDates with Equal, not ==, and do not use them as map keys: two
// values for the same day in the same zone can hold different pointers.
//
// A LocationDate with a missing date, including the zero value, is missing.
type LocationDate struct {
	// date is the calendar day, or the zero Date if it is missing.
	date Date

	// location holds the zone whose rules apply to date.
	location *time.Location
}

// NewLocationDate returns the date d in the time zone named zone, an IANA Time
// Zone Database name such as "America/New_York" or "UTC".
//
// It returns an error if d is the zero Date, or if zone is empty, is "Local"
// or time.LoadLocation does not load it.
//
// The zone is loaded with time.LoadLocation, so the machine needs a time zone
// database or the program must import time/tzdata.
func NewLocationDate(d Date, zone string) (LocationDate, error) {
	// The date is checked first, so a missing one costs no read of the time
	// zone database.
	ld := LocationDate{date: d}
	if err := ld.validate(); err != nil {
		return LocationDate{}, err
	}

	// time.LoadLocation reads "" as UTC, which would hide a missing name.
	if zone == "" {
		return LocationDate{}, errors.New("zone must not be empty")
	}

	if zone == localZone {
		return LocationDate{}, fmt.Errorf("zone must not be %q", localZone)
	}

	location, err := time.LoadLocation(zone)
	if err != nil {
		return LocationDate{}, fmt.Errorf("invalid zone %q: %w", zone, err)
	}

	ld.location = location

	return ld, nil
}

// NewLocationDateFromTime returns the calendar date t shows in its own
// location and retains that location.
func NewLocationDateFromTime(t time.Time) LocationDate {
	return NewDateFromTime(t).In(t.Location())
}

// Date returns the calendar day.
func (ld LocationDate) Date() Date { return ld.date }

// Location returns the time zone the day is read in.
func (ld LocationDate) Location() *time.Location { return ld.location }

func (ld LocationDate) IsValid() bool {
	return ld.date.IsValid() && ld.location != nil
}

// At resolves t on ld's date using [time.Date], including its choice for
// skipped or repeated readings. It panics if ld has no location.
func (ld LocationDate) At(t Time) time.Time {
	return ld.date.At(t).In(ld.location)
}

// Equal reports whether ld and ld2 have the same date and zone name.
func (ld LocationDate) Equal(ld2 LocationDate) bool {
	return ld.Compare(ld2) == 0
}

// Compare orders by date, then by zone name.
func (ld LocationDate) Compare(ld2 LocationDate) int {
	return cmp.Or(
		ld.date.Compare(ld2.date),
		cmp.Compare(ld.location.String(), ld2.location.String()),
	)
}

// String returns the date followed by the zone name in brackets:
// 2026-03-09[America/New_York]. The form is for logs and messages; nothing in
// the package reads it back.
func (ld LocationDate) String() string {
	b := ld.date.appendTo(make([]byte, 0, dateLen+len("[America/New_York]")))
	b = append(b, '[')
	b = append(b, ld.location.String()...)

	return string(append(b, ']'))
}

// MarshalJSON returns the location date as an object with the date and the
// zone name: {"date":"2026-03-09","timeZone":"America/New_York"}. It makes
// LocationDate a json.Marshaler.
//
// The zone name comes from [time.Location.String]. A missing date gives an error.
func (ld LocationDate) MarshalJSON() ([]byte, error) {
	if !ld.IsValid() {
		return nil, errors.New("cannot marshal an invalid LocationDate")
	}

	zone := ld.location.String()

	return json.Marshal(locationDateJSON{Date: &ld.date, TimeZone: &zone})
}

// UnmarshalJSON sets ld from an object with a date and a timeZone field, both
// required, as NewLocationDate would build it. On error ld is unchanged. A
// JSON null leaves ld unchanged and is not an error. It makes *LocationDate a
// json.Unmarshaler.
func (ld *LocationDate) UnmarshalJSON(data []byte) error {
	// A pointer target leaves the decoder to recognize null.
	var raw *locationDateJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid location date: %w", err)
	}

	if raw == nil {
		return nil
	}

	if raw.Date == nil {
		return errors.New("invalid location date: date is required")
	}

	if raw.TimeZone == nil {
		return errors.New("invalid location date: timeZone is required")
	}

	parsed, err := NewLocationDate(*raw.Date, *raw.TimeZone)
	if err != nil {
		return fmt.Errorf("invalid location date: %w", err)
	}

	*ld = parsed

	return nil
}

func (ld LocationDate) validate() error {
	if !ld.date.IsValid() {
		return errors.New("date must not be the zero Date")
	}

	return nil
}

// locationDateJSON is the JSON object form of a LocationDate. The fields are
// pointers so that decoding can tell a missing field from an empty one.
type locationDateJSON struct {
	// Date is the calendar day, written as Date writes it.
	Date *Date `json:"date"`

	// TimeZone is the name of the location.
	TimeZone *string `json:"timeZone"`
}
