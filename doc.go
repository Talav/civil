// Package civil provides calendar dates, date ranges, times of day and
// datetimes, all without a time zone.
//
// A Date names a day on the calendar, such as a trading day or an expiry. A
// Time names a reading of a wall clock, such as a session close. A DateTime
// holds one of each. Date.At pairs a date and time into a DateTime. Date.In
// pairs a date and time zone into a LocationDate.
//
// LocationDate.At and DateTime.In resolve a wall-clock reading in either
// order using [time.Date], including its choice for skipped or repeated readings.
// NewDateFromTime, NewTimeFromTime and NewDateTimeFromTime read the civil
// fields of an instant in its own location.
//
// IsValid reports whether a date is present. Encoding a missing value is an error.
//
// Dates are counted in the proleptic Gregorian calendar.
package civil
