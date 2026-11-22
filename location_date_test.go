package civil_test

import (
	"encoding/json"
	"testing"
	"time"

	// The embedded time zone database lets the tests load zones on a machine
	// that has none of its own.
	_ "time/tzdata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talav/civil"
)

// locationDate builds a LocationDate that the test knows to be valid.
func locationDate(t *testing.T, d civil.Date, zone string) civil.LocationDate {
	t.Helper()

	ld, err := civil.NewLocationDate(d, zone)
	require.NoError(t, err)

	return ld
}

func TestNewLocationDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name describes the supplied date and zone.
		name string

		// date is the calendar day.
		date civil.Date

		// zone is the name of the time zone.
		zone string

		// want is the String form of a valid location date.
		want string

		// wantErr is the reason construction fails.
		wantErr string
	}{
		{name: "zone with daylight saving", date: date(t, 2026, time.March, 9), zone: "America/New_York", want: "2026-03-09[America/New_York]"},
		{name: "UTC", date: date(t, 2026, time.March, 9), zone: "UTC", want: "2026-03-09[UTC]"},
		{name: "year before 1000", date: date(t, 33, time.April, 3), zone: "Asia/Jerusalem", want: "0033-04-03[Asia/Jerusalem]"},
		{name: "last day of year 9999", date: date(t, 9999, time.December, 31), zone: "UTC", want: "9999-12-31[UTC]"},
		{name: "alias of a zone", date: date(t, 2026, time.March, 9), zone: "US/Eastern", want: "2026-03-09[US/Eastern]"},
		{name: "zone name with a sign", date: date(t, 2026, time.March, 9), zone: "Etc/GMT+5", want: "2026-03-09[Etc/GMT+5]"},
		{name: "zone name with hyphens", date: date(t, 2026, time.March, 9), zone: "America/Port-au-Prince", want: "2026-03-09[America/Port-au-Prince]"},
		{name: "zone name of three parts", date: date(t, 2026, time.March, 9), zone: "America/Argentina/Buenos_Aires", want: "2026-03-09[America/Argentina/Buenos_Aires]"},
		{name: "zero date", date: civil.Date{}, zone: "America/New_York", wantErr: "date must not be the zero Date"},
		{name: "empty zone", date: date(t, 2026, time.March, 9), zone: "", wantErr: "zone must not be empty"},
		{name: "zone of the machine", date: date(t, 2026, time.March, 9), zone: "Local", wantErr: `zone must not be "Local"`},
		{name: "unknown zone", date: date(t, 2026, time.March, 9), zone: "Mars/Phobos", wantErr: `invalid zone "Mars/Phobos": unknown time zone Mars/Phobos`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := civil.NewLocationDate(tt.date, tt.zone)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.False(t, got.IsValid())
				assert.Nil(t, got.Location())

				return
			}

			require.NoError(t, err)
			assert.True(t, got.IsValid())
			assert.Equal(t, tt.date, got.Date())
			assert.Equal(t, tt.zone, got.Location().String())
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestLocationDateIsValid(t *testing.T) {
	t.Parallel()

	var missing civil.LocationDate

	assert.False(t, missing.IsValid())
	assert.True(t, locationDate(t, date(t, 2026, time.March, 9), "UTC").IsValid())
}

func TestLocationDateAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name describes the zone and the wall time.
		name string

		// day is the location date the clock is read on.
		day civil.LocationDate

		// time is the wall time in the zone of day.
		time civil.Time

		// want is the expected instant.
		want time.Time
	}{
		{name: "midnight in UTC", day: locationDate(t, date(t, 2026, time.March, 9), "UTC"), time: civil.Time{}, want: utc(2026, time.March, 9)},
		{name: "session open in winter time", day: locationDate(t, date(t, 2026, time.March, 6), "America/Chicago"), time: clock(t, 8, 30, 0, 0), want: time.Date(2026, time.March, 6, 14, 30, 0, 0, time.UTC)},
		{name: "session open in summer time", day: locationDate(t, date(t, 2026, time.March, 9), "America/Chicago"), time: clock(t, 8, 30, 0, 0), want: time.Date(2026, time.March, 9, 13, 30, 0, 0, time.UTC)},
		{name: "fraction of a second in a zone ahead", day: locationDate(t, date(t, 2026, time.March, 10), "Asia/Tokyo"), time: clock(t, 5, 15, 30, 250), want: time.Date(2026, time.March, 9, 20, 15, 30, 250, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.day.At(tt.time)

			assert.True(t, tt.want.Equal(got), "got %s", got)
			assert.Equal(t, tt.day.Location(), got.Location())
			assert.Equal(t, tt.day.Date(), dateOf(t, got))
			assert.Equal(t, tt.time, civil.NewTimeFromTime(got))
		})
	}
}

