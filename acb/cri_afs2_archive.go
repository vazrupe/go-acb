package acb

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"

	"github.com/vazrupe/endibuf"
)

var sigatureAfs2Archive = []byte{0x41, 0x46, 0x53, 0x32}

// CriAfs2Archive is Afs2 archive structure
type CriAfs2Archive struct {
	Signature     []byte
	Version       []byte
	FileCount     uint32
	ByteAlignment uint32
	Files         map[uint16]CriAfs2File
}

// CriAfs2File is file data in Afs2 archive
type CriAfs2File struct {
	CueID                 uint16
	FileOffsetRaw         int64
	FileOffsetByteAligned int64
	FileLength            int64
	Data                  []byte
}

// ErrFileCountExceeds is error (file count exceeds max value for uint16)
var ErrFileCountExceeds = errors.New("file count exceeds max value for uint16")

// ErrNoAfs2Header is no afs2 archive header error
var ErrNoAfs2Header = errors.New("no afs2 archive header")

// LoadCriAfs2Archive is Afs2 struce load from readseeker
func LoadCriAfs2Archive(buf io.ReadSeeker, offset int64) (arh *CriAfs2Archive, err error) {
	r := endibuf.NewReader(buf)
	r.Endian = binary.LittleEndian
	arh = &CriAfs2Archive{}

	arh.Signature, err = r.ReadBytesFromOffset(offset, 4)
	if err != nil {
		return nil, err
	}
	arh.Version, err = r.ReadBytesFromOffset(offset+4, 4)
	if err != nil {
		return nil, err
	}
	arh.FileCount, err = r.ReadUint32FromOffset(offset + 8)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(arh.Signature, sigatureAfs2Archive) {
		return nil, ErrNoAfs2Header
	}

	offsetFieldSize := int64(arh.Version[1])
	offsetMask := uint(0)

	for j := uint(0); j < uint(offsetFieldSize); j++ {
		offsetMask |= 0xFF << (j * 8)
	}

	if arh.FileCount > 0xFFFF {
		return nil, ErrFileCountExceeds
	}

	arh.ByteAlignment, err = r.ReadUint32FromOffset(offset + 0xC)
	if err != nil {
		return nil, err
	}

	fileCount := uint16(arh.FileCount)
	previousCueID := uint16(0xFFFF)
	arh.Files = make(map[uint16]CriAfs2File)
	for i := uint16(0); i < fileCount; i++ {
		var dummy CriAfs2File

		dummy.CueID, err = r.ReadUint16FromOffset(offset + (0x10 + (2 * int64(i))))
		if err != nil {
			return nil, err
		}
		fileOffsetRaw, err := r.ReadUint32FromOffset(offset + (0x10 + (int64(arh.FileCount) * 2) + (offsetFieldSize * int64(i))))
		if err != nil {
			return nil, err
		}
		// mask off unneeded info
		fileOffsetRaw &= uint32(offsetMask)

		// add offset
		dummy.FileOffsetRaw = offset + int64(fileOffsetRaw)

		// set file offset to byte alignment
		if (dummy.FileOffsetRaw & int64(arh.ByteAlignment)) != 0 {
			dummy.FileOffsetByteAligned = roundUpToByteAlignment(dummy.FileOffsetRaw, int64(arh.ByteAlignment))
		} else {
			dummy.FileOffsetByteAligned = dummy.FileOffsetRaw
		}

		// set file size
		// last file will use final offset entry
		if uint32(i) == (arh.FileCount - 1) {
			lengthOffset := offset + (0x10 + (int64(arh.FileCount) * 2) + (offsetFieldSize * int64(i)) + offsetFieldSize)
			fileLength, err := r.ReadUint32FromOffset(lengthOffset)
			if err != nil {
				return nil, err
			}
			dummy.FileLength = int64(fileLength) + offset - dummy.FileOffsetByteAligned
		}

		if previousCueID != 0xFFFF {
			criFile := arh.Files[previousCueID]
			fileLength := dummy.FileOffsetRaw - criFile.FileOffsetByteAligned
			criFile.FileLength = fileLength
			arh.Files[previousCueID] = criFile
		}

		arh.Files[dummy.CueID] = dummy
		previousCueID = dummy.CueID
	}

	for id, file := range arh.Files {
		file.Data, err = r.ReadBytesFromOffset(file.FileOffsetByteAligned, int(file.FileLength))
		if err != nil {
			return nil, err
		}
		// remove zeros
		for i, d := range file.Data {
			if d > 0 {
				file.Data = file.Data[i:]
				break
			}
		}
		arh.Files[id] = file
	}

	return
}

func roundUpToByteAlignment(valueToRound, byteAlignment int64) int64 {
	return (valueToRound + byteAlignment - 1) / byteAlignment * byteAlignment
}
