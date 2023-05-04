package av1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitstreamUnmarshal(t *testing.T) {
	for _, ca := range []struct {
		name string
		enc  []byte
		dec  [][]byte
	}{
		{
			"standard",
			[]byte{
				0x0a, 0x0e, 0x00, 0x00, 0x00, 0x4a, 0xab, 0xbf,
				0xc3, 0x77, 0x6b, 0xe4, 0x40, 0x40, 0x40, 0x41,
				0x0a, 0x0e, 0x00, 0x00, 0x00, 0x4a, 0xab, 0xbf,
				0xc3, 0x77, 0x6b, 0xe4, 0x40, 0x40, 0x40, 0x41,
			},
			[][]byte{
				{
					0x08, 0x00, 0x00, 0x00, 0x4a, 0xab, 0xbf, 0xc3,
					0x77, 0x6b, 0xe4, 0x40, 0x40, 0x40, 0x41,
				},
				{
					0x08, 0x00, 0x00, 0x00, 0x4a, 0xab, 0xbf, 0xc3,
					0x77, 0x6b, 0xe4, 0x40, 0x40, 0x40, 0x41,
				},
			},
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			dec, err := BitstreamUnmarshal(ca.enc, true)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
		})
	}
}

func FuzzBitstreamUnmarshal(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		BitstreamUnmarshal(b, true)
	})
}
