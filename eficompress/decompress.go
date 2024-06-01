package eficompress

/*
#include "EfiTianoDecompress.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

var ErrGetInfoFail = errors.New("failed to get compressed data info")
var ErrDecompressFail = errors.New("failed to decompress data")

func DecompressEFI(compressed []byte, tiano bool) ([]byte, error) {
	var outSize, scratchSize uint32
	var outBuf, scratchBuf []byte

	if C.EfiTianoGetInfo(
		unsafe.Pointer(&compressed[0]),
		C.UINT32(len(compressed)),
		(*C.UINT32)(&outSize),
		(*C.UINT32)(&scratchSize),
	) != 0 {
		return nil, ErrGetInfoFail
	}

	outBuf = make([]byte, outSize)
	scratchBuf = make([]byte, scratchSize)

	if C.EfiDecompress(
		unsafe.Pointer(&compressed[0]),
		C.UINT32(len(compressed)),
		unsafe.Pointer(&outBuf[0]),
		C.UINT32(outSize), unsafe.Pointer(&scratchBuf[0]),
		C.UINT32(scratchSize),
	) != 0 {
		return nil, ErrDecompressFail
	}

	return outBuf, nil
}
