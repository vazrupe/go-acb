package acb

import (
	"errors"
	"os"
)

type CriAcbFile struct {
	base               *CriUtfTable
	Cue                []CriAcbCueRecord
	CueNameToWaveForms map[string]uint16

	InternalAwb *CriAfs2Archive
	ExternalAwb *CriAfs2Archive
}

func LoadCriAcbFile(path string) (acbFile *CriAcbFile, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	acbFile = &CriAcbFile{}
	acbFile.base, err = NewCriUtfTable(f, 0)
	if err != nil {
		return
	}

	err = acbFile.initializeCueList()
	if err != nil {
		return
	}
	err = acbFile.initializeCueNameToWaveformMap()
	if err != nil {
		return
	}

	internalAwbFile, ok := acbFile.base.Rows[0]["AwbFile"]
	if ok && internalAwbFile.Size > 0 {
		acbFile.InternalAwb, err = LoadCriAfs2Archive(acbFile.base.buf, int64(internalAwbFile.Offset))
		if err != nil {
			return
		}
	}

	streamAwbFileSize, ok := acbFile.base.Rows[0]["StreamAwbAfs2Header"].Value.(uint64)
	if ok && streamAwbFileSize > 0 {
		acbFile.ExternalAwb, err = LoadCriAfs2Archive(acbFile.base.buf, 0)
		if err != nil {
			return
		}
	}
	return
}

var ErrUnexpectedReferenceType = errors.New("unexpected ReferenceType")

func (af *CriAcbFile) initializeCueList() (err error) {
	cueTableField, okCue := af.base.Rows[0]["CueTable"]
	waveformTableField, okWaveform := af.base.Rows[0]["WaveformTable"]
	synthTableField, okSynth := af.base.Rows[0]["SynthTable"]
	if !okCue || !okWaveform || !okSynth {
		return errors.New("Not Load Tables")
	}

	cueTableUtf, err := NewCriUtfTable(af.base.buf, int64(cueTableField.Offset))
	if err != nil {
		return err
	}
	waveformTableUtf, err := NewCriUtfTable(af.base.buf, int64(waveformTableField.Offset))
	if err != nil {
		return err
	}
	synthTableUtf, err := NewCriUtfTable(af.base.buf, int64(synthTableField.Offset))
	if err != nil {
		return err
	}
	var referenceItemsOffset, referenceItemsSize, referenceCorrection uint32 = 0, 0, 0
	af.Cue = make([]CriAcbCueRecord, cueTableUtf.NumberOfRows)
	for i := uint32(0); i < cueTableUtf.NumberOfRows; i++ {
		af.Cue[i].IsWaveformIdentified = false

		af.Cue[i].CueID = cueTableUtf.Rows[i]["CueId"].Value.(uint32)
		af.Cue[i].ReferenceType = cueTableUtf.Rows[i]["ReferenceType"].Value.(byte)
		af.Cue[i].ReferenceIndex = cueTableUtf.Rows[i]["ReferenceIndex"].Value.(uint16)

		switch af.Cue[i].ReferenceType {
		case 2:
			referenceItems := synthTableUtf.Rows[af.Cue[i].ReferenceIndex]["ReferenceItems"]
			referenceItemsOffset = referenceItems.Offset
			referenceItemsSize = referenceItems.Size
			referenceCorrection = referenceItemsSize + 2
		case 3, 8:
			if i == 0 {
				referenceItems := synthTableUtf.Rows[0]["ReferenceItems"]
				referenceItemsOffset = referenceItems.Offset
				referenceItemsSize = referenceItems.Size
				referenceCorrection = referenceItemsSize - 2
			} else {
				referenceCorrection += 2
			}
		default:
			return ErrUnexpectedReferenceType
		}

		if referenceItemsSize != 0 {
			indexOffset := int64(referenceItemsOffset + referenceCorrection)
			af.Cue[i].WaveformIndex, err = af.base.buf.ReadUint16FromOffset(indexOffset)
			if err != nil {
				return
			}

			af.Cue[i].WaveformID = waveformTableUtf.Rows[af.Cue[i].WaveformIndex]["Id"].Value.(uint16)
			af.Cue[i].EncodeType = waveformTableUtf.Rows[af.Cue[i].WaveformIndex]["EncodeType"].Value.(byte)

			isStreaming := waveformTableUtf.Rows[af.Cue[i].WaveformIndex]["Streaming"].Value.(byte)
			af.Cue[i].IsStreaming = isStreaming != 0

			af.Cue[i].IsWaveformIdentified = true
		}
	}
	return nil
}

func (af *CriAcbFile) initializeCueNameToWaveformMap() (err error) {
	cueNameTableField, okCueName := af.base.Rows[0]["CueNameTable"]
	if !okCueName {
		return errors.New("Not Load Tables")
	}

	cueNameTableUtf, err := NewCriUtfTable(af.base.buf, int64(cueNameTableField.Offset))
	if err != nil {
		return err
	}
	af.CueNameToWaveForms = make(map[string]uint16)
	for i := uint32(0); i < cueNameTableUtf.NumberOfRows; i++ {
		cueIndex := cueNameTableUtf.Rows[i]["CueIndex"].Value.(uint16)

		if af.Cue[cueIndex].IsWaveformIdentified {
			cueName := cueNameTableUtf.Rows[i]["CueName"].Value.(string)
			af.Cue[cueIndex].CueName = cueName

			af.CueNameToWaveForms[cueName] = af.Cue[cueIndex].WaveformID
		}
	}
	return nil
}
