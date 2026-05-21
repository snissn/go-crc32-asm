//go:build !go1.25

package crc32asm

import (
	"hash"
	"testing"
)

func cloneHashForTest(t *testing.T, h hash.Hash32, rest []byte, want uint32) {
	t.Helper()
}
