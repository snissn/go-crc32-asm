//go:build go1.25

package crc32asm

import (
	"hash"
	"testing"
)

func cloneHashForTest(t *testing.T, h hash.Hash32, rest []byte, want uint32) {
	t.Helper()
	cloned, err := h.(hash.Cloner).Clone()
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	cloneHash := cloned.(hash.Hash32)
	_, _ = cloneHash.Write(rest)
	if got := cloneHash.Sum32(); got != want {
		t.Fatalf("cloned Sum32 got=%08x want=%08x", got, want)
	}
}
