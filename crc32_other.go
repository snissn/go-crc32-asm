//go:build !arm64 && !amd64

package crc32asm

func useIEEEFallback(crc uint32, n int) bool {
	return true
}

func useCastagnoliFallback(crc uint32, n int) bool {
	return true
}

func checksumIEEE(data []byte) uint32 {
	return checksumIEEEFallback(data)
}

func checksumCastagnoli(data []byte) uint32 {
	return checksumCastagnoliFallback(data)
}

func updateIEEEFast(crc uint32, p []byte) uint32 {
	return updateIEEEFallback(crc, p)
}

func updateCastagnoliFast(crc uint32, p []byte) uint32 {
	return updateCastagnoliFallback(crc, p)
}
