package main

import (
	"encoding/binary"
	"testing"
)

func TestWindowsResourceRejectsBadICO(t *testing.T) {
	for _, input := range [][]byte{nil, []byte{0, 0, 2, 0, 1, 0}, []byte{0, 0, 1, 0, 0, 0}, []byte{0, 0, 1, 0, 1, 0}} {
		if _, err := makeWindowsResource(input); err == nil {
			t.Fatalf("expected bad ICO rejection: %x", input)
		}
	}
}

func TestWindowsResourceStructure(t *testing.T) {
	ico := make([]byte, 6+16+4)
	binary.LittleEndian.PutUint16(ico[2:4], 1)
	binary.LittleEndian.PutUint16(ico[4:6], 1)
	ico[6], ico[7], ico[10], ico[11] = 16, 16, 1, 0
	binary.LittleEndian.PutUint32(ico[14:18], 4)
	binary.LittleEndian.PutUint32(ico[18:22], 22)
	copy(ico[22:], []byte{1, 2, 3, 4})
	data, err := makeWindowsResource(ico)
	if err != nil { t.Fatal(err) }
	if got := binary.LittleEndian.Uint16(data[:2]); got != 0x8664 { t.Fatalf("machine: %x", got) }
	if got := binary.LittleEndian.Uint16(data[20+32:20+34]); got != 2 { t.Fatalf("relocations: %d", got) }
	root := data[60:]
	if count := binary.LittleEndian.Uint16(root[14:16]); count != 2 { t.Fatalf("root directory entries: %d", count) }
	for _, entryOffset := range []int{20, 28} {
		if value := binary.LittleEndian.Uint32(root[entryOffset:]); value&0x80000000 == 0 || value&0x7fffffff == 0 {
			t.Fatalf("root entry at %d does not point to a subdirectory: %08x", entryOffset, value)
		}
	}
	sym := int(binary.LittleEndian.Uint32(data[8:12]))
	if sym+40 != len(data) { t.Fatalf("symbol offset %d vs length %d", sym, len(data)) }
	if string(data[sym:sym+5]) != ".rsrc" { t.Fatal("missing section symbol") }
	if got := binary.LittleEndian.Uint32(data[len(data)-4:]); got != 4 { t.Fatalf("string table: %d", got) }
}
