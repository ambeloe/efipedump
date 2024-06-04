package main

import (
	_ "embed"
	"flag"
	"fmt"
	"github.com/linuxboot/fiano/pkg/uefi"
	"os"
	"path"
)

//go:embed help.txt
var helpText string

func main() {
	os.Exit(rMain())
}

func printHelp(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message, "\n\n", helpText)
}

func rMain() int {
	var err error
	var file []byte
	var fw uefi.Firmware

	var inFile = flag.String("i", "", "input efi filesystem to open")
	var outDir = flag.String("o", "", "output directory")

	flag.Parse()

	if *inFile == "" {
		printHelp("filename must be specified!")
		return 1
	} else {
		file, err = os.ReadFile(*inFile)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error reading input file:", err)
			return 1
		}
	}

	if *outDir != "" {
		err = os.MkdirAll(*outDir, 0750)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error creating output directory:", err)
			return 1
		}
	}

	fw, err = uefi.Parse(file)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error parsing efi image:", err)
		return 1
	}

	p := new(PEE)

	err = p.Run(fw)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error running parser on firmware:", err)
		return 1
	}

	var exec *Executable
	var total int
	for _, fs := range p.GUIDMap {
		for i, f := range *fs {
			total++
			exec, err = FileToExecutable(f)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "error parsing executable:", err)
				return 1
			}

			filename := path.Join(*outDir, fmt.Sprintf("%s_%s_%s_%s_%d.efi", exec.Name, exec.Type, exec.Guid, exec.Version, i))

			err = os.WriteFile(filename, exec.File, 0644)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error writing file %s: %v\n", filename, err)
				return 1
			}
		}
	}
	fmt.Printf("Extracted %d unique executables; %d total\n", total, p.PECount)

	return 0
}
