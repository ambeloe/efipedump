package main

import (
	"crypto/sha512"
	"fmt"
	"github.com/linuxboot/fiano/pkg/guid"
	"github.com/linuxboot/fiano/pkg/uefi"
	"strings"
)

//var ErrUnknownFVType = errors.New("unknown file volume type")

type PEE struct {
	PECount int

	GUIDMap map[guid.GUID]*[]*uefi.File
}

func (p *PEE) Run(f uefi.Firmware) error {
	var err error

	p.GUIDMap = make(map[guid.GUID]*[]*uefi.File)

	err = f.Apply(p)
	if err != nil {
		return err
	}
	p.dedupFiles()

	return nil
}

func (p *PEE) Visit(f uefi.Firmware) error {
	var uf *uefi.File

	switch f.(type) {
	case *uefi.File:
		uf = f.(*uefi.File)
		switch uefi.NamesToFileType[strings.TrimPrefix(uf.Type, "EFI_FV_FILETYPE_")] {
		case uefi.FVFileTypeApplication, uefi.FVFileTypeDriver:
			//fmt.Println(uf.Header.GUID, uf.Header.Size)
			if p.GUIDMap[uf.Header.GUID] == nil {
				p.GUIDMap[uf.Header.GUID] = &[]*uefi.File{uf}
			} else {
				*p.GUIDMap[uf.Header.GUID] = append(*p.GUIDMap[uf.Header.GUID], uf)
			}

			p.PECount++
			return nil
		}
	}

	return f.ApplyChildren(p)
}

// todo: optimize
func (p *PEE) dedupFiles() {
	dd := make(map[guid.GUID]*[]*uefi.File, len(p.GUIDMap))
	for g, f := range p.GUIDMap {
		if len(*f) > 1 {
			temp := map[[64]byte]int{}
			for i := 0; i < len(*f); i++ {
				temp[sha512.Sum512([]byte(fmt.Sprint((*f)[i])))] = i
			}

			i := 0
			t := make([]*uefi.File, len(temp))
			for _, j := range temp {
				t[i] = (*f)[j]
			}
			dd[g] = &t
		} else {
			dd[g] = f
		}
	}

	p.GUIDMap = dd
}
