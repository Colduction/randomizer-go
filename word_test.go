package randomizer_test

import (
	"bytes"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/colduction/randomizer-go"
)

var noAdjacentWord = randomizer.Word.WithMode(randomizer.NoAdjacentRepeats)

var (
	benchWordString string
	benchWordBytes  []byte
)

func hasAdjacentDuplicate(b []byte) bool {
	if len(b) < 2 {
		return false
	}
	last := b[0]
	for i := 1; i < len(b); i++ {
		if b[i] == last {
			return true
		}
		last = b[i]
	}
	return false
}

func allInAlphabet(b []byte, allow [256]bool) bool {
	for _, c := range b {
		if !allow[c] {
			return false
		}
	}
	return true
}

func makeAlphabet(chars string) [256]bool {
	var allow [256]bool
	for i := 0; i < len(chars); i++ {
		allow[chars[i]] = true
	}
	return allow
}

func TestWordBuiltInAlphabets(t *testing.T) {
	const n = 4096
	cases := []struct {
		alphabet randomizer.Alphabet
		chars    string
	}{
		{randomizer.DecimalAlphabet, "0123456789"},
		{randomizer.HexLowerAlphabet, "0123456789abcdef"},
		{randomizer.HexUpperAlphabet, "0123456789ABCDEF"},
		{randomizer.OctalAlphabet, "01234567"},
		{randomizer.LowerAlphabet, "abcdefghijklmnopqrstuvwxyz"},
		{randomizer.UpperAlphabet, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{randomizer.AlphaAlphabet, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{randomizer.AlphaNumericAlphabet, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{randomizer.Base32Alphabet, "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"},
		{randomizer.Base64URLAlphabet, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"},
	}

	for _, tc := range cases {
		allow := makeAlphabet(tc.chars)

		s := noAdjacentWord.String(tc.alphabet, n)
		if len(s) != n {
			t.Fatalf("String(%v) length = %d, want %d", tc.alphabet, len(s), n)
		}
		sb := []byte(s)
		if !allInAlphabet(sb, allow) {
			t.Fatalf("String(%v) produced invalid character", tc.alphabet)
		}
		if hasAdjacentDuplicate(sb) {
			t.Fatalf("String(%v) produced adjacent duplicate character", tc.alphabet)
		}

		b := noAdjacentWord.Bytes(tc.alphabet, n)
		if len(b) != n {
			t.Fatalf("Bytes(%v) length = %d, want %d", tc.alphabet, len(b), n)
		}
		if !allInAlphabet(b, allow) {
			t.Fatalf("Bytes(%v) produced invalid character", tc.alphabet)
		}
		if hasAdjacentDuplicate(b) {
			t.Fatalf("Bytes(%v) produced adjacent duplicate character", tc.alphabet)
		}

		var buf [n + 3]byte
		dst := append(buf[:0], "id:"...)
		out := noAdjacentWord.Append(dst, tc.alphabet, n)
		if len(out) != n+3 || string(out[:3]) != "id:" || &out[3] != &buf[3] {
			t.Fatalf("Append(%v) did not preserve prefix or reuse caller buffer", tc.alphabet)
		}
		if !allInAlphabet(out[3:], allow) {
			t.Fatalf("Append(%v) produced invalid character", tc.alphabet)
		}
		if hasAdjacentDuplicate(out[3:]) {
			t.Fatalf("Append(%v) produced adjacent duplicate character", tc.alphabet)
		}
	}
}

func TestWordZeroLength(t *testing.T) {
	if got := noAdjacentWord.String(randomizer.DecimalAlphabet, 0); got != "" {
		t.Fatalf("String(DecimalAlphabet,0) = %q, want empty string", got)
	}
	if got := noAdjacentWord.String(randomizer.AlphaNumericAlphabet, 0); got != "" {
		t.Fatalf("String(AlphaNumericAlphabet,0) = %q, want empty string", got)
	}
	if got := noAdjacentWord.String(randomizer.HexLowerAlphabet, 0); got != "" {
		t.Fatalf("String(HexLowerAlphabet,0) = %q, want empty string", got)
	}
	if got := noAdjacentWord.String(randomizer.OctalAlphabet, 0); got != "" {
		t.Fatalf("String(OctalAlphabet,0) = %q, want empty string", got)
	}
	if got := noAdjacentWord.Bytes(randomizer.DecimalAlphabet, 0); got != nil {
		t.Fatalf("Bytes(DecimalAlphabet,0) = %v, want nil", got)
	}
	if got := noAdjacentWord.Bytes(randomizer.AlphaNumericAlphabet, 0); got != nil {
		t.Fatalf("Bytes(AlphaNumericAlphabet,0) = %v, want nil", got)
	}
	if got := noAdjacentWord.Append([]byte("x"), randomizer.AlphaNumericAlphabet, 0); string(got) != "x" {
		t.Fatalf("Append zero length = %q, want %q", got, "x")
	}
	if got := noAdjacentWord.Bytes(randomizer.HexLowerAlphabet, 0); got != nil {
		t.Fatalf("Bytes(HexLowerAlphabet,0) = %v, want nil", got)
	}
	if got := noAdjacentWord.Bytes(randomizer.OctalAlphabet, 0); got != nil {
		t.Fatalf("Bytes(OctalAlphabet,0) = %v, want nil", got)
	}
}

func TestWordDecimalOutput(t *testing.T) {
	const n = 4096
	allow := makeAlphabet("0123456789")

	s := noAdjacentWord.String(randomizer.DecimalAlphabet, n)
	if len(s) != n {
		t.Fatalf("Decimal length = %d, want %d", len(s), n)
	}
	sb := []byte(s)
	if !allInAlphabet(sb, allow) {
		t.Fatal("Decimal produced invalid character")
	}
	if hasAdjacentDuplicate(sb) {
		t.Fatal("Decimal produced adjacent duplicate character")
	}

	b := noAdjacentWord.Bytes(randomizer.DecimalAlphabet, n)
	if len(b) != n {
		t.Fatalf("DecimalBytes length = %d, want %d", len(b), n)
	}
	if !allInAlphabet(b, allow) {
		t.Fatal("DecimalBytes produced invalid character")
	}
	if hasAdjacentDuplicate(b) {
		t.Fatal("DecimalBytes produced adjacent duplicate character")
	}
}

func TestWordHexOutput(t *testing.T) {
	const n = 4096
	allowLower := makeAlphabet("0123456789abcdef")
	allowUpper := makeAlphabet("0123456789ABCDEF")

	sLower := noAdjacentWord.String(randomizer.HexLowerAlphabet, n)
	if len(sLower) != n {
		t.Fatalf("Hex length = %d, want %d", len(sLower), n)
	}
	sbLower := []byte(sLower)
	if !allInAlphabet(sbLower, allowLower) {
		t.Fatal("Hex lower produced invalid character")
	}
	if hasAdjacentDuplicate(sbLower) {
		t.Fatal("Hex lower produced adjacent duplicate character")
	}

	sUpper := noAdjacentWord.String(randomizer.HexUpperAlphabet, n)
	if len(sUpper) != n {
		t.Fatalf("Hex uppercase length = %d, want %d", len(sUpper), n)
	}
	sbUpper := []byte(sUpper)
	if !allInAlphabet(sbUpper, allowUpper) {
		t.Fatal("Hex uppercase produced invalid character")
	}
	if hasAdjacentDuplicate(sbUpper) {
		t.Fatal("Hex uppercase produced adjacent duplicate character")
	}

	bLower := noAdjacentWord.Bytes(randomizer.HexLowerAlphabet, n)
	if len(bLower) != n {
		t.Fatalf("HexBytes lower length = %d, want %d", len(bLower), n)
	}
	if !allInAlphabet(bLower, allowLower) {
		t.Fatal("HexBytes lower produced invalid character")
	}
	if hasAdjacentDuplicate(bLower) {
		t.Fatal("HexBytes lower produced adjacent duplicate character")
	}

	bUpper := noAdjacentWord.Bytes(randomizer.HexUpperAlphabet, n)
	if len(bUpper) != n {
		t.Fatalf("HexBytes upper length = %d, want %d", len(bUpper), n)
	}
	if !allInAlphabet(bUpper, allowUpper) {
		t.Fatal("HexBytes upper produced invalid character")
	}
	if hasAdjacentDuplicate(bUpper) {
		t.Fatal("HexBytes upper produced adjacent duplicate character")
	}
}

func TestWordOctalOutput(t *testing.T) {
	const n = 4096
	allow := makeAlphabet("01234567")

	s := noAdjacentWord.String(randomizer.OctalAlphabet, n)
	if len(s) != n {
		t.Fatalf("Octal length = %d, want %d", len(s), n)
	}
	sb := []byte(s)
	if !allInAlphabet(sb, allow) {
		t.Fatal("Octal produced invalid character")
	}
	if hasAdjacentDuplicate(sb) {
		t.Fatal("Octal produced adjacent duplicate character")
	}

	b := noAdjacentWord.Bytes(randomizer.OctalAlphabet, n)
	if len(b) != n {
		t.Fatalf("OctalBytes length = %d, want %d", len(b), n)
	}
	if !allInAlphabet(b, allow) {
		t.Fatal("OctalBytes produced invalid character")
	}
	if hasAdjacentDuplicate(b) {
		t.Fatal("OctalBytes produced adjacent duplicate character")
	}
}

func TestWordCustomOutput(t *testing.T) {
	const n = 4096
	const dict = "AABCXYZ9"
	allow := makeAlphabet(dict)

	s := noAdjacentWord.Custom(dict, n)
	if len(s) != n {
		t.Fatalf("Custom length = %d, want %d", len(s), n)
	}
	sb := []byte(s)
	if !allInAlphabet(sb, allow) {
		t.Fatal("Custom produced invalid character")
	}
	if hasAdjacentDuplicate(sb) {
		t.Fatal("Custom produced adjacent duplicate character")
	}

	fromBytes := noAdjacentWord.CustomFromBytes([]byte(dict), n)
	if len(fromBytes) != n {
		t.Fatalf("CustomFromBytes length = %d, want %d", len(fromBytes), n)
	}
	fromBytesBytes := []byte(fromBytes)
	if !allInAlphabet(fromBytesBytes, allow) {
		t.Fatal("CustomFromBytes produced invalid character")
	}
	if hasAdjacentDuplicate(fromBytesBytes) {
		t.Fatal("CustomFromBytes produced adjacent duplicate character")
	}

	b := noAdjacentWord.CustomBytes([]byte(dict), n)
	if len(b) != n {
		t.Fatalf("CustomBytes length = %d, want %d", len(b), n)
	}
	if !allInAlphabet(b, allow) {
		t.Fatal("CustomBytes produced invalid character")
	}
	if hasAdjacentDuplicate(b) {
		t.Fatal("CustomBytes produced adjacent duplicate character")
	}

	fromString := noAdjacentWord.CustomBytesFromString(dict, n)
	if len(fromString) != n {
		t.Fatalf("CustomBytesFromString length = %d, want %d", len(fromString), n)
	}
	if !allInAlphabet(fromString, allow) {
		t.Fatal("CustomBytesFromString produced invalid character")
	}
	if hasAdjacentDuplicate(fromString) {
		t.Fatal("CustomBytesFromString produced adjacent duplicate character")
	}
}

func TestWordCustomInvalidOutput(t *testing.T) {
	if got := noAdjacentWord.Custom("AB", 0); got != "" {
		t.Fatalf("Custom zero length = %q, want empty string", got)
	}
	if got := noAdjacentWord.Custom("", 1); got != "" {
		t.Fatalf("Custom empty dictionary = %q, want empty string", got)
	}
	if got := noAdjacentWord.Custom("AA", 2); got != "" {
		t.Fatalf("Custom duplicate-only dictionary = %q, want empty string", got)
	}
	if got := noAdjacentWord.Custom("AA", 1); len(got) != 1 || got[0] != 'A' {
		t.Fatalf("Custom duplicate-only dictionary length 1 = %q, want %q", got, "A")
	}
	if got := noAdjacentWord.CustomBytes([]byte("AB"), 0); got != nil {
		t.Fatalf("CustomBytes zero length = %v, want nil", got)
	}
	if got := noAdjacentWord.CustomBytes(nil, 1); got != nil {
		t.Fatalf("CustomBytes nil dictionary = %v, want nil", got)
	}
	if got := noAdjacentWord.CustomBytes([]byte("AA"), 2); got != nil {
		t.Fatalf("CustomBytes duplicate-only dictionary = %v, want nil", got)
	}
	if got := noAdjacentWord.CustomBytes([]byte("AA"), 1); len(got) != 1 || got[0] != 'A' {
		t.Fatalf("CustomBytes duplicate-only dictionary length 1 = %v, want %v", got, []byte("A"))
	}
}

func TestWordAppendCustom(t *testing.T) {
	const n = 256
	const dict = "ABC123"
	allow := makeAlphabet(dict)

	var buf [n + 4]byte
	dst := append(buf[:0], "key:"...)
	out := noAdjacentWord.AppendCustom(dst, dict, n)
	if len(out) != n+4 || string(out[:4]) != "key:" || &out[4] != &buf[4] {
		t.Fatal("AppendCustom did not preserve prefix or reuse caller buffer")
	}
	if !allInAlphabet(out[4:], allow) {
		t.Fatal("AppendCustom produced invalid character")
	}
	if hasAdjacentDuplicate(out[4:]) {
		t.Fatal("AppendCustom produced adjacent duplicate character")
	}

	fromBytes := noAdjacentWord.AppendCustomFromBytes(dst[:4], []byte(dict), n)
	if len(fromBytes) != n+4 || string(fromBytes[:4]) != "key:" || &fromBytes[4] != &buf[4] {
		t.Fatal("AppendCustomFromBytes did not preserve prefix or reuse caller buffer")
	}
	if !allInAlphabet(fromBytes[4:], allow) {
		t.Fatal("AppendCustomFromBytes produced invalid character")
	}
	if hasAdjacentDuplicate(fromBytes[4:]) {
		t.Fatal("AppendCustomFromBytes produced adjacent duplicate character")
	}

	invalid := noAdjacentWord.AppendCustom([]byte("x"), "AA", 2)
	if string(invalid) != "x" {
		t.Fatalf("AppendCustom invalid dictionary = %q, want %q", invalid, "x")
	}
}

func BenchmarkWordDecimal(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.String(randomizer.DecimalAlphabet, n)
	}
}

func BenchmarkWordDecimalBytes(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordBytes = randomizer.Word.Bytes(randomizer.DecimalAlphabet, n)
	}
}

func BenchmarkWordHex(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.String(randomizer.HexLowerAlphabet, n)
	}
}

func BenchmarkWordHexBytes(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordBytes = randomizer.Word.Bytes(randomizer.HexLowerAlphabet, n)
	}
}

