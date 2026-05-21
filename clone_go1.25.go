//go:build go1.25

package crc32asm

import "hash"

func (d *digest) Clone() (hash.Cloner, error) {
	clone := *d
	return &clone, nil
}
