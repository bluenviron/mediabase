package mpegts

import (
	"errors"
	"fmt"

	"github.com/asticode/go-astits"
	"github.com/bluenviron/mediacommon/pkg/codecs/ac3"
	"github.com/bluenviron/mediacommon/pkg/codecs/mpeg4audio"
)

const (
	h264Identifier = 'H'<<24 | 'D'<<16 | 'M'<<8 | 'V'
	h265Identifier = 'H'<<24 | 'E'<<16 | 'V'<<8 | 'C'
	opusIdentifier = 'O'<<24 | 'p'<<16 | 'u'<<8 | 's'
	klvaIdentifier = 'K'<<24 | 'L'<<16 | 'V'<<8 | 'A'
)

var errUnsupportedCodec = errors.New("unsupported codec")

func findMPEG4AudioConfig(dem *astits.Demuxer, pid uint16) (*mpeg4audio.Config, error) {
	for {
		data, err := dem.NextData()
		if err != nil {
			return nil, err
		}

		if data.PES == nil || data.PID != pid {
			continue
		}

		var adtsPkts mpeg4audio.ADTSPackets
		err = adtsPkts.Unmarshal(data.PES.Data)
		if err != nil {
			return nil, fmt.Errorf("unable to decode ADTS: %w", err)
		}

		pkt := adtsPkts[0]
		return &mpeg4audio.Config{
			Type:         pkt.Type,
			SampleRate:   pkt.SampleRate,
			ChannelCount: pkt.ChannelCount,
		}, nil
	}
}

func findAC3Parameters(dem *astits.Demuxer, pid uint16) (int, int, error) {
	for {
		data, err := dem.NextData()
		if err != nil {
			return 0, 0, err
		}

		if data.PES == nil || data.PID != pid {
			continue
		}

		var syncInfo ac3.SyncInfo
		err = syncInfo.Unmarshal(data.PES.Data)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid AC-3 frame: %w", err)
		}

		var bsi ac3.BSI
		err = bsi.Unmarshal(data.PES.Data[5:])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid AC-3 frame: %w", err)
		}

		return syncInfo.SampleRate(), bsi.ChannelCount(), nil
	}
}

func findOpusRegistration(descriptors []*astits.Descriptor) bool {
	for _, sd := range descriptors {
		if sd.Registration != nil {
			if sd.Registration.FormatIdentifier == opusIdentifier {
				return true
			}
		}
	}
	return false
}

func findOpusChannelCount(descriptors []*astits.Descriptor) int {
	for _, sd := range descriptors {
		if sd.Extension != nil && sd.Extension.Tag == 0x80 &&
			sd.Extension.Unknown != nil && len(*sd.Extension.Unknown) >= 1 {
			return int((*sd.Extension.Unknown)[0])
		}
	}
	return 0
}

func findOpusCodec(descriptors []*astits.Descriptor) *CodecOpus {
	if !findOpusRegistration(descriptors) {
		return nil
	}

	channelCount := findOpusChannelCount(descriptors)
	if channelCount <= 0 {
		return nil
	}

	return &CodecOpus{
		ChannelCount: channelCount,
	}

}

func findKLVARegistration(descriptors []*astits.Descriptor) bool {
	for _, sd := range descriptors {
		if sd.Registration != nil {
			if sd.Registration.FormatIdentifier == klvaIdentifier {
				return true
			}
		}
	}
	return false
}

// Track is a MPEG-TS track.
type Track struct {
	PID   uint16
	Codec Codec

	isLeading  bool // Writer-only
	mp3Checked bool // Writer-only
}

func (t *Track) marshal() (*astits.PMTElementaryStream, error) {
	return t.Codec.marshal(t.PID)
}

func (t *Track) unmarshal(dem *astits.Demuxer, es *astits.PMTElementaryStream) error {
	t.PID = es.ElementaryPID

	switch es.StreamType {
	case astits.StreamTypeH265Video:
		t.Codec = &CodecH265{}
		return nil

	case astits.StreamTypeH264Video:
		t.Codec = &CodecH264{}
		return nil

	case astits.StreamTypeMPEG4Video:
		t.Codec = &CodecMPEG4Video{}
		return nil

	case astits.StreamTypeMPEG2Video, astits.StreamTypeMPEG1Video:
		t.Codec = &CodecMPEG1Video{}
		return nil

	case astits.StreamTypeAACAudio:
		conf, err := findMPEG4AudioConfig(dem, es.ElementaryPID)
		if err != nil {
			return err
		}

		t.Codec = &CodecMPEG4Audio{
			Config: *conf,
		}
		return nil

	case astits.StreamTypeMPEG1Audio:
		t.Codec = &CodecMPEG1Audio{}
		return nil

	case astits.StreamTypeAC3Audio:
		sampleRate, channelCount, err := findAC3Parameters(dem, es.ElementaryPID)
		if err != nil {
			return err
		}

		t.Codec = &CodecAC3{
			SampleRate:   sampleRate,
			ChannelCount: channelCount,
		}
		return nil

	case astits.StreamTypePrivateData:
		codec := findOpusCodec(es.ElementaryStreamDescriptors)
		if codec != nil {
			t.Codec = codec
			return nil
		} else if findKLVARegistration(es.ElementaryStreamDescriptors) {

			for {
				data, err := dem.NextData()

				if err != nil {
					return err
				}

				if data.PES == nil || data.PID != t.PID {
					continue
				}
				t.Codec = &CodecKLV{
					StreamType:      astits.StreamTypePrivateData,
					PTSDTSIndicator: data.PES.Header.OptionalHeader.PTSDTSIndicator,
					StreamID:        data.PES.Header.StreamID,
				}

				break
			}

			return nil
		}

	case astits.StreamTypeMetadata:
		codec := &CodecKLV{
			StreamType:      astits.StreamTypeMetadata,
			StreamID:        0xFC,
			PTSDTSIndicator: astits.PTSDTSIndicatorOnlyPTS,
		}
		t.Codec = codec

		return nil
	}

	return errUnsupportedCodec
}