func BenchmarkWordOctal(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.String(randomizer.OctalAlphabet, n)
	}
}

func BenchmarkWordOctalBytes(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordBytes = randomizer.Word.Bytes(randomizer.OctalAlphabet, n)
	}
}

func BenchmarkWordAlphaNumeric(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.String(randomizer.AlphaNumericAlphabet, n)
	}
}

func BenchmarkWordAppendAlphaNumeric(b *testing.B) {
	const n = 256
	buf := make([]byte, 0, n)
	b.ReportAllocs()
	for b.Loop() {
		benchWordBytes = randomizer.Word.Append(buf[:0], randomizer.AlphaNumericAlphabet, n)
	}
}

func BenchmarkWordBase64URL(b *testing.B) {
	const n = 256
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.String(randomizer.Base64URLAlphabet, n)
	}
}

func BenchmarkWordLower(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.String(randomizer.LowerAlphabet, 256)
	}
}

func BenchmarkWordAlphaNumeric16(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.String(randomizer.AlphaNumericAlphabet, 16)
	}
}

func BenchmarkWordCustom(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = randomizer.Word.Custom("AABCXYZ9", 256)
	}
}

func BenchmarkWordAppendAlphaNumericParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var buf [256]byte
		for pb.Next() {
			_ = randomizer.Word.Append(buf[:0], randomizer.AlphaNumericAlphabet, 256)
		}
	})
}

