//go:build !arm64

package crc32asm

func checksumIEEE(data []byte) uint32 {
	return checksumIEEEFallback(data)
}
