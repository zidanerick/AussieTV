package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// The icon data is generated from AussieTV.ico. The COFF .syso resource is
// consumed by the standard Go linker when producing a Windows/amd64 EXE.
// No windres, compiler, or third-party resource package is required.
const (
	iconResource      = 3
	groupIconResource = 14
	languageEnglishUS = 1033
)

type winResource struct {
	id       uint32
	children []winResource
	data     []byte
}
type reloc struct{ offset uint32 }
type resourceSection struct {
	bytes  []byte
	fixups []reloc
}

func (s *resourceSection) reserve(length int) uint32 {
	for len(s.bytes)%4 != 0 {
		s.bytes = append(s.bytes, 0)
	}
	off := uint32(len(s.bytes))
	s.bytes = append(s.bytes, make([]byte, length)...)
	return off
}
func (s *resourceSection) directory(children []winResource) uint32 {
	off := s.reserve(16 + 8*len(children))
	binary.LittleEndian.PutUint16(s.bytes[off+14:], uint16(len(children)))
	for i, child := range children {
		slot := off + 16 + uint32(i*8)
		binary.LittleEndian.PutUint32(s.bytes[slot:], child.id)
		if len(child.children) > 0 {
			childOffset := s.directory(child.children)
			binary.LittleEndian.PutUint32(s.bytes[slot+4:], childOffset|0x80000000)
		} else {
			entry := s.reserve(16)
			payload := s.reserve(len(child.data))
			copy(s.bytes[payload:], child.data)
			binary.LittleEndian.PutUint32(s.bytes[entry:], payload)
			binary.LittleEndian.PutUint32(s.bytes[entry+4:], uint32(len(child.data)))
			s.fixups = append(s.fixups, reloc{entry})
			binary.LittleEndian.PutUint32(s.bytes[slot+4:], entry)
		}
	}
	return off
}
func parseIcons(ico []byte) ([]winResource, []byte, error) {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[:2]) != 0 || binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		return nil, nil, errors.New("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(ico[4:6]))
	if count < 1 || count > 64 || len(ico) < 6+16*count {
		return nil, nil, errors.New("invalid ICO image count")
	}
	icons := make([]winResource, 0, count)
	grp := make([]byte, 6+14*count)
	binary.LittleEndian.PutUint16(grp[2:], 1)
	binary.LittleEndian.PutUint16(grp[4:], uint16(count))
	for i := 0; i < count; i++ {
		e := ico[6+i*16 : 6+(i+1)*16]
		size := binary.LittleEndian.Uint32(e[8:])
		offset := binary.LittleEndian.Uint32(e[12:])
		if size == 0 || uint64(offset)+uint64(size) > uint64(len(ico)) {
			return nil, nil, fmt.Errorf("invalid ICO image %d offset", i)
		}
		id := uint32(i + 1)
		icons = append(icons, winResource{id: id, children: []winResource{{id: languageEnglishUS, data: ico[offset : offset+size]}}})
		dest := grp[6+i*14 : 6+(i+1)*14]
		copy(dest[:12], e[:12])
		binary.LittleEndian.PutUint16(dest[12:], uint16(id))
	}
	return icons, grp, nil
}
func makeWindowsResource(ico []byte) ([]byte, error) {
	icons, group, err := parseIcons(ico)
	if err != nil {
		return nil, err
	}
	resources := []winResource{
		{id: iconResource, children: icons},
		{id: groupIconResource, children: []winResource{{id: 1, children: []winResource{{id: languageEnglishUS, data: group}}}}},
	}
	sec := &resourceSection{}
	if off := sec.directory(resources); off != 0 {
		return nil, errors.New("nonzero root directory")
	}
	if len(sec.bytes) > int(^uint32(0)) || len(sec.fixups) > 65535 {
		return nil, errors.New("resource too large")
	}

	raw := sec.bytes
	var b bytes.Buffer
	write := func(v any) {
		if err := binary.Write(&b, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	// COFF object header, x86_64, one .rsrc section, two symbols (section + auxiliary).
	const headerSize = 20 + 40
	symbolOffset := uint32(headerSize + len(raw) + len(sec.fixups)*10)
	write(uint16(0x8664))
	write(uint16(1))
	write(uint32(0))
	write(symbolOffset)
	write(uint32(2))
	write(uint16(0))
	write(uint16(0))

	b.Write([]byte{'.', 'r', 's', 'r', 'c', 0, 0, 0})
	write(uint32(0)) // virtual size
	write(uint32(0)) // virtual address
	write(uint32(len(raw)))
	write(uint32(headerSize))
	write(uint32(headerSize + len(raw)))
	write(uint32(0))
	write(uint16(len(sec.fixups)))
	write(uint16(0))
	write(uint32(0x40000040)) // initialized, read-only data
	b.Write(raw)
	for _, rel := range sec.fixups {
		write(rel.offset)
		write(uint32(0))      // section symbol number
		write(uint16(0x0003)) // IMAGE_REL_AMD64_ADDR32NB
	}
	// COFF section symbol and section definition auxiliary symbol.
	b.Write([]byte{'.', 'r', 's', 'r', 'c', 0, 0, 0})
	write(uint32(0))
	write(uint16(1))
	write(uint16(0))
	b.WriteByte(3) // IMAGE_SYM_CLASS_STATIC
	b.WriteByte(1) // one auxiliary symbol
	write(uint32(len(raw)))
	write(uint16(len(sec.fixups)))
	write(uint16(0))
	write(uint32(0))
	write(uint16(0))
	b.WriteByte(0)
	b.WriteByte(0)
	write(uint16(0)) // section aux high number
	write(uint32(4)) // string table: size only
	return b.Bytes(), nil
}
func writeWindowsResource(icoPath, outputPath string) error {
	ico, err := os.ReadFile(icoPath)
	if err != nil {
		return err
	}
	b, err := makeWindowsResource(ico)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, b, 0644)
}
