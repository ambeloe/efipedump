package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ambeloe/efipedump/eficompress"
	"github.com/linuxboot/fiano/pkg/guid"
	"github.com/linuxboot/fiano/pkg/uefi"
	"strconv"
	"unsafe"
)

var ErrUnknownCompression = errors.New("section compressed with unknown algorithm")

type Executable struct {
	Guid guid.GUID
	Deps []uefi.DepExOp
	Type string

	Name string
	File []byte

	BuildNumber string
	Version     string
}

func FileToExecutable(f *uefi.File) (*Executable, error) {
	var err error
	var exec = Executable{
		Guid:        f.Header.GUID,
		Name:        "Unknown",
		BuildNumber: "UnknownBuild",
		Version:     "UnknownVersion",
	}

	switch f.Header.Type {
	case uefi.FVFileTypeApplication:
		exec.Type = "APP"
	case uefi.FVFileTypeDriver:
		exec.Type = "DXE"
	case uefi.FVFileTypeSMM:
		exec.Type = "SMM"
	case uefi.FVFileTypePEIM:
		exec.Type = "PEI"

		//need to generate sections since fiano doesn't do it and I don't want to maintain a fork
		for i, offset := 0, f.DataOffset; offset < f.Header.ExtendedSize; i++ {
			s, err := uefi.NewSection(f.Buf()[offset:], i)
			if err != nil {
				return nil, fmt.Errorf("error parsing sections of file %v: %v", f.Header.GUID, err)
			}
			if s.Header.ExtendedSize == 0 {
				return nil, fmt.Errorf("invalid length of section of file %v", f.Header.GUID)
			}
			offset += uint64(s.Header.ExtendedSize)
			// Align to 4 bytes for now. The PI Spec doesn't say what alignment it should be
			// but UEFITool aligns to 4 bytes, and this seems to work on everything I have.
			offset = uefi.Align4(offset)
			f.Sections = append(f.Sections, s)
		}
	}

	err = handleSections(&exec, f.Sections)
	if err != nil {
		return nil, err
	}

	return &exec, nil
}

func handleSections(exec *Executable, s []*uefi.Section) error {
	var err error
	var off int
	var decompressed []byte
	var secs []*uefi.Section

	for i := 0; i < len(s); i++ {
		switch s[i].Header.Type {
		case uefi.SectionTypeDXEDepEx, uefi.SectionTypePEIDepEx:
			exec.Deps = s[i].DepEx
		case uefi.SectionTypeUserInterface:
			exec.Name = s[i].Name
		case uefi.SectionTypePE32:
			off, _, err = sectionInfo(s[i].Buf())
			if err != nil {
				return err
			}
			exec.File = s[i].Buf()[off:]
		case uefi.SectionTypeVersion:
			exec.Version = s[i].Version
			exec.BuildNumber = strconv.Itoa(int(s[i].BuildNumber))
		case uefi.SectionTypeCompression:
			off, _, err = sectionInfo(s[i].Buf())
			if err != nil {
				return err
			}

			switch s[i].Buf()[off+4] {
			case 0:
				decompressed = s[i].Buf()[off+5:]
			case 1:
				decompressed, err = eficompress.DecompressEFI(s[i].Buf()[off+5:], false)
				if err != nil {
					return err
				}
			default:
				return ErrUnknownCompression
			}

			secs, err = bufToSections(decompressed)
			if err != nil {
				return err
			}

			err = handleSections(exec, secs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func bufToSections(buf []byte) ([]*uefi.Section, error) {
	var err error
	var off uint
	var secLen uint32
	var fileOrder int
	var sec *uefi.Section
	var secs = make([]*uefi.Section, 0, 3)

	for off < uint(len(buf)) {
		sec, err = uefi.NewSection(buf[off:], fileOrder)
		if err != nil {
			//os.WriteFile(fmt.Sprintf("wtf_%x_erroffset_%x.bin", len(buf), off), buf, 0644)
			return nil, err
		}

		secs = append(secs, sec)

		fileOrder++
		//fmt.Printf("%X ", off)
		_, secLen, err = sectionInfo(buf[off:])
		if err != nil {
			return nil, err
		}
		off += uint(secLen)
		off += off % 4
		//fmt.Printf("%X + %X -> %X\n", off-uint(secLen), secLen, off)
	}

	return secs, err
}

// returns the length of the header and the length of the whole section from a section buffer
// based off of uefi.NewSection
func sectionInfo(buf []byte) (int, uint32, error) {
	var err error
	var s uefi.SectionExtHeader
	var r = bytes.NewReader(buf)

	err = binary.Read(r, binary.LittleEndian, &s.SectionHeader)
	if err != nil {
		return -1, 0, err
	}

	headerSize := unsafe.Sizeof(uefi.SectionHeader{})
	switch s.Type {
	case uefi.SectionTypeAll, uefi.SectionTypeCompression, uefi.SectionTypeGUIDDefined, uefi.SectionTypeDisposable,
		uefi.SectionTypePE32, uefi.SectionTypePIC, uefi.SectionTypeTE, uefi.SectionTypeDXEDepEx, uefi.SectionTypeVersion,
		uefi.SectionTypeUserInterface, uefi.SectionTypeCompatibility16, uefi.SectionTypeFirmwareVolumeImage,
		uefi.SectionTypeFreeformSubtypeGUID, uefi.SectionTypeRaw, uefi.SectionTypePEIDepEx, uefi.SectionMMDepEx:
		if s.Size == [3]uint8{0xFF, 0xFF, 0xFF} {
			err = binary.Read(r, binary.LittleEndian, &s.ExtendedSize)
			if err != nil {
				return -1, 0, err
			}
			if s.ExtendedSize == 0xFFFFFFFF {
				return -1, 0, errors.New("section size and extended size are all FFs! there should not be free space inside a file")
			}
			headerSize = unsafe.Sizeof(uefi.SectionExtHeader{})
		} else {
			// Copy small size into big for easier handling.
			// uefi.Section's extended size is 32 bits unlike file's
			s.ExtendedSize = uint32(uefi.Read3Size(s.Size))
		}
	default:
		s.ExtendedSize = uint32(uefi.Read3Size(s.Size))
		if bufLen := len(buf); int(s.ExtendedSize) > bufLen {
			s.ExtendedSize = uint32(bufLen)
		}
	}

	return int(headerSize), s.ExtendedSize, nil
}