func TestWordSmallDictionaries(t *testing.T) {
	if got := noAdjacentWord.Custom("A", 1); got != "A" {
		t.Fatalf("Custom(A, 1) = %q, want A", got)
	}
	ab := noAdjacentWord.Custom("AB", 64)
	if len(ab) != 64 || hasAdjacentDuplicate([]byte(ab)) || !strings.Contains(ab, "A") || !strings.Contains(ab, "B") {
		t.Fatalf("Custom(AB, 64) = %q, want alternating A and B", ab)
	}
	for range 1000 {
		if got := noAdjacentWord.Custom("AAB", 2); got == "AA" || len(got) != 2 {
			t.Fatalf("Custom(AAB, 2) = %q", got)
		}
	}
	abc := noAdjacentWord.Custom("ABC", 4096)
	if hasAdjacentDuplicate([]byte(abc)) || !allInAlphabet([]byte(abc), makeAlphabet("ABC")) {
		t.Fatal("Custom(ABC) produced invalid output")
	}
}

func TestWordWeightedDictionaryNoAdjacent(t *testing.T) {
	const n = 100000
	out := noAdjacentWord.CustomBytes([]byte("AABC"), n)
	if len(out) != n || hasAdjacentDuplicate(out) || !allInAlphabet(out, makeAlphabet("ABC")) {
		t.Fatal("CustomBytes(AABC) produced invalid output")
	}
	var counts [256]int
	for _, c := range out {
		counts[c]++
	}
	within := func(count, want int) bool { return count > want*97/100 && count < want*103/100 }
	if !within(counts['A'], n*4/10) || !within(counts['B'], n*3/10) || !within(counts['C'], n*3/10) {
		t.Fatalf("CustomBytes(AABC) counts A=%d B=%d C=%d, want 40%%/30%%/30%%", counts['A'], counts['B'], counts['C'])
	}
}

