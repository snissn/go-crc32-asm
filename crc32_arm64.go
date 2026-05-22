//go:build arm64

package crc32asm

const fourWayThreshold = 16 << 10
const castagnoliFourWayThreshold = 4 << 10
const pmullX12Threshold = 3 * 192

func crc32IEEE4Way(p []byte, chunkLen uintptr) (uint32, uint32, uint32, uint32)
func crc32Castagnoli4Way(p []byte, chunkLen uintptr) (uint32, uint32, uint32, uint32)
func crc32IEEEPmullX12(p []byte, blocks uintptr) uint32
func crc32IEEEPmullX12Eor3(p []byte, blocks uintptr) uint32
func crc32CastagnoliPmullX12(p []byte, blocks uintptr) uint32
func crc32CastagnoliPmullX12Eor3(p []byte, blocks uintptr) uint32

func useIEEEFallback(crc uint32, n int) bool {
	if !hasARM64CRC32() {
		return true
	}
	hasPMULL := hasARM64PMULL()
	if crc == 0 || hasPMULL {
		return !(hasPMULL && n >= pmullX12Threshold) && n < fourWayThreshold
	}
	return n < fourWayThreshold
}

func useCastagnoliFallback(crc uint32, n int) bool {
	if !hasARM64CRC32() {
		return true
	}
	if hasARM64PMULL() {
		return n < pmullX12Threshold
	}
	return n < castagnoliFourWayThreshold
}

func checksumIEEE(data []byte) uint32 {
	if !hasARM64CRC32() {
		return checksumIEEEFallback(data)
	}
	if hasARM64PMULL() && len(data) >= pmullX12Threshold {
		prefixLen := len(data) / 192 * 192
		var crc uint32
		if hasARM64SHA3() {
			crc = crc32IEEEPmullX12Eor3(data[:prefixLen], uintptr(prefixLen/192))
		} else {
			crc = crc32IEEEPmullX12(data[:prefixLen], uintptr(prefixLen/192))
		}
		if prefixLen != len(data) {
			crc = combineIEEECached(crc, checksumIEEEFallback(data[prefixLen:]), int64(len(data)-prefixLen))
		}
		return crc
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

func checksumCastagnoli(data []byte) uint32 {
	if !hasARM64CRC32() {
		return checksumCastagnoliFallback(data)
	}
	if hasARM64PMULL() && len(data) >= pmullX12Threshold {
		prefixLen := len(data) / 192 * 192
		var crc uint32
		if hasARM64SHA3() {
			crc = crc32CastagnoliPmullX12Eor3(data[:prefixLen], uintptr(prefixLen/192))
		} else {
			crc = crc32CastagnoliPmullX12(data[:prefixLen], uintptr(prefixLen/192))
		}
		if prefixLen != len(data) {
			crc = combineCastagnoliCached(crc, checksumCastagnoliFallback(data[prefixLen:]), int64(len(data)-prefixLen))
		}
		return crc
	}
	if len(data) < castagnoliFourWayThreshold {
		return checksumCastagnoliFallback(data)
	}

	chunkLen := len(data) / 4
	chunkLen &^= 7
	if chunkLen == 0 {
		return checksumCastagnoliFallback(data)
	}

	covered := chunkLen * 4
	c0, c1, c2, c3 := crc32Castagnoli4Way(data[:covered], uintptr(chunkLen))
	crc := combineCastagnoliCached(c0, c1, int64(chunkLen))
	crc = combineCastagnoliCached(crc, c2, int64(chunkLen))
	crc = combineCastagnoliCached(crc, c3, int64(chunkLen))
	if covered != len(data) {
		crc = combineCastagnoliCached(crc, checksumCastagnoliFallback(data[covered:]), int64(len(data)-covered))
	}
	return crc
}

func updateIEEEFast(crc uint32, p []byte) uint32 {
	if !hasARM64CRC32() || len(p) < pmullX12Threshold || (len(p) < fourWayThreshold && !hasARM64PMULL()) {
		return updateIEEEFallback(crc, p)
	}
	return combineIEEECached(crc, checksumIEEE(p), int64(len(p)))
}

func updateCastagnoliFast(crc uint32, p []byte) uint32 {
	if !hasARM64CRC32() || len(p) < castagnoliFourWayThreshold || (len(p) < pmullX12Threshold && !hasARM64PMULL()) {
		return updateCastagnoliFallback(crc, p)
	}
	return combineCastagnoliCached(crc, checksumCastagnoli(p), int64(len(p)))
}
