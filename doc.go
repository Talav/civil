// Package civil provides calendar dates, date ranges, times of day and
// datetimes, all without a time zone.
//
// A Date names a day on the calendar, such as a trading day or an expiry. A
// Time names a reading of a wall clock, such as a session close. A DateTime
// holds one of each. None is an instant: the same Date and Time happen at
// different moments in different time zones. Date.At and DateTime.In join
// them with a location into an instant, and NewDateFromTime, NewTimeFromTime
// and NewDateTimeFromTime split an instant back up.
//
// Dates are counted in the proleptic Gregorian calendar.
package civil
