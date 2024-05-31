package main

import (
	"fmt"
	"github.com/linuxboot/fiano/pkg/guid"
	"github.com/linuxboot/fiano/pkg/uefi"
)

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
	//var err error
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
	}

	for i := 0; i < len(f.Sections); i++ {
		switch f.Sections[i].Header.Type {
		case uefi.SectionTypeDXEDepEx:
			exec.Deps = f.Sections[i].DepEx
		case uefi.SectionTypeUserInterface:
			exec.Name = f.Sections[i].Name
		case uefi.SectionTypePE32:
			exec.File = f.Sections[i].Buf()[4:]
		case uefi.SectionTypeVersion:
			exec.Version = f.Sections[i].Version
		case uefi.SectionTypeCompression:
			fmt.Printf("Compressed sections not currently supported: found in file %s\n", exec.Guid)
			//todo: handle compressed sections
		}
	}

	return &exec, nil
}
