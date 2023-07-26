package h264

import (
	"fmt"
)

// AVCCUnmarshal decodes NALUs from the AVCC stream format.
// Specification: ?
func AVCCUnmarshal(buf []byte) ([][]byte, error) {
	bl := len(buf)
	pos := 0
	var ret [][]byte

	for {
		if (bl - pos) < 4 {
			return nil, fmt.Errorf("invalid length")
		}

		l := int(uint32(buf[pos])<<24 | uint32(buf[pos+1])<<16 | uint32(buf[pos+2])<<8 | uint32(buf[pos+3]))
		pos += 4

		if l == 0 {
			return nil, fmt.Errorf("invalid NALU")
		}

		if l > MaxNALUSize {
			return nil, fmt.Errorf("NALU size (%d) is too big, maximum is %d", l, MaxNALUSize)
		}

		if (len(ret) + 1) > MaxNALUsPerGroup {
			return nil, fmt.Errorf("NALU count (%d) exceeds maximum allowed (%d)",
				len(ret)+1, MaxNALUsPerGroup)
		}

		if (bl - pos) < l {
			return nil, fmt.Errorf("invalid length")
		}

		ret = append(ret, buf[pos:pos+l])
		pos += l

		if (bl - pos) == 0 {
			break
		}
	}

	return ret, nil
}

func avccMarshalSize(nalus [][]byte) int {
	n := 0
	for _, nalu := range nalus {
		n += 4 + len(nalu)
	}
	return n
}

// AVCCMarshal encodes NALUs into the AVCC stream format.
// Specification: ?
func AVCCMarshal(nalus [][]byte) ([]byte, error) {
	buf := make([]byte, avccMarshalSize(nalus))
	pos := 0

	for _, nalu := range nalus {
		naluLen := len(nalu)
		buf[pos] = byte(naluLen >> 24)
		buf[pos+1] = byte(naluLen >> 16)
		buf[pos+2] = byte(naluLen >> 8)
		buf[pos+3] = byte(naluLen)
		pos += 4

		pos += copy(buf[pos:], nalu)
	}

	return buf, nil
}
