package acb

import "fmt"

const (
	waveformEncodeTypeAdx         = 0
	waveformEncodeTypeHca         = 2
	waveformEncodeTypeVag         = 7
	waveformEncodeTypeAtrac3      = 8
	waveformEncodeTypeBcwav       = 9
	waveformEncodeTypeNintendoDsp = 13
)

// CriAcbCueRecord is cue data structure
type CriAcbCueRecord struct {
	CueID          uint32
	ReferenceType  byte
	ReferenceIndex uint16

	IsWaveformIdentified bool
	WaveformIndex        uint16
	WaveformID           uint16
	EncodeType           byte
	IsStreaming          bool

	CueName string
}

// GetFileExtension return file extension (.xxx)
func (cr CriAcbCueRecord) GetFileExtension() string {
	switch cr.EncodeType {
	case waveformEncodeTypeAdx:
		return ".adx"
	case waveformEncodeTypeHca:
		return ".hca"
	case waveformEncodeTypeVag:
		return ".vag"
	case waveformEncodeTypeAtrac3:
		return ".at3"
	case waveformEncodeTypeBcwav:
		return ".bcwav"
	case waveformEncodeTypeNintendoDsp:
		return ".dsp"
	default:
		return fmt.Sprintf(".EncodeType-%d.bin", cr.EncodeType)
	}
}
