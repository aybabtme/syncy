package dirsync

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTraceSource(t *testing.T) {
	nowpb := timestamppb.New(time.Unix(-62135596800, 0))
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
				Info: &typesv1.FileInfo{
					Name:    "root",
					Mode:    uint32(2147484013),
					Size:    61,
					IsDir:   true,
					ModTime: nowpb,
				},
				Dirs: []*SourceDir{
					{
						Info: &typesv1.FileInfo{
							Name:    "en",
							Mode:    uint32(2147484013),
							Size:    11,
							IsDir:   true,
							ModTime: nowpb,
						},
						Files: []*SourceFile{
							{Info: &typesv1.FileInfo{
								Name: "world", Mode: uint32(regFile), Size: 11,
								ModTime: nowpb,
							}},
						},
					},
					{
						Info: &typesv1.FileInfo{
							Name:    "hello",
							Mode:    uint32(2147484013),
							Size:    39,
							IsDir:   true,
							ModTime: nowpb,
						},
						Dirs: []*SourceDir{
							{
								Info: &typesv1.FileInfo{
									Name:    "fr",
									Mode:    uint32(2147484013),
									Size:    14,
									IsDir:   true,
									ModTime: nowpb,
								},
								Files: []*SourceFile{
									{Info: &typesv1.FileInfo{
										Name: "le_monde", Mode: uint32(regFile), Size: 14,
										ModTime: nowpb,
									}},
								},
							},
						},
						Files: []*SourceFile{
							{Info: &typesv1.FileInfo{
								Name: "le_monde", Mode: uint32(regFile), Size: 14,
								ModTime: nowpb,
							}},
							{Info: &typesv1.FileInfo{
								Name: "world", Mode: uint32(regFile), Size: 11,
								ModTime: nowpb,
							}},
						},
					},
				},
				Files: []*SourceFile{
					{Info: &typesv1.FileInfo{
						Name: "world", Mode: uint32(regFile), Size: 11,
						ModTime: nowpb,
					}},
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
