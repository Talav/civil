package civil

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// localZone is the name that time.LoadLocation reads as the zone of the
// machine. It is no zone of its own: it means another zone on another
// machine, so it cannot identify the zone of a LocationDate.
const localZone = "Local"

// A LocationDate is a calendar day in a named time zone.
//
// A LocationDate comes only from a constructor, so its date is a day that
// exists and its zone is one the time zone database knows. The zone name is
// the zone's identity; the rules behind the name come from the time zone
// database of the machine.
//
// Compare LocationDates with Equal, not ==, and do not use them as map keys: two
// values for the same day in the same zone can hold different pointers.
//
// The zero value is the one exception: it has no date and no zone. It stands
// for a missing one, IsValid reports false for it, and other methods are
// meaningless on it.
type LocationDate struct {
	// date is the calendar day. It is the zero Date only in the zero value.
	date Date

	// location holds the rules of the time zone the day is read in, loaded
	// by the name that identifies the zone. It is nil only in the zero value.
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

	ld := LocationDate{date: d, location: location}
	if err := ld.validate(); err != nil {
		return LocationDate{}, err
	}

	return ld, nil
}

// Date returns the calendar day.
func (ld LocationDate) Date() Date { return ld.date }

// Location returns the time zone the day is read in. Its String method gives
// the zone name that NewLocationDate was called with.
func (ld LocationDate) Location() *time.Location { return ld.location }

// IsValid reports whether ld is a day in a zone. It is false only for the zero
// value, which stands for a missing one. Every valid LocationDate can be
// encoded.
func (ld LocationDate) IsValid() bool {
	return ld.date.IsValid()
}

// At returns the instant at which a clock in the zone shows t on the date:
// Date.At for the date and the zone.
//
// A wall time that the zone skips or repeats on that day has no single
// instant. time.Date then picks one, and At returns it unchanged. The zero
// LocationDate has no zone, and At panics on it as time.Date does without a
// location.
func (ld LocationDate) At(t Time) time.Time {
	return ld.date.At(t, ld.location)
}

// Equal reports whether ld and ld2 are the same date in the same zone. Zones
// are the same when their names are.
func (ld LocationDate) Equal(ld2 LocationDate) bool {
	return ld.Compare(ld2) == 0
}

// Compare orders two location dates for sorting by date, then by zone name.
// The result is negative, zero or positive as in Date.Compare.
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
// For the zero LocationDate, which is not valid, it returns an error.
func (ld LocationDate) MarshalJSON() ([]byte, error) {
	if !ld.IsValid() {
		return nil, errors.New("cannot marshal the zero LocationDate")
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

	// TimeZone is the IANA Time Zone Database name of the zone.
	TimeZone *string `json:"timeZone"`
}
