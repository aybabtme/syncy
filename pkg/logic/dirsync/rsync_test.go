package dirsync

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRsync(t *testing.T) {
	tests := []struct {
		name        string
		src         *bytes.Buffer
		blockSize   uint32
		init        *bytes.Reader
		wantPatches []any
	}{
		{
			name:      "holes",
			src:       bytes.NewBuffer([]byte("hello world")),
			blockSize: 4,
			init:      bytes.NewReader([]byte("    o wor")),
			wantPatches: []any{
				"hell",
				uint32(1),
				uint32(2),
				"ld",
			},
		},
		{
			name:      "missing sufix",
			src:       bytes.NewBuffer([]byte("hello world")),
			blockSize: 4,
			init:      bytes.NewReader([]byte("hello")),
			wantPatches: []any{
				uint32(0),
				uint32(1),
				" world",
			},
		},
		{
			name:      "missing prefix",
			src:       bytes.NewBuffer([]byte("hello world")),
			blockSize: 4,
			init:      bytes.NewReader([]byte("world")),
			wantPatches: []any{
				"hello ",
				uint32(0),
				uint32(1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			want := tt.src.String()
			orig := tt.init

			sum, err := computeFileSum(ctx, orig, nil, tt.blockSize)
			require.NoError(t, err)
			_, err = orig.Seek(0, io.SeekStart)
			require.NoError(t, err)

			dst := bytes.NewBuffer(nil)

			patcher := NewFilePatcher(orig, dst, sum)

			var gotPatches []any
			_, err = Rsync(ctx, tt.src, sum,
				func(data []byte) (int, error) {
					if len(data) == 0 {
						panic("what is this")
					}
					n, err := patcher.Write(data)
					gotPatches = append(gotPatches, string(data))
					return n, err
				},
				func(blockID uint32) (int, error) {
					gotPatches = append(gotPatches, blockID)
					n, err := patcher.WriteBlock(blockID)
					return n, err
				},
			)
			require.NoError(t, err)
			got := dst.String()

			require.Equal(t, want, got)
			require.Equal(t, tt.wantPatches, gotPatches)
		})
	}
}
