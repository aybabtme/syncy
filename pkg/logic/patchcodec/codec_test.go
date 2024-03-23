package patchcodec

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodec(t *testing.T) {
	tests := []struct {
		name string
		want []any
	}{
		{
			name: "base",
			want: []any{
				uint32(0),
				uint32(1),
				"hello",
				uint32(4),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := bytes.NewBuffer(nil)
			enc := NewEncoder(w)
			wantN := 0
			for _, op := range tt.want {
				switch p := op.(type) {
				case uint32:
					_, err := enc.WriteBlockID(p)
					require.NoError(t, err)
					wantN += 4
				case string:
					_, err := enc.WriteBlock([]byte(p))
					require.NoError(t, err)
					wantN += len(p)
				}
			}
			wantBytes := w.String()
			var got []any
			gotN, err := NewDecoder(w).Decode(
				func(u uint32) (int, error) {
					got = append(got, u)
					return 4, nil
				},
				func(r io.Reader) (int, error) {
					block, err := io.ReadAll(r)
					require.NoError(t, err)
					got = append(got, string(block))
					return len(block), nil
				},
			)
			require.NoError(t, err)
			require.Equal(t, wantN, gotN)
			require.Equal(t, tt.want, got)

			// check that reencoding gives the same bytes

			w.Reset()
			enc = NewEncoder(w)
			for _, op := range tt.want {
				switch p := op.(type) {
				case uint32:
					_, err := enc.WriteBlockID(p)
					require.NoError(t, err)
				case string:
					_, err := enc.WriteBlock([]byte(p))
					require.NoError(t, err)
				}
			}
			gotBytes := w.String()
			require.Equal(t, wantBytes, gotBytes)
		})
	}
}
