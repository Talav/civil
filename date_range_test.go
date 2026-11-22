package civil_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talav/civil"
)

func dateRange(t *testing.T, start, end civil.Date) civil.DateRange {
	t.Helper()

	r, err := civil.NewDateRange(start, end)
	require.NoError(t, err)

	return r
}

func TestNewDateRange(t *testing.T) {
	t.Parallel()

	start := date(t, 2024, time.February, 28)
	end := date(t, 2024, time.March, 1)

	tests := []struct {
		// name describes the supplied bounds.
		name string

		// start is the inclusive bound.
		start civil.Date

		// end is the exclusive bound.
		end civil.Date

		// days is the expected length of a valid range.
		days int

		// wantErr is the reason construction fails.
		wantErr string
	}{
		{name: "across leap day", start: start, end: end, days: 2},
		{name: "one day", start: start, end: plusDays(t, start, 1), days: 1},
		{name: "missing start", end: end, wantErr: "start must not be the zero Date"},
		{name: "missing end", start: start, wantErr: "end must not be the zero Date"},
		{name: "both missing", wantErr: "start must not be the zero Date"},
		{name: "equal", start: start, end: start, wantErr: "end must be after start"},
		{name: "reversed", start: end, end: start, wantErr: "end must be after start"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewDateRange(tt.start, tt.end)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Equal(t, civil.DateRange{}, got)
				assert.False(t, got.IsValid())

				return
			}

			require.NoError(t, err)
			assert.True(t, got.IsValid())
			assert.Equal(t, tt.start, got.Start())
			assert.Equal(t, tt.end, got.End())
			assert.Equal(t, tt.days, got.Days())
		})
	}
}

func TestDateRangeContains(t *testing.T) {
	t.Parallel()

	start := date(t, 0, time.December, 31)
	end := date(t, 1, time.January, 2)
	r := dateRange(t, start, end)

	for _, d := range []civil.Date{start, plusDays(t, end, -1)} {
		assert.True(t, r.Contains(d), d)
	}

	for _, d := range []civil.Date{plusDays(t, start, -1), end, plusDays(t, end, 1)} {
		assert.False(t, r.Contains(d), d)
	}
}

func TestDateRangeRelations(t *testing.T) {
	t.Parallel()

	origin := date(t, 0, time.December, 15)
	span := func(start, end int) civil.DateRange {
		return dateRange(t, plusDays(t, origin, start), plusDays(t, origin, end))
	}
	base := span(10, 20)

	tests := []struct {
		// name describes the position of the other range.
		name string

		// other is compared with base.
		other civil.DateRange

		// cmp is the expected order of base relative to other.
		cmp int

		// overlaps is true when the two ranges share a day.
		overlaps bool

		// union is the expected merge result, zero for a gap.
		union civil.DateRange

		// common is the expected intersection, zero when no day is shared.
		common civil.DateRange
	}{
		{name: "same", other: base, overlaps: true, union: base, common: base},
		{name: "starts earlier and overlaps", other: span(5, 15), cmp: 1, overlaps: true, union: span(5, 20), common: span(10, 15)},
		{name: "starts later and overlaps", other: span(15, 25), cmp: -1, overlaps: true, union: span(10, 25), common: span(15, 20)},
		{name: "encloses", other: span(5, 25), cmp: 1, overlaps: true, union: span(5, 25), common: base},
		{name: "inside", other: span(12, 15), cmp: -1, overlaps: true, union: base, common: span(12, 15)},
		{name: "same start shorter", other: span(10, 18), cmp: 1, overlaps: true, union: base, common: span(10, 18)},
		{name: "same start longer", other: span(10, 23), cmp: -1, overlaps: true, union: span(10, 23), common: base},
		{name: "one shared day", other: span(19, 30), cmp: -1, overlaps: true, union: span(10, 30), common: span(19, 20)},
		{name: "adjacent on left", other: span(5, 10), cmp: 1, union: span(5, 20)},
		{name: "adjacent on right", other: span(20, 25), cmp: -1, union: span(10, 25)},
		{name: "gap on left", other: span(5, 9), cmp: 1},
		{name: "gap on right", other: span(21, 25), cmp: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.cmp, base.Compare(tt.other))
			assert.Equal(t, -tt.cmp, tt.other.Compare(base))
			assert.Equal(t, tt.overlaps, base.Overlaps(tt.other))
			assert.Equal(t, tt.overlaps, tt.other.Overlaps(base))

			got, ok := base.Merge(tt.other)
			assert.Equal(t, tt.union.IsValid(), ok)
			assert.Equal(t, tt.union, got)

			got, ok = tt.other.Merge(base)
			assert.Equal(t, tt.union.IsValid(), ok)
			assert.Equal(t, tt.union, got)

			got, ok = base.Intersect(tt.other)
			assert.Equal(t, tt.overlaps, ok)
			assert.Equal(t, tt.common, got)

			got, ok = tt.other.Intersect(base)
			assert.Equal(t, tt.overlaps, ok)
			assert.Equal(t, tt.common, got)
		})
	}
}

