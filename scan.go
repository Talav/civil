package civil

import (
	"encoding"
	"fmt"
	"time"
)

// scan sets dst from a database column value, for the Scan method of a type
// that maps to one column. kind names the type in errors.
//
// Drivers deliver such a column as text in a string or []byte, or as a
// time.Time. The time is written with layout, a layout of the time package
// that gives the text of the type, so that all three forms go through
// UnmarshalText and its one check of what the type can hold. NULL and every
// other type are errors. On error dst is unchanged.
func scan[T any, P interface {
	*T
	encoding.TextUnmarshaler
}](dst P, src any, kind, layout string) error {
	switch value := src.(type) {
	case time.Time:
		return dst.UnmarshalText(value.AppendFormat(nil, layout))
	case string:
		return dst.UnmarshalText([]byte(value))
	case []byte:
		return dst.UnmarshalText(value)
	case nil:
		return fmt.Errorf("invalid %s: NULL", kind)
	default:
		return fmt.Errorf("invalid %s: unsupported type %T", kind, src)
	}
}