func TestWordLargeDictionary(t *testing.T) {
	dict := make([]byte, 300)
	for i := range dict {
		dict[i] = byte(i)
	}
	out := noAdjacentWord.CustomBytes(dict, 4096)
	if len(out) != 4096 || hasAdjacentDuplicate(out) {
		t.Fatal("CustomBytes(300-byte dictionary) produced invalid output")
	}
}

func TestWordUniformity(t *testing.T) {
	const n = 200000
	cases := []struct {
		alphabet randomizer.Alphabet
		chars    string
	}{
		{randomizer.DecimalAlphabet, "0123456789"},
		{randomizer.LowerAlphabet, "abcdefghijklmnopqrstuvwxyz"},
		{randomizer.AlphaAlphabet, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{randomizer.AlphaNumericAlphabet, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{randomizer.HexLowerAlphabet, "0123456789abcdef"},
	}
	for _, tc := range cases {
		var counts [256]int
		for _, c := range noAdjacentWord.Bytes(tc.alphabet, n) {
			counts[c]++
		}
		mean := n / len(tc.chars)
		for i := 0; i < len(tc.chars); i++ {
			if c := counts[tc.chars[i]]; c < mean*9/10 || c > mean*11/10 {
				t.Fatalf("alphabet %v symbol %q count %d, want within 10%% of %d", tc.alphabet, tc.chars[i], c, mean)
			}
		}
	}
}

func TestWordFixedProviderLanes(t *testing.T) {
	w := randomizer.FromSource(fixedProvider(^uint64(0))).Word.WithMode(randomizer.NoAdjacentRepeats)
	if got := w.String(randomizer.DecimalAlphabet, 6); got != "989898" {
		t.Fatalf("Decimal with all-one lanes = %q, want 989898", got)
	}
	if got := w.Custom("xyz", 5); got != "zyzyz" {
		t.Fatalf("Custom(xyz) with all-one lanes = %q, want zyzyz", got)
	}
}

func TestWordIndependentDefault(t *testing.T) {
	w := randomizer.FromSource(fixedProvider(^uint64(0))).Word
	for _, tc := range []struct {
		alphabet randomizer.Alphabet
		want     string
	}{
		{randomizer.DecimalAlphabet, "999999"},
		{randomizer.HexLowerAlphabet, "ffffff"},
		{randomizer.Base64URLAlphabet, "______"},
		{randomizer.Alphabet(255), "999999"},
	} {
		if got := w.String(tc.alphabet, 6); got != tc.want {
			t.Fatalf("String(%v) = %q, want %q", tc.alphabet, got, tc.want)
		}
	}
	if got := w.Custom("AA", 16); got != strings.Repeat("A", 16) {
		t.Fatalf("Custom(AA) = %q", got)
	}
	if got := w.Custom("xyz", 5); got != "zzzzz" {
		t.Fatalf("Custom(xyz) = %q, want zzzzz", got)
	}
	if got := w.WithMode(randomizer.NoAdjacentRepeats).WithMode(randomizer.WordMode(255)).Custom("AA", 2); got != "AA" {
		t.Fatalf("unknown mode = %q, want AA", got)
	}
	if got := w.WithMode(randomizer.NoAdjacentRepeats).Custom("AA", 2); got != "" {
		t.Fatalf("single-symbol no-repeat dictionary = %q, want empty", got)
	}
	if got := w.Custom("AA", 2); got != "AA" {
		t.Fatalf("WithMode changed original generator: %q", got)
	}
}

func TestWordOverlappingDictionary(t *testing.T) {
	for _, mode := range []randomizer.WordMode{randomizer.IndependentSymbols, randomizer.NoAdjacentRepeats} {
		for _, dictionary := range []string{"0123456789abcdef", "AABC", strings.Repeat("A", 511) + "B"} {
			seed := [32]byte{1, 2, 3}
			first := randomizer.FromSource(rand.NewChaCha8(seed)).Word.WithMode(mode)
			second := randomizer.FromSource(rand.NewChaCha8(seed)).Word.WithMode(mode)
			storage := make([]byte, len(dictionary)+300)
			copy(storage, dictionary)
			got := first.AppendCustomFromBytes(storage[:0], storage[:len(dictionary)], 257)
			want := second.CustomBytesFromString(dictionary, 257)
			if !bytes.Equal(got, want) {
				t.Fatalf("overlapping dictionary mode=%v size=%d changed sampling", mode, len(dictionary))
			}
		}
	}
}

func TestWordDictionaryOwnership(t *testing.T) {
	w := randomizer.FromSource(fixedProvider(^uint64(0))).Word
	dictionary := []byte("ABCD")
	got := w.CustomFromBytes(dictionary, 32)
	for i := range dictionary {
		dictionary[i] = 'X'
	}
	if got != strings.Repeat("D", 32) {
		t.Fatalf("dictionary mutation changed string: %q", got)
	}
	first := w.CustomBytes([]byte("ABCD"), 8)
	second := w.CustomBytes([]byte("ABCD"), 8)
	first[0] = 'X'
	if second[0] != 'D' {
		t.Fatal("byte results share storage")
	}
}

func TestWordWeightedIndependent(t *testing.T) {
	w := randomizer.FromSource(rand.NewPCG(7, 11)).Word
	out := w.CustomBytes([]byte("AABC"), 100000)
	var counts [256]int
	for _, c := range out {
		counts[c]++
	}
	if counts['A'] < 49000 || counts['A'] > 51000 || counts['B'] < 24000 || counts['B'] > 26000 || counts['C'] < 24000 || counts['C'] > 26000 {
		t.Fatalf("weighted IID counts: A=%d B=%d C=%d", counts['A'], counts['B'], counts['C'])
	}
	if !hasAdjacentDuplicate(out) {
		t.Fatal("independent output has no repeated adjacent byte")
	}
}

func TestWordSharedGenerator(t *testing.T) {
	w := randomizer.FromSource(rand.NewChaCha8([32]byte{42})).Word
	var group sync.WaitGroup
	for range 16 {
		group.Go(func() {
			for range 64 {
				var storage [64]byte
				out := w.AppendCustomFromBytes(storage[:0], []byte("AABC"), len(storage))
				if len(out) != len(storage) || !allInAlphabet(out, makeAlphabet("ABC")) {
					t.Error("shared word generator produced invalid output")
				}
			}
		})
	}
	group.Wait()
}

func TestWordAppendAllocations(t *testing.T) {
	for _, mode := range []randomizer.WordMode{randomizer.IndependentSymbols, randomizer.NoAdjacentRepeats} {
		w := randomizer.Word.WithMode(mode)
		for _, dictionary := range []string{"0123456789", "AABC", strings.Repeat("A", 300) + "BC"} {
			var storage [256]byte
			if got := testing.AllocsPerRun(100, func() {
				out := w.AppendCustom(storage[:0], dictionary, len(storage))
				if len(out) != len(storage) {
					t.Fatal("invalid append result")
				}
			}); got != 0 {
				t.Fatalf("AppendCustom mode=%v size=%d allocations = %v, want 0", mode, len(dictionary), got)
			}
		}
	}
}

func FuzzWordCustom(f *testing.F) {
	for _, dictionary := range [][]byte{nil, []byte("A"), []byte("AABC"), {0, 1, 2, 255}, []byte(strings.Repeat("A", 300) + "BC")} {
		f.Add(dictionary, uint16(64), false)
		f.Add(dictionary, uint16(64), true)
	}
	f.Fuzz(func(t *testing.T, dictionary []byte, requested uint16, noAdjacent bool) {
		if len(dictionary) > 4096 {
			t.Skip()
		}
		length := int(requested % 513)
		w := randomizer.FromSource(rand.NewPCG(1, 2)).Word
		if noAdjacent {
			w = w.WithMode(randomizer.NoAdjacentRepeats)
		}
		var seen [256]bool
		var distinct int
		for _, c := range dictionary {
			if !seen[c] {
				seen[c] = true
				distinct++
			}
		}
		out := w.CustomBytes(dictionary, length)
		if length == 0 || len(dictionary) == 0 || noAdjacent && length > 1 && distinct == 1 {
			if out != nil {
				t.Fatalf("invalid request produced %x", out)
			}
			return
		}
		if len(out) != length || !allInAlphabet(out, seen) || noAdjacent && hasAdjacentDuplicate(out) {
			t.Fatalf("invalid custom output for size=%d length=%d noAdjacent=%v", len(dictionary), length, noAdjacent)
		}
	})
}

func BenchmarkWordModes(b *testing.B) {
	for _, mode := range []randomizer.WordMode{randomizer.IndependentSymbols, randomizer.NoAdjacentRepeats} {
		for _, length := range []int{16, 256, 4096} {
			b.Run(strconv.Itoa(int(mode))+"/"+strconv.Itoa(length), func(b *testing.B) {
				w := randomizer.Word.WithMode(mode)
				storage := make([]byte, 0, length)
				b.SetBytes(int64(length))
				b.ReportAllocs()
				for b.Loop() {
					benchWordBytes = w.Append(storage[:0], randomizer.AlphaNumericAlphabet, length)
				}
			})
		}
	}
}

func BenchmarkWordWeightedNoAdjacentSkew(b *testing.B) {
	dictionary := strings.Repeat("A", 100000) + "BC"
	w := randomizer.Word.WithMode(randomizer.NoAdjacentRepeats)
	var storage [256]byte
	b.ReportAllocs()
	for b.Loop() {
		benchWordBytes = w.AppendCustom(storage[:0], dictionary, len(storage))
	}
}

func BenchmarkWordCustomNoAdjacent(b *testing.B) {
	w := randomizer.Word.WithMode(randomizer.NoAdjacentRepeats)
	b.ReportAllocs()
	for b.Loop() {
		benchWordString = w.Custom("AABCXYZ9", 256)
	}
}
