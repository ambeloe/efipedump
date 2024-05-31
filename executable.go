package main

import (
	"fmt"
	"github.com/linuxboot/fiano/pkg/guid"
	"github.com/linuxboot/fiano/pkg/uefi"
)

type Executable struct {
	Guid guid.GUID
	Deps []uefi.DepExOp

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

	for i := 0; i < len(f.Sections); i++ {
		switch f.Sections[i].Type {
		case uefi.SectionTypeDXEDepEx.String():
			exec.Deps = f.Sections[i].DepEx
		case uefi.SectionTypeUserInterface.String():
			exec.Name = f.Sections[i].Name
		case uefi.SectionTypePE32.String():
			exec.File = f.Sections[i].Buf()[4:]
		case uefi.SectionTypeVersion.String():
			exec.Version = f.Sections[i].Version
		case uefi.SectionTypeCompression.String():
			fmt.Printf("Compressed sections not currently supported: found in file %s\n", exec.Guid)
			//todo: handle compressed sections
		}
	}

	return &exec, nil
}
