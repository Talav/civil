# civil

[![tag](https://img.shields.io/github/tag/talav/civil.svg)](https://github.com/talav/civil/tags)
[![Go Reference](https://pkg.go.dev/badge/github.com/talav/civil.svg)](https://pkg.go.dev/github.com/talav/civil)
[![CI](https://github.com/talav/civil/workflows/Civil%20CI/badge.svg)](https://github.com/talav/civil/actions)
[![codecov](https://codecov.io/gh/talav/civil/graph/badge.svg)](https://codecov.io/gh/talav/civil)
[![License](https://img.shields.io/github/license/talav/civil)](./LICENSE)

Calendar dates, times of day and date ranges for Go. Use them for birthdays, business hours and reporting periods.

## Features

- **No normalizing**: an out-of-range part is rejected, never rolled over into the next month or hour
- **Comparable**: values work with `==` and as map keys (`LocationDate` uses `Equal` instead)
- **One notion of validity**: `IsValid()` is false only for a missing value
- **Text, JSON and SQL**: `encoding.TextMarshaler`, `encoding.TextAppender`, `driver.Valuer` and `sql.Scanner`

## Installation

Requires Go 1.27 or later and uses only the Go standard library.

```bash
go get github.com/talav/civil
```

## Usage

| Type | Text form |
| --- | --- |
| `Date` | `2026-06-01` |
| `Time` | `09:30:00` |
| `DateTime` | `2026-06-01T09:30:00` |
| `DateRange` | `[2026-06-01,2026-07-01)` |

A `DateRange` includes its start and excludes its end.

```go
start, err := civil.NewDate(2026, time.June, 1)
if err != nil {
    return err
}
end := start.AddMonths(1)
june, err := civil.NewDateRange(start, end)
if err != nil {
    return err
}
fmt.Println(june.Days()) // 30
```

`june.ContainsDate(day)` checks a single date. `june.Contains(period)` checks whether every date in `period` falls in June. It returns true for `[2026-06-30,2026-07-01)` and for `june` itself.

`Merge` joins two overlapping or adjacent ranges, and `Intersect` returns the days two ranges share.

### Turning a civil value into an instant

```go
newYork, err := time.LoadLocation("America/New_York")
if err != nil {
    return err
}
fmt.Println(start.In(newYork).UTC()) // 2026-06-01 04:00:00 +0000 UTC
```

A wall time that a zone skips or repeats on that day (a daylight saving change) has no single instant. These methods return whatever `time.Date` picks.

### LocationDate

A `LocationDate` is a `Date` in a named IANA time zone. `"Local"` is rejected, because it names a different zone on each machine.

```go
day, err := civil.NewLocationDate(start, "America/New_York")
if err != nil {
    return err
}
fmt.Println(day) // 2026-06-01[America/New_York]

open, err := civil.NewTime(9, 30, 0, 0)
if err != nil {
    return err
}
fmt.Println(day.At(open).UTC()) // 2026-06-01 13:30:00 +0000 UTC
```

### JSON

```json
{
  "day": "2026-06-01",
  "open": "09:30:00",
  "appointment": "2026-06-01T09:30:00",
  "period": {"start": "2026-06-01", "end": "2026-07-01"},
  "deadline": {"date": "2026-06-01", "timeZone": "America/New_York"}
}
```

### SQL

`Date`, `Time` and `DateTime` each map to one SQL column. Store a `DateRange` in two `DATE` columns.

```sql
CREATE TABLE reporting_periods (
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    CHECK (start_date < end_date)
);
```

Bind `r.Start()` and `r.End()` on writes. On reads, scan into two `civil.Date` values and call `civil.NewDateRange(start, end)`.

## Zero Values

- The zero values of `Date`, `DateTime`, `DateRange` and `LocationDate` represent missing values. Check them with `IsValid()`.
- Encoding a missing value to text, JSON or SQL is an error. In JSON, tag the field `omitzero` to leave it out.
- `civil.Time{}` is midnight, a real time of day, so `Time` has no `IsValid`.

## Testing

```bash
go test -race -cover ./...
```

## License

MIT License. See [LICENSE](./LICENSE).