func TestDateRangeText(t *testing.T) {
	t.Parallel()

	r := dateRange(t, date(t, 0, time.February, 29), date(t, 0, time.March, 1))
	const want = "[0000-02-29,0000-03-01)"

	assert.Equal(t, want, r.String())

	b, err := r.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, want, string(b))

	b, err = r.AppendText([]byte("range="))
	require.NoError(t, err)
	assert.Equal(t, "range="+want, string(b))

	got, err := civil.NewDateRangeFromString(want)
	require.NoError(t, err)
	assert.Equal(t, r, got)

	var decoded civil.DateRange
	require.NoError(t, decoded.UnmarshalText([]byte(want)))
	assert.Equal(t, r, decoded)
}

func TestNewDateRangeFromStringErrors(t *testing.T) {
	t.Parallel()

	const good = "[0000-02-29,0000-03-01)"
	const formatErr = "must be [YYYY-MM-DD,YYYY-MM-DD)"

	before := dateRange(t, date(t, 0, time.February, 29), date(t, 0, time.March, 1))
	tests := []struct {
		// name identifies the malformed part.
		name string

		// input is passed to both text entry points.
		input string

		// reason is the error following the quoted input.
		reason string
	}{
		{name: "empty", reason: formatErr},
		{name: "empty keyword", input: "empty", reason: formatErr},
		{name: "slash separator", input: "0000-02-29/0000-03-01", reason: formatErr},
		{name: "leading space", input: " " + good, reason: formatErr},
		{name: "trailing space", input: good + " ", reason: formatErr},
		{name: "exclusive start", input: "(0000-02-29,0000-03-01)", reason: formatErr},
		{name: "inclusive end", input: "[0000-02-29,0000-03-01]", reason: formatErr},
		{name: "wrong separator", input: "[0000-02-29;0000-03-01)", reason: formatErr},
		{name: "space after comma", input: "[0000-02-29, 0000-03-01)", reason: formatErr},
		{name: "invalid start", input: "[0000-02-30,0000-03-01)", reason: `start: invalid date "0000-02-30": day 30 must be in 1..29 for February 0`},
		{name: "invalid end", input: "[0000-02-29,0000-02-30)", reason: `end: invalid date "0000-02-30": day 30 must be in 1..29 for February 0`},
		{name: "equal bounds", input: "[0000-02-29,0000-02-29)", reason: "end must be after start"},
		{name: "reversed bounds", input: "[0000-03-01,0000-02-29)", reason: "end must be after start"},
		{name: "expanded year", input: "[9999-12-31,+10000-01-01)", reason: formatErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wantErr := fmt.Sprintf("invalid date range %q: %s", tt.input, tt.reason)
			got, err := civil.NewDateRangeFromString(tt.input)
			require.EqualError(t, err, wantErr)
			assert.Equal(t, civil.DateRange{}, got)

			got = before
			require.EqualError(t, got.UnmarshalText([]byte(tt.input)), wantErr)
			assert.Equal(t, before, got)
		})
	}
}

func TestDateRangeJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		// Window is the range encoded as an object.
		Window civil.DateRange `json:"window"`

		// Optional is omitted for a missing range.
		Optional civil.DateRange `json:"optional,omitzero"`

		// Labels uses the text representation for map keys.
		Labels map[civil.DateRange]string `json:"labels"`
	}

	june := dateRange(t, date(t, 2026, time.June, 1), date(t, 2026, time.July, 1))
	in := payload{Window: june, Labels: map[civil.DateRange]string{june: "June"}}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `{"window":{"start":"2026-06-01","end":"2026-07-01"},"labels":{"[2026-06-01,2026-07-01)":"June"}}`, string(data))

	var out payload
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in, out)

	got := june
	require.NoError(t, json.Unmarshal([]byte(" null "), &got))
	assert.Equal(t, june, got)

	require.NoError(t, json.Unmarshal([]byte(`{"end":"2026-06-03","start":"2026-06-02"}`), &got))
	assert.Equal(t, dateRange(t, plusDays(t, june.Start(), 1), plusDays(t, june.Start(), 2)), got)
}

