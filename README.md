# civil

[![tag](https://img.shields.io/github/tag/talav/civil.svg)](https://github.com/talav/civil/tags)
[![Go Reference](https://pkg.go.dev/badge/github.com/talav/civil.svg)](https://pkg.go.dev/github.com/talav/civil)
[![CI](https://github.com/talav/civil/workflows/Civil%20CI/badge.svg)](https://github.com/talav/civil/actions)
[![codecov](https://codecov.io/gh/talav/civil/graph/badge.svg)](https://codecov.io/gh/talav/civil)
[![License](https://img.shields.io/github/license/talav/civil)](./LICENSE)

Calendar dates and times of day for Go, without a time zone, that cannot hold an invalid value.

## Features

- **Three types**: `Date` (a calendar day), `Time` (a time of day) and `DateTime` (both)
- **Valid by construction**: constructors return an error instead of a February 30 or a 24:61
- **No normalizing**: an out-of-range part is rejected, never rolled over into the next month or hour
- **Comparable**: values work with `==` and as map keys
- **Text, JSON and SQL**: `encoding.TextMarshaler`, `encoding.TextAppender`, `driver.Valuer` and `sql.Scanner`
- **Date arithmetic**: add days, months and years; count the days between two dates
- **No dependencies** (except testing)

## Installation

```bash
go get github.com/talav/civil
```

Requires Go 1.27 or later.

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/talav/civil"
)

func main() {
    d, err := civil.NewDate(2026, time.March, 9)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(d)            // 2026-03-09
    fmt.Println(d.Weekday())  // Monday
    fmt.Println(d.AddDays(5)) // 2026-03-14

    _, err = civil.NewDate(2026, time.February, 30)
    fmt.Println(err) // day 30 must be in 1..28 for February 2026
}
```

## Why

A `time.Time` is an instant. A birthday, an expiry date or a shop's opening hour is not: it is what a calendar or a wall clock shows, and it happens at different instants in different places. Storing one in a `time.Time` forces a time zone and a time of day onto it that mean nothing, and they come back as off-by-one-day bugs.

The types here hold only what is known. They turn into an instant when, and only when, a location is supplied.

## Usage

### Date

```go
d, err := civil.NewDate(2026, time.March, 9)
d, err = civil.NewDateFromString("2026-03-09")
d = civil.NewDateFromTime(time.Now())

d.Year()    // 2026
d.Month()   // time.March
d.Day()     // 9
d.Weekday() // time.Monday

d.AddDays(30)
d.AddMonths(-1)
d.AddYears(1)

expiry, _ := civil.NewDate(2026, time.March, 20)
expiry.DaysSince(d) // 11
d.Before(expiry)    // true
d.Compare(expiry)   // -1
```

`AddMonths` and `AddYears` keep the day of the month. When the target month is too short, the surplus days run on into the next month, as `time.Time.AddDate` does: January 31 plus one month is March 3, or March 2 in a leap year.

Dates use the proleptic Gregorian calendar and may be in any year, including year 0 and negative years.

### Time

```go
t, err := civil.NewTime(9, 30, 0, 0) // hour, minute, second, nanosecond
t, err = civil.NewTimeFromString("09:30")
t = civil.NewTimeFromTime(time.Now())

t.Hour()       // 9
t.Minute()     // 30
t.Second()     // 0
t.Nanosecond() // 0

closing, _ := civil.NewTime(16, 0, 0, 0)
t.Before(closing) // true
```

### DateTime

```go
dt, err := civil.NewDateTime(d, t)
dt, err = civil.NewDateTimeFromString("2026-03-09T09:30:00")
dt = civil.NewDateTimeFromTime(time.Now())

dt.Date() // 2026-03-09
dt.Time() // 09:30:00
```

### Turning a civil value into an instant

Supply a location:

```go
newYork, _ := time.LoadLocation("America/New_York")

d.In(newYork)    // midnight of that day in New York
d.At(t, newYork) // 09:30 on that day in New York
dt.In(newYork)   // the same, from a DateTime
```

The `New*FromTime` constructors go the other way and read the date and the time of day in the `time.Time`'s own location.

A wall time that a zone skips or repeats on that day (a daylight saving change) has no single instant. These methods return whatever `time.Date` picks.

### JSON

All three types encode as strings and work as map keys:

```go
type Session struct {
    Day   civil.Date `json:"day"`
    Open  civil.Time `json:"open"`
    Close civil.Time `json:"close"`
}

// {"day":"2026-03-09","open":"09:30:00","close":"16:00:00"}
```

Text that is not a valid value fails to decode and leaves the target unchanged.

### SQL

All three types implement `driver.Valuer` and `sql.Scanner`, so they work as query arguments and scan targets. `Scan` accepts a `time.Time`, a `string` or a `[]byte`.

```go
var day civil.Date
err := db.QueryRow(`SELECT trade_date FROM trades WHERE id = $1`, id).Scan(&day)
```

`NULL` is an error. Scan a nullable column into a `sql.Null`:

```go
var settled sql.Null[civil.Date]
```

## Text Formats

| Type | Written as | Also read |
|------|------------|-----------|
| `Date` | `2026-03-09` | |
| `Time` | `09:30:00`, or `09:30:00.250000000` when the nanosecond is not zero | `09:30`, and one to nine fraction digits |
| `DateTime` | `2026-03-09T09:30:00` | a space in place of the `T` |

Parsing is strict: parts must be zero-padded, and no zone suffix, surrounding space or other separator is accepted.

A `Date` in a year outside 0 to 9999 is written with a sign and all its digits (`+10000-01-01`), as in ISO 8601. `NewDateFromString` does not read that form.

## Zero Values

A value is built by a constructor, but Go still lets anyone write `civil.Date{}`. `IsValid` tells such a value apart:

- `civil.Date{}` is not a day. It stands for a missing date: `IsValid` returns false, and other methods are meaningless on it.
- `civil.DateTime{}` has no date, so it is not a datetime either, and `IsValid` returns false. `NewDateTime` rejects the zero `Date`.
- `civil.Time{}` is midnight, a real time of day, so `Time` has no `IsValid`.

## Testing

```bash
go test -race -cover ./...
```

Date arithmetic is checked against the standard `time` package for every day of years -400 through 2800.

## License

MIT License. See [LICENSE](./LICENSE).
