# civil

[![tag](https://img.shields.io/github/tag/talav/civil.svg)](https://github.com/talav/civil/tags)
[![Go Reference](https://pkg.go.dev/badge/github.com/talav/civil.svg)](https://pkg.go.dev/github.com/talav/civil)
[![CI](https://github.com/talav/civil/workflows/Civil%20CI/badge.svg)](https://github.com/talav/civil/actions)
[![codecov](https://codecov.io/gh/talav/civil/graph/badge.svg)](https://codecov.io/gh/talav/civil)
[![License](https://img.shields.io/github/license/talav/civil)](./LICENSE)

Calendar dates, times of day and date ranges for Go. Use them for birthdays, business hours and reporting periods.

## Features

- **No normalizing**: an out-of-range part is rejected, never rolled over into the next month or hour
- **Comparable**: values work with `==` and as map keys
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
june, err := civil.NewDateRange(start, start.AddMonths(1))
if err != nil {
    return err
}
fmt.Println(june.Days()) // 30
```

### Turning a civil value into an instant

```go
newYork, err := time.LoadLocation("America/New_York")
if err != nil {
    return err
}
fmt.Println(start.In(newYork).UTC()) // 2026-06-01 04:00:00 +0000 UTC
```

A wall time that a zone skips or repeats on that day (a daylight saving change) has no single instant. These methods return whatever `time.Date` picks.

### JSON

```json
{
  "day": "2026-06-01",
  "open": "09:30:00",
  "appointment": "2026-06-01T09:30:00",
  "period": {"start": "2026-06-01", "end": "2026-07-01"}
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

- The zero values of `Date`, `DateTime` and `DateRange` represent missing values. Check them with `IsValid()`.
- `civil.Time{}` is midnight, a real time of day, so `Time` has no `IsValid`.

## Testing

```bash
go test -race -cover ./...
```

## License

MIT License. See [LICENSE](./LICENSE).
