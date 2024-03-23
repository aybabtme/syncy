package dirsync

import (
	"context"
	"encoding/hex"
	"testing"
	"testing/fstest"
	"time"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestComputeFileSum(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		in        *fstest.MapFile
		blockSize uint32
		want      *typesv1.FileSum
	}{
		{
			name:     "single short block",
			filename: "hello.txt",
			in: &fstest.MapFile{
				Data:    []byte("hello world"),
				ModTime: time.Date(2023, 3, 21, 17, 42, 58, 0, time.UTC),
			},
			blockSize: 32,
			want: &typesv1.FileSum{
				Info: &typesv1.FileInfo{
					Name:    "hello.txt",
					Size:    11,
					ModTime: timestamppb.New(time.Date(2023, 3, 21, 17, 42, 58, 0, time.UTC)),
				},
				BlockSize: 32,
				SumBlocks: []*typesv1.FileSumBlock{
					{
						Size:      11,
						FastSig:   3468377999,
						StrongSig: hexTo32byte("d74981efa70a0c880b8d8c1985d075dbcbf679b99a5f9914e5aaf96b831a9e24"),
					},
				},
			},
		},
		{
			name:     "across blocks",
			filename: "hello.txt",
			in: &fstest.MapFile{
				Data:    []byte("hello world, how are you doing today?"),
				ModTime: time.Date(2023, 3, 21, 17, 42, 58, 0, time.UTC),
			},
			blockSize: 16,
			want: &typesv1.FileSum{
				Info: &typesv1.FileInfo{
					Name:    "hello.txt",
					Size:    37,
					ModTime: timestamppb.New(time.Date(2023, 3, 21, 17, 42, 58, 0, time.UTC)),
				},
				BlockSize: 16,
				SumBlocks: []*typesv1.FileSumBlock{
					{
						Size:      16,
						FastSig:   2602787317,
						StrongSig: hexTo32byte("b88aed0ebc9f5b98d92237e2f00916d7479b0c88ce01a35c387f7a83e18cde79"),
					},
					{
						Size:      16,
						FastSig:   1782890572,
						StrongSig: hexTo32byte("6c8edc89526634d0e2e8bd280531b75d5c366babf023456636ad9403aa74aa14"),
					},
					{
						Size:      5,
						FastSig:   3249705445,
						StrongSig: hexTo32byte("acd98d44cc8ca272607e32c9f3beb73c47f522b150288cd278212c61a63e5313"),
					},
				},
			},
		},
		{
			name:     "flipped blocks",
			filename: "hello.txt",
			in: &fstest.MapFile{
				Data:    []byte("abcd12341234abcd"),
				ModTime: time.Date(2023, 3, 21, 17, 42, 58, 0, time.UTC),
			},
			blockSize: 4,
			want: &typesv1.FileSum{
				Info: &typesv1.FileInfo{
					Name:    "hello.txt",
					Size:    16,
					ModTime: timestamppb.New(time.Date(2023, 3, 21, 17, 42, 58, 0, time.UTC)),
				},
				BlockSize: 4,
				SumBlocks: []*typesv1.FileSumBlock{
					{
						Size:      4,
						FastSig:   23584314,
						StrongSig: hexTo32byte("8c9c9881805d1a847102d7a42e58b990d088dd88a84f7314d71c838107571f2b"),
					},
					{
						Size:      4,
						FastSig:   4036357995,
						StrongSig: hexTo32byte("cde13a55f41e387480391c47238acfe9c0136dd56bf365b01416aec03eec7dc4"),
					},
					{
						Size:      4,
						FastSig:   4036357995,
						StrongSig: hexTo32byte("cde13a55f41e387480391c47238acfe9c0136dd56bf365b01416aec03eec7dc4"),
					},
					{
						Size:      4,
						FastSig:   23584314,
						StrongSig: hexTo32byte("8c9c9881805d1a847102d7a42e58b990d088dd88a84f7314d71c838107571f2b"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fs := fstest.MapFS{tt.filename: tt.in}
			f, err := fs.Open(tt.filename)
			require.NoError(t, err)
			defer f.Close()
			fi, err := f.Stat()
			require.NoError(t, err)

			got, err := computeFileSum(ctx, f, typesv1.FileInfoFromFS(fi), tt.blockSize)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func hexTo32byte(h string) *typesv1.Uint256 {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	if len(b) != 32 {
		panic(len(b))
	}
	return typesv1.Uint256FromArray32Byte([32]byte(b))
}