func TestLocationDateEqualAndCompare(t *testing.T) {
	t.Parallel()

	march9 := date(t, 2026, time.March, 9)

	tests := []struct {
		// name describes how the two location dates relate.
		name string

		// a is the receiver.
		a civil.LocationDate

		// b is the argument.
		b civil.LocationDate

		// wantEqual is true when a and b are the same date in the same zone.
		wantEqual bool

		// wantCompare is the expected order of a relative to b.
		wantCompare int
	}{
		{name: "same date and zone, loaded twice", a: locationDate(t, march9, "America/New_York"), b: locationDate(t, march9, "America/New_York"), wantEqual: true, wantCompare: 0},
		{name: "earlier date", a: locationDate(t, date(t, 2026, time.March, 8), "America/New_York"), b: locationDate(t, march9, "America/New_York"), wantCompare: -1},
		{name: "later date", a: locationDate(t, date(t, 2026, time.March, 10), "America/New_York"), b: locationDate(t, march9, "America/New_York"), wantCompare: 1},
		{name: "same date, zone names in order", a: locationDate(t, march9, "Europe/Berlin"), b: locationDate(t, march9, "Europe/Paris"), wantCompare: -1},
		{name: "same date, zone names out of order", a: locationDate(t, march9, "Europe/Paris"), b: locationDate(t, march9, "Europe/Berlin"), wantCompare: 1},
		{name: "two names for the same rules", a: locationDate(t, march9, "America/New_York"), b: locationDate(t, march9, "US/Eastern"), wantCompare: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantEqual, tt.a.Equal(tt.b))
			assert.Equal(t, tt.wantEqual, tt.b.Equal(tt.a))
			assert.Equal(t, tt.wantCompare, tt.a.Compare(tt.b))
			assert.Equal(t, -tt.wantCompare, tt.b.Compare(tt.a))
			assert.Equal(t, tt.a.Equal(tt.b), tt.a.Compare(tt.b) == 0, "Equal and Compare must agree")
		})
	}
}

func TestLocationDateJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string             `json:"name"`
		Day  civil.LocationDate `json:"day"`
	}

	in := payload{Name: "FOMC decision", Day: locationDate(t, date(t, 2026, time.March, 18), "America/New_York")}

	data, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"FOMC decision","day":{"date":"2026-03-18","timeZone":"America/New_York"}}`, string(data))

	var out payload

	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in.Name, out.Name)
	assert.True(t, in.Day.Equal(out.Day), "got %s", out.Day)
}

func TestLocationDateJSONOmitZero(t *testing.T) {
	t.Parallel()

	type payload struct {
		Day civil.LocationDate `json:"day,omitzero"`
	}

	data, err := json.Marshal(payload{})

	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(data))
}

func TestLocationDateMarshalJSONErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name describes why the value has no JSON form.
		name string

		// day is the value to encode.
		day civil.LocationDate

		// display is the printable form, which still works.
		display string

		// wantErr is the reason encoding fails.
		wantErr string
	}{
		{name: "zero value", day: civil.LocationDate{}, display: "0000-00-00[UTC]", wantErr: "cannot marshal the zero LocationDate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.display, tt.day.String())

			_, err := tt.day.MarshalJSON()
			require.EqualError(t, err, tt.wantErr)

			_, err = json.Marshal(tt.day)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestLocationDateUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		// name describes the JSON input.
		name string

		// data is the JSON to decode.
		data string

		// want is the String form of the decoded value.
		want string

		// wantErr is the reason decoding fails.
		wantErr string
	}{
		{name: "date and zone", data: `{"date":"2026-03-09","timeZone":"America/New_York"}`, want: "2026-03-09[America/New_York]"},
		{name: "fields in the other order", data: `{"timeZone":"UTC","date":"2026-03-09"}`, want: "2026-03-09[UTC]"},
		{name: "unknown field", data: `{"date":"2026-03-09","timeZone":"UTC","note":"x"}`, want: "2026-03-09[UTC]"},
		{name: "missing date", data: `{"timeZone":"America/New_York"}`, wantErr: "invalid location date: date is required"},
		{name: "null date", data: `{"date":null,"timeZone":"America/New_York"}`, wantErr: "invalid location date: date is required"},
		{name: "missing zone", data: `{"date":"2026-03-09"}`, wantErr: "invalid location date: timeZone is required"},
		{name: "empty object", data: `{}`, wantErr: "invalid location date: date is required"},
		{name: "first and last years of the text form", data: `{"date":"9999-12-31","timeZone":"UTC"}`, want: "9999-12-31[UTC]"},
		{name: "empty zone", data: `{"date":"2026-03-09","timeZone":""}`, wantErr: "invalid location date: zone must not be empty"},
		{name: "zone of the machine", data: `{"date":"2026-03-09","timeZone":"Local"}`, wantErr: `invalid location date: zone must not be "Local"`},
		{name: "unknown zone", data: `{"date":"2026-03-09","timeZone":"Mars/Phobos"}`, wantErr: `invalid location date: invalid zone "Mars/Phobos": unknown time zone Mars/Phobos`},
		{name: "date that does not exist", data: `{"date":"2026-02-30","timeZone":"UTC"}`, wantErr: `invalid location date: invalid date "2026-02-30": day 30 must be in 1..28 for February 2026`},
		{name: "string in place of the object", data: `"2026-03-09"`, wantErr: "invalid location date: json: cannot unmarshal string into Go value of type civil.locationDateJSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := locationDate(t, date(t, 2001, time.January, 1), "Asia/Tokyo")
			got := before

			err := got.UnmarshalJSON([]byte(tt.data))
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.True(t, before.Equal(got), "a failed decode must leave the value unchanged, got %s", got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestLocationDateUnmarshalJSONNull(t *testing.T) {
	t.Parallel()

	type payload struct {
		Day civil.LocationDate `json:"day"`
	}

	before := locationDate(t, date(t, 2026, time.March, 9), "America/New_York")
	out := payload{Day: before}

	require.NoError(t, json.Unmarshal([]byte(`{"day":null}`), &out))
	assert.True(t, before.Equal(out.Day), "got %s", out.Day)

	// A direct call can pass white space that json.Unmarshal would strip.
	direct := before

	require.NoError(t, direct.UnmarshalJSON([]byte(" null\n")))
	assert.True(t, before.Equal(direct), "got %s", direct)
}
