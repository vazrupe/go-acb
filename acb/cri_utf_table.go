package acb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"

	"github.com/vazrupe/endibuf"
)

// CriUtfTable is '@UTF' bytes structure
type CriUtfTable struct {
	Signature []byte
	Size      uint32

	IsEncrypt bool
	Seek      byte
	Increment byte

	Unknown1          uint16
	RowOffset         uint16
	StringTableOffset uint32
	DataOffset        uint32
	TableNameOffset   uint32
	TableName         string
	NumberOfFields    uint16
	RowSize           uint16
	NumberOfRows      uint32

	buf        *endibuf.Reader
	baseOffset int64

	Rows []map[string]CriField
}

// CriField is column field data
type CriField struct {
	Type   byte
	Name   string
	Value  interface{}
	Offset uint32
	Size   uint32
}

// SignatureCriUtfTable is utfTable signature
var SignatureCriUtfTable = []byte("@UTF")

// ErrNoUtfTableHeader is no UTF Table signature error
var ErrNoUtfTableHeader = errors.New("no utftable header")

// ErrUnknownColumnType is unknown column type error
var ErrUnknownColumnType = errors.New("unknown column type")

// NewCriUtfTable return Table Data
func NewCriUtfTable(base io.ReadSeeker, offset int64) (table *CriUtfTable, err error) {
	r := endibuf.NewReader(base)
	r.Endian = binary.BigEndian

	table = &CriUtfTable{}

	table.IsEncrypt = false
	table.baseOffset = offset

	magicBytes, err := r.ReadBytesFromOffset(offset, 4)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(magicBytes, SignatureCriUtfTable) {
		table.IsEncrypt = true
		table.Seek, table.Increment, err = getDecryptKey(magicBytes)
		if err != nil {
			return nil, err
		}
	}
	table.Signature = SignatureCriUtfTable
	lengthByte, err := r.ReadBytesFromOffset(offset+4, 4)
	if err != nil {
		return nil, err
	}
	if table.IsEncrypt {
		lengthByte = decryptUtfData(table.Seek, table.Increment, 4, lengthByte)
	}
	table.Size = r.Endian.Uint32(lengthByte)
	data, err := r.ReadBytesFromOffset(offset+8, int(table.Size))
	if err != nil {
		return nil, err
	}
	if table.IsEncrypt {
		data = decryptUtfData(table.Seek, table.Increment, 8, data)
	}
	baseHeader := append(table.Signature, lengthByte...)
	data = append(baseHeader, data...)
	tmpR := bytes.NewReader(data)
	sr := io.NewSectionReader(tmpR, 0, tmpR.Size())
	table.buf = endibuf.NewReader(sr)
	table.buf.Endian = r.Endian
	err = table.initializeHeader()
	if err != nil {
		return nil, err
	}
	err = table.initializeSchema()
	if err != nil {
		return nil, err
	}
	return
}

func (tb *CriUtfTable) initializeHeader() (err error) {
	tb.buf.Seek(8, 0)
	tb.Unknown1, err = tb.buf.ReadUint16()
	if err != nil {
		return
	}
	tb.RowOffset, err = tb.buf.ReadUint16()
	if err != nil {
		return
	}
	tb.RowOffset += 8
	tb.StringTableOffset, err = tb.buf.ReadUint32()
	if err != nil {
		return
	}
	tb.StringTableOffset += 8
	tb.DataOffset, err = tb.buf.ReadUint32()
	if err != nil {
		return
	}
	tb.DataOffset += 8
	tb.TableNameOffset, err = tb.buf.ReadUint32()
	if err != nil {
		return
	}
	tb.TableName, err = tb.buf.ReadCStringFromOffset(int64(tb.StringTableOffset + tb.TableNameOffset))
	if err != nil {
		return
	}
	tb.NumberOfFields, err = tb.buf.ReadUint16()
	if err != nil {
		return
	}
	tb.RowSize, err = tb.buf.ReadUint16()
	if err != nil {
		return
	}
	tb.NumberOfRows, err = tb.buf.ReadUint32()
	if err != nil {
		return
	}
	tb.Rows = make([]map[string]CriField, tb.NumberOfRows)
	return
}

const (
	columnStorageMask = 0xF0

	columnStorageConstant  = 0x30
	columnStorageConstant2 = 0x70
	columnStoragePerrow    = 0x50
)

const (
	columnTypeMask byte = 0x0F

	columnTypeString = 0x0A
	columnType8Byte  = 0x06
	columnTypeData   = 0x0B
	columnTypeFloat  = 0x08
	columnType4Byte2 = 0x05
	columnType4Byte  = 0x04
	columnType2Byte2 = 0x03
	columnType2Byte  = 0x02
	columnType1Byte2 = 0x01
	columnType1Byte  = 0x00
)

