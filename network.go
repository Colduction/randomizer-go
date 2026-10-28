package randomizer

//go:generate go run ./internal/ouigen -o oui_table.go

// NetworkGenerator produces network addresses and identifiers using a [Generator]'s source.
// Its zero value uses the package fast source. Use [New] to create a generator.
// Concurrent calls must use separate destination buffers.
type NetworkGenerator struct {
	provider provider
}

// Network generates network addresses and identifiers using the package fast source.
var Network NetworkGenerator

func fixedBytes(dst []byte, n int) []byte {
	if cap(dst) < n {
		return make([]byte, n)
	}
	return dst[:n]
}
