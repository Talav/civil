package civil

import "strconv"

// appendPadded appends the non-negative n in decimal, padded on the left with
// zeros to at least width digits.
func appendPadded(b []byte, n, width int) []byte {
	limit := 1
	for range width - 1 {
		limit *= 10
	}

	for ; limit > 1 && n < limit; limit /= 10 {
		b = append(b, '0')
	}

	return strconv.AppendInt(b, int64(n), 10)
}

// parseDigits returns the number that s spells in ASCII decimal digits. It
// reports false if s holds any other byte, a sign included.
func parseDigits(s string) (int, bool) {
	n := 0

	for i := range len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}

		n = n*10 + int(c-'0')
	}

	return n, true
}