func (tb *CriUtfTable) initializeSchema() (err error) {
	var schemaOffset int64 = 0x20
	var nameOffset uint32
	var constantOffset int64
	var currentOffset, currentRowBase, currentRowOffset int64
	for i := uint32(0); i < tb.NumberOfRows; i++ {
		tb.Rows[i] = make(map[string]CriField)

		currentOffset = schemaOffset
		currentRowBase = int64(uint32(tb.RowOffset) + (i * uint32(tb.RowSize)))
		currentRowOffset = 0

		for j := uint16(0); j < tb.NumberOfFields; j++ {
			var field CriField
			field.Type, err = tb.buf.ReadByteFromOffset(currentOffset)
			if err != nil {
				return
			}
			nameOffset, err = tb.buf.ReadUint32FromOffset(currentOffset + 1)
			if err != nil {
				return
			}
			field.Name, err = tb.buf.ReadCStringFromOffset(int64(tb.StringTableOffset + nameOffset))
			if err != nil {
				return
			}

			switch columnStorageMask & field.Type {
			case columnStorageConstant, columnStorageConstant2:
				constantOffset = currentOffset + 5

				switch field.Type & columnTypeMask {
				case columnTypeString:
					dataOffset, err := tb.buf.ReadUint32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					field.Value, err = tb.buf.ReadCStringFromOffset(int64(tb.StringTableOffset + dataOffset))
					if err != nil {
						return err
					}
					currentOffset += 4
				case columnType8Byte:
					field.Value, err = tb.buf.ReadUint64FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentOffset += 8
				case columnTypeData:
					dataOffset, err := tb.buf.ReadUint32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					dataSize, err := tb.buf.ReadUint32FromOffset(constantOffset + 4)
					if err != nil {
						return err
					}
					field.Offset = tb.DataOffset + dataOffset
					field.Size = dataSize
					field.Value, err = tb.buf.ReadBytesFromOffset(int64(field.Offset), int(field.Size))
					if err != nil {
						return err
					}
					field.Offset += uint32(tb.baseOffset)
					currentOffset += 8
				case columnTypeFloat:
					field.Value, err = tb.buf.ReadFloat32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentOffset += 4
				case columnType4Byte2:
					field.Value, err = tb.buf.ReadInt32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentOffset += 4
				case columnType4Byte:
					field.Value, err = tb.buf.ReadUint32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentOffset += 4
				case columnType2Byte2:
					field.Value, err = tb.buf.ReadInt16FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentOffset += 2
				case columnType2Byte:
					field.Value, err = tb.buf.ReadUint16FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentOffset += 2
				case columnType1Byte2, columnType1Byte:
					field.Value, err = tb.buf.ReadByteFromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentOffset++
				default:
					return ErrUnknownColumnType
				}
			case columnStoragePerrow:
				constantOffset = currentRowBase + currentRowOffset

				switch field.Type & columnTypeMask {
				case columnTypeString:
					dataOffset, err := tb.buf.ReadUint32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					field.Value, err = tb.buf.ReadCStringFromOffset(int64(tb.StringTableOffset + dataOffset))
					if err != nil {
						return err
					}
					currentRowOffset += 4
				case columnType8Byte:
					field.Value, err = tb.buf.ReadUint64FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentRowOffset += 8
				case columnTypeData:
					dataOffset, err := tb.buf.ReadUint32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					dataSize, err := tb.buf.ReadUint32FromOffset(constantOffset + 4)
					if err != nil {
						return err
					}
					field.Offset = tb.DataOffset + dataOffset
					field.Size = dataSize
					field.Value, err = tb.buf.ReadBytesFromOffset(int64(field.Offset), int(field.Size))
					if err != nil {
						return err
					}
					field.Offset += uint32(tb.baseOffset)
					currentRowOffset += 8
				case columnTypeFloat:
					field.Value, err = tb.buf.ReadFloat32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentRowOffset += 4
				case columnType4Byte2:
					field.Value, err = tb.buf.ReadInt32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentRowOffset += 4
				case columnType4Byte:
					field.Value, err = tb.buf.ReadUint32FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentRowOffset += 4
				case columnType2Byte2:
					field.Value, err = tb.buf.ReadInt16FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentRowOffset += 2
				case columnType2Byte:
					field.Value, err = tb.buf.ReadUint16FromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentRowOffset += 2
				case columnType1Byte2, columnType1Byte:
					field.Value, err = tb.buf.ReadByteFromOffset(constantOffset)
					if err != nil {
						return err
					}
					currentRowOffset++
				default:
					return ErrUnknownColumnType
				}
			}

			tb.Rows[i][field.Name] = field

			currentOffset += 5
		}
	}
	return nil
}

func decryptUtfData(seed, inc byte, step int, data []byte) []byte {
	curXor := seed
	for i := 0; i < step; i++ {
		if i > 0 {
			curXor *= inc
		}
	}
	res := make([]byte, len(data))
	for i := range data {
		if i > 0 {
			curXor *= inc
		}
		res[i] = data[i] ^ curXor
	}
	return res
}

func getDecryptKey(encSig []byte) (seed, increment byte, err error) {
	byteMax := byte(255)
	seed = encSig[0] ^ SignatureCriUtfTable[0]
	for increment = byte(0); increment <= byteMax; increment++ {
		m := seed * increment
		if (encSig[1] ^ m) == SignatureCriUtfTable[1] {
			t := increment

			for j := 2; j < len(SignatureCriUtfTable); j++ {
				m *= t

				if (encSig[j] ^ m) != SignatureCriUtfTable[j] {
					break
				} else if (len(SignatureCriUtfTable) - 1) == j {
					return
				}
			}
		}
	}
	err = ErrNoUtfTableHeader
	return
}
