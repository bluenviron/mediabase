package mpegts

import (
	"fmt"

	"github.com/asticode/go-astits"
	"github.com/bluenviron/mediacommon/pkg/codecs/mpeg4audio"
)

const (
	h264Identifier = 'H'<<24 | 'D'<<16 | 'M'<<8 | 'V'
	h265Identifier = 'H'<<24 | 'E'<<16 | 'V'<<8 | 'C'
	opusIdentifier = 'O'<<24 | 'p'<<16 | 'u'<<8 | 's'
)

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
			return nil, fmt.Errorf("unable to decode ADTS: %s", err)
		}

		pkt := adtsPkts[0]
		return &mpeg4audio.Config{
			Type:         pkt.Type,
			SampleRate:   pkt.SampleRate,
			ChannelCount: pkt.ChannelCount,
		}, nil
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

func findOpusChannels(descriptors []*astits.Descriptor) int {
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

	channels := findOpusChannels(descriptors)
	if channels <= 0 {
		return nil
	}

	return &CodecOpus{
		Channels: channels,
	}
}

// Track is a MPEG-TS track.
type Track struct {
	ES    *astits.PMTElementaryStream
	Codec Codec
}

// FindTracks finds the tracks in a MPEG-TS stream.
func FindTracks(dem *astits.Demuxer) ([]*Track, error) {
	var tracks []*Track

	for {
		data, err := dem.NextData()
		if err != nil {
			return nil, err
		}

		if data.PMT != nil {
			for _, es := range data.PMT.ElementaryStreams {
				track := &Track{
					ES: es,
				}

				switch es.StreamType {
				case astits.StreamTypeH264Video:
					track.Codec = &CodecH264{}
					tracks = append(tracks, track)

				case astits.StreamTypeH265Video:
					track.Codec = &CodecH265{}
					tracks = append(tracks, track)

				case astits.StreamTypeAACAudio:
					conf, err := findMPEG4AudioConfig(dem, es.ElementaryPID)
					if err != nil {
						return nil, err
					}

					track.Codec = &CodecMPEG4Audio{
						Config: *conf,
					}
					tracks = append(tracks, track)

				case astits.StreamTypePrivateData:
					codec := findOpusCodec(es.ElementaryStreamDescriptors)
					if codec != nil {
						track.Codec = codec
						tracks = append(tracks, track)
					}
				}
			}
			break
		}
	}

	if tracks == nil {
		return nil, fmt.Errorf("no tracks found")
	}

	return tracks, nil
}