func TestDateRangeUnmarshalJSON(t *testing.T) {
	t.Parallel()

	before := dateRange(t, date(t, 2026, time.June, 1), date(t, 2026, time.July, 1))
	got := before
	require.NoError(t, got.UnmarshalJSON([]byte(" \tnull\n")))
	assert.Equal(t, before, got)

	tests := []struct {
		// name identifies the input error.
		name string

		// input is the JSON passed to the range.
		input string

		// wantErr is the error reported by the range.
		wantErr string

		// wrongType is the JSON type rejected by the standard decoder.
		wrongType string
	}{
		{name: "empty object", input: `{}`, wantErr: "invalid date range: start is required"},
		{name: "missing end", input: `{"start":"2026-06-01"}`, wantErr: "invalid date range: end is required"},
		{name: "missing start", input: `{"end":"2026-07-01"}`, wantErr: "invalid date range: start is required"},
		{name: "null start", input: `{"start":null,"end":"2026-07-01"}`, wantErr: "invalid date range: start is required"},
		{name: "null end", input: `{"start":"2026-06-01","end":null}`, wantErr: "invalid date range: end is required"},
		{name: "invalid start", input: `{"start":"2026-06-31","end":"2026-07-01"}`, wantErr: `invalid date range: invalid date "2026-06-31": day 31 must be in 1..30 for June 2026`},
		{name: "invalid end", input: `{"start":"2026-06-01","end":"2026-06-31"}`, wantErr: `invalid date range: invalid date "2026-06-31": day 31 must be in 1..30 for June 2026`},
		{name: "equal bounds", input: `{"start":"2026-06-01","end":"2026-06-01"}`, wantErr: "invalid date range: end must be after start"},
		{name: "reversed bounds", input: `{"start":"2026-07-01","end":"2026-06-01"}`, wantErr: "invalid date range: end must be after start"},
		{name: "expanded year", input: `{"start":"9999-12-31","end":"+10000-01-01"}`, wantErr: `invalid date range: invalid date "+10000-01-01": must be YYYY-MM-DD`},
		{name: "string instead of object", input: `"[2026-06-01,2026-07-01)"`, wrongType: "string"},
		{name: "array instead of object", input: `[]`, wrongType: "array"},
		{name: "number instead of date", input: `{"start":12,"end":"2026-07-01"}`, wrongType: "number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := before
			err := got.UnmarshalJSON([]byte(tt.input))
			if tt.wrongType != "" {
				var typeErr *json.UnmarshalTypeError
				require.ErrorAs(t, err, &typeErr)
				assert.Equal(t, tt.wrongType, typeErr.Value)
				assert.ErrorContains(t, err, "invalid date range: ")
			} else {
				require.EqualError(t, err, tt.wantErr)
			}

			assert.Equal(t, before, got)
		})
	}
}

func TestDateRangeMarshalYears(t *testing.T) {
	t.Parallel()

	first := date(t, 0, time.January, 1)
	last := date(t, 9999, time.December, 31)

	r := dateRange(t, first, last)
	b, err := r.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"start":"0000-01-01","end":"9999-12-31"}`, string(b))

	var decoded civil.DateRange
	require.NoError(t, json.Unmarshal(b, &decoded))
	assert.Equal(t, r, decoded)

	tests := []struct {
		// name identifies why the range has no wire format.
		name string

		// r is the range to encode.
		r civil.DateRange

		// display is the printable form, which still works.
		display string

		// wantErr is the reason encoding fails.
		wantErr string
	}{
		{name: "zero value", r: civil.DateRange{}, display: "[0000-00-00,0000-00-00)", wantErr: "cannot marshal the zero DateRange"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.display, tt.r.String())

			b, err := tt.r.AppendText([]byte("range="))
			require.EqualError(t, err, tt.wantErr)
			assert.Equal(t, "range=", string(b))

			_, err = tt.r.MarshalText()
			require.EqualError(t, err, tt.wantErr)
			_, err = tt.r.MarshalJSON()
			require.EqualError(t, err, tt.wantErr)
			_, err = json.Marshal(tt.r)
			require.ErrorContains(t, err, tt.wantErr)
			_, err = json.Marshal(map[civil.DateRange]int{tt.r: 1})
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
