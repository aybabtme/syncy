package dirsync

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/stretchr/testify/require"
)

func TestTraceSource(t *testing.T) {
	regFile := fs.FileMode(fs.ModeAppend)
	require.True(t, regFile.IsRegular())
	tests := []struct {
		name string
		base string
		in   fstest.MapFS
		want *SourceDir
	}{
		{
			name: "base",
			base: "root",
			in: fstest.MapFS{
				"root/hello/world":       &fstest.MapFile{Mode: regFile, Data: []byte("hello world")},
				"root/hello/le_monde":    &fstest.MapFile{Mode: regFile, Data: []byte("hello le monde")},
				"root/hello/fr/le_monde": &fstest.MapFile{Mode: regFile, Data: []byte("hello le monde")},
				"root/en/world":          &fstest.MapFile{Mode: regFile, Data: []byte("hello world")},
				"root/world":             &fstest.MapFile{Mode: regFile, Data: []byte("hello world")},
			},
			want: &SourceDir{
				Name: "root",
				Mode: uint32(2147484013),
				Size: 61,
				Dirs: []*SourceDir{
					{
						Name: "en",
						Mode: uint32(2147484013),
						Size: 11,
						Files: []*SourceFile{
							{Info: &typesv1.FileInfo{Name: "world", Mode: uint32(regFile), Size: 11}},
						},
					},
					{
						Name: "hello",
						Mode: uint32(2147484013),
						Size: 39,
						Dirs: []*SourceDir{
							{
								Name: "fr",
								Mode: uint32(2147484013),
								Size: 14,
								Files: []*SourceFile{
									{Info: &typesv1.FileInfo{Name: "le_monde", Mode: uint32(regFile), Size: 14}},
								},
							},
						},
						Files: []*SourceFile{
							{Info: &typesv1.FileInfo{Name: "le_monde", Mode: uint32(regFile), Size: 14}},
							{Info: &typesv1.FileInfo{Name: "world", Mode: uint32(regFile), Size: 11}},
						},
					},
				},
				Files: []*SourceFile{
					{Info: &typesv1.FileInfo{Name: "world", Mode: uint32(regFile), Size: 11}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := TraceSource(ctx, tt.base, tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
