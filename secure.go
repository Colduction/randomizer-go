package randomizer

import (
	"crypto/rand"
	"encoding/binary"
	"unsafe"
)

func fillSecureIntegers[T Integers](dst []T) {
	var (
		value T
		buf   [256]byte
	)
	width := int(unsafe.Sizeof(value))
	for len(dst) > 0 {
		count := min(len(dst), len(buf)/width)
		_, _ = rand.Read(buf[:count*width])
		decodeIntegers(dst[:count], buf[:count*width])
		dst = dst[count:]
	}
}

func decodeIntegers[T Integers](dst []T, data []byte) {
	var value T
	switch unsafe.Sizeof(value) {
	case 1:
		for i := range dst {
			dst[i] = T(data[i])
		}
	case 2:
		for i := range dst {
			dst[i] = T(binary.LittleEndian.Uint16(data[i*2:]))
		}
	case 4:
		for i := range dst {
			dst[i] = T(binary.LittleEndian.Uint32(data[i*4:]))
		}
	case 8:
		for i := range dst {
			dst[i] = T(binary.LittleEndian.Uint64(data[i*8:]))
		}
	}
}

func fillSecureFloats[T ~float32 | ~float64](dst []T) {
	var (
		value T
		buf   [256]byte
	)
	width := int(unsafe.Sizeof(value))
	for len(dst) > 0 {
		count := min(len(dst), len(buf)/width)
		_, _ = rand.Read(buf[:count*width])
		decodeFloats(dst[:count], buf[:count*width])
		dst = dst[count:]
	}
}

func decodeFloats[T ~float32 | ~float64](dst []T, data []byte) {
	var value T
	if unsafe.Sizeof(value) == 4 {
		for i := range dst {
			dst[i] = T(float32(binary.LittleEndian.Uint32(data[i*4:])>>8) * inv24)
		}
		return
	}
	for i := range dst {
		dst[i] = T(float64(binary.LittleEndian.Uint64(data[i*8:])>>11) * inv53)
	}
}
