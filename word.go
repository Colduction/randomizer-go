package randomizer

import "unsafe"

const (
	deci     string = "0123456789"
	octi     string = "01234567"
	lhexdict string = "0123456789abcdef"
	uhexdict string = "0123456789ABCDEF"
)

type word struct{}

// Word provides random decimal, hexadecimal, and octal strings using the
// active [Provider] selected by [SetProvider].
var Word word

// fillAlphabetNoRepeat fills out with characters from dict ensuring no two adjacent characters are identical.
func fillAlphabetNoRepeat(out []byte, dict string, bits uint8, provider Provider) {
	var (
		raw   uint64
		avail uint8
		last  byte
	)
	if bits > 0 {
		mask := uint64((1 << bits) - 1)
		for i := 0; i < len(out); {
			if avail < bits {
				raw = provider.Sum64()
				avail = 64
			}
			c := dict[raw&mask]
			raw >>= bits
			avail -= bits
			if i > 0 && c == last {
				continue
			}
			out[i] = c
			last = c
			i++
		}
		return
	}
	// Byte-rejection path: discard values >= cutoff to eliminate modulo bias.
	dn := len(dict)
	cutoff := (256 / dn) * dn
	for i := 0; i < len(out); {
		if avail < 8 {
			raw = provider.Sum64()
			avail = 64
		}
		v := int(uint8(raw))
		raw >>= 8
		avail -= 8
		if v >= cutoff {
			continue
		}
		c := dict[v%dn]
		if i > 0 && c == last {
			continue
		}
		out[i] = c
		last = c
		i++
	}
}

// Decimal returns a random decimal string of the given length.
func (word) Decimal(length int) string {
	if length <= 0 {
		return ""
	}
	out := make([]byte, length)
	fillAlphabetNoRepeat(out, deci, 0, currentProvider())
	return unsafe.String(unsafe.SliceData(out), len(out))
}

// DecimalBytes returns a random decimal byte slice of the given length.
func (word) DecimalBytes(length int) []byte {
	if length <= 0 {
		return nil
	}
	out := make([]byte, length)
	fillAlphabetNoRepeat(out, deci, 0, currentProvider())
	return out
}

// Hex returns a random hexadecimal string of the given length.
// If uppercase is true, A-F are used; otherwise a-f.
func (word) Hex(length int, uppercase bool) string {
	if length <= 0 {
		return ""
	}
	dict := lhexdict
	if uppercase {
		dict = uhexdict
	}
	out := make([]byte, length)
	fillAlphabetNoRepeat(out, dict, 4, currentProvider())
	return unsafe.String(unsafe.SliceData(out), len(out))
}

// HexBytes returns a random hexadecimal byte slice of the given length.
// If uppercase is true, A-F are used; otherwise a-f.
func (word) HexBytes(length int, uppercase bool) []byte {
	if length <= 0 {
		return nil
	}
	dict := lhexdict
	if uppercase {
		dict = uhexdict
	}
	out := make([]byte, length)
	fillAlphabetNoRepeat(out, dict, 4, currentProvider())
	return out
}

// Octal returns a random octal string of the given length.
func (word) Octal(length int) string {
	if length <= 0 {
		return ""
	}
	out := make([]byte, length)
	fillAlphabetNoRepeat(out, octi, 3, currentProvider())
	return unsafe.String(unsafe.SliceData(out), len(out))
}

// OctalBytes returns a random octal byte slice of the given length.
func (word) OctalBytes(length int) []byte {
	if length <= 0 {
		return nil
	}
	out := make([]byte, length)
	fillAlphabetNoRepeat(out, octi, 3, currentProvider())
	return out
}
