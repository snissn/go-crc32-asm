//go:build arm64

package crc32asm

const fourWayThreshold = 16 << 10

func crc32IEEE4Way(p []byte, chunkLen uintptr) (uint32, uint32, uint32, uint32)

func checksumIEEE(data []byte) uint32 {
	if !hasARM64CRC32() {
		return checksumIEEEFallback(data)
	}
	if len(data) < fourWayThreshold {
		return checksumIEEEFallback(data)
	}

	chunkLen := len(data) / 4
	chunkLen &^= 7
	if chunkLen == 0 {
		return checksumIEEEFallback(data)
	}

	covered := chunkLen * 4
	c0, c1, c2, c3 := crc32IEEE4Way(data[:covered], uintptr(chunkLen))
	crc := combineIEEECached(c0, c1, int64(chunkLen))
	crc = combineIEEECached(crc, c2, int64(chunkLen))
	crc = combineIEEECached(crc, c3, int64(chunkLen))
	if covered != len(data) {
		crc = combineIEEECached(crc, checksumIEEEFallback(data[covered:]), int64(len(data)-covered))
	}
	return crc
}
