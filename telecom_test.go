package randomizer_test

import (
	"strings"
	"testing"

	"github.com/colduction/randomizer-go"
)

var benchTelecom []byte

func luhnValid(s string) bool {
	sum := 0
	double := false
	for i := len(s) - 1; i >= 0; i-- {
		v := int(s[i] - '0')
		if double {
			v *= 2
			if v > 9 {
				v -= 9
			}
		}
		sum += v
		double = !double
	}
	return sum%10 == 0
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func TestTelecomLuhnVector(t *testing.T) {
	got := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{TAC: "49015420"})
	if len(got) != 15 || !strings.HasPrefix(got, "49015420") || !luhnValid(got) {
		t.Fatalf("IMEI with TAC = %q, want 15 Luhn-valid digits", got)
	}
	if !luhnValid("490154203237518") {
		t.Fatal("reference Luhn implementation rejects the GSMA sample")
	}
}

func TestTelecomIMEI(t *testing.T) {
	for range 1000 {
		imei := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{})
		if len(imei) != 15 || !allDigits(imei) || !luhnValid(imei) {
			t.Fatalf("IMEI = %q, want 15 Luhn-valid digits", imei)
		}
		if rbi := imei[:2]; rbi == "00" || rbi == "99" || rbi[0] == '0' && rbi[1] != '1' {
			t.Fatalf("IMEI = %q uses test or reserved Reporting Body Identifier", imei)
		}
		test := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{Test: true})
		if len(test) != 15 || !strings.HasPrefix(test, "00") || !luhnValid(test) {
			t.Fatalf("IMEI test = %q, want 00 range", test)
		}
	}
}

func TestTelecomIMEISV(t *testing.T) {
	for range 1000 {
		sv := randomizer.Telecom.String(randomizer.IMEISV, randomizer.TelecomOptions{})
		if len(sv) != 16 || !allDigits(sv) || sv[14:] == "99" {
			t.Fatalf("IMEISV = %q, want 16 digits with SVN other than 99", sv)
		}
	}
	got := randomizer.Telecom.String(randomizer.IMEISV, randomizer.TelecomOptions{TAC: "35123456", SVN: "07"})
	if len(got) != 16 || !strings.HasPrefix(got, "35123456") || !strings.HasSuffix(got, "07") {
		t.Fatalf("IMEISV with options = %q", got)
	}
}

func TestTelecomICCID(t *testing.T) {
	for range 1000 {
		iccid := randomizer.Telecom.String(randomizer.ICCID, randomizer.TelecomOptions{})
		if len(iccid) != 19 || !strings.HasPrefix(iccid, "89") || !allDigits(iccid) || !luhnValid(iccid) {
			t.Fatalf("ICCID = %q, want 19 Luhn-valid digits starting with 89", iccid)
		}
	}
	for _, length := range []int{18, 19, 20} {
		got := randomizer.Telecom.String(randomizer.ICCID, randomizer.TelecomOptions{IIN: "8901", Length: length})
		if len(got) != length || !strings.HasPrefix(got, "8901") || !luhnValid(got) {
			t.Fatalf("ICCID(IIN 8901, %d) = %q", length, got)
		}
	}
}

func TestTelecomMEID(t *testing.T) {
	for range 1000 {
		meid := randomizer.Telecom.String(randomizer.MEID, randomizer.TelecomOptions{})
		if len(meid) != 14 || strings.ToUpper(meid) != meid {
			t.Fatalf("MEID = %q, want 14 uppercase hex digits", meid)
		}
		for i := 0; i < len(meid); i++ {
			if !strings.ContainsRune("0123456789ABCDEF", rune(meid[i])) {
				t.Fatalf("MEID = %q contains non-hex byte", meid)
			}
		}
		if meid[0] < 'A' {
			t.Fatalf("MEID = %q, want regional code in [A0, FF]", meid)
		}
	}
}

func TestTelecomInvalidOptions(t *testing.T) {
	cases := []struct {
		kind    randomizer.TelecomKind
		options randomizer.TelecomOptions
	}{
		{randomizer.IMEI, randomizer.TelecomOptions{TAC: "12"}},
		{randomizer.IMEI, randomizer.TelecomOptions{TAC: "1234567x"}},
		{randomizer.IMEISV, randomizer.TelecomOptions{SVN: "99"}},
		{randomizer.IMEISV, randomizer.TelecomOptions{SVN: "7"}},
		{randomizer.ICCID, randomizer.TelecomOptions{IIN: "1234"}},
		{randomizer.ICCID, randomizer.TelecomOptions{IIN: "89"}},
		{randomizer.ICCID, randomizer.TelecomOptions{Length: 17}},
		{randomizer.ICCID, randomizer.TelecomOptions{Length: 21}},
	}
	for _, tc := range cases {
		if got := randomizer.Telecom.String(tc.kind, tc.options); got != "" {
			t.Fatalf("String(%d, %+v) = %q, want empty", tc.kind, tc.options, got)
		}
		dst := []byte("id:")
		if got := randomizer.Telecom.Append(dst, tc.kind, tc.options); string(got) != "id:" {
			t.Fatalf("Append(%d, %+v) = %q, want dst unchanged", tc.kind, tc.options, got)
		}
	}
}

func TestTelecomAppendAllocs(t *testing.T) {
	for _, kind := range []randomizer.TelecomKind{randomizer.IMEI, randomizer.IMEISV, randomizer.ICCID, randomizer.MEID} {
		if allocs := testing.AllocsPerRun(1000, func() {
			var buf [32]byte
			out := randomizer.Telecom.Append(buf[:0], kind, randomizer.TelecomOptions{})
			if len(out) == 0 {
				t.Fatal("empty identifier")
			}
		}); allocs != 0 {
			t.Fatalf("Append(%d) with caller buffer allocs/op = %v, want 0", kind, allocs)
		}
	}
}

func BenchmarkTelecomAppendIMEI(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchTelecom = randomizer.Telecom.Append(benchBuffer[:0], randomizer.IMEI, randomizer.TelecomOptions{})
	}
}

func BenchmarkTelecomStringICCID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Telecom.String(randomizer.ICCID, randomizer.TelecomOptions{})
	}
}
