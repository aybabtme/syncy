package dirsync

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestTraceSource(t *testing.T) {
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
				"root/hello/world":       &fstest.MapFile{Data: []byte("hello world")},
				"root/hello/le_monde":    &fstest.MapFile{Data: []byte("hello le monde")},
				"root/hello/fr/le_monde": &fstest.MapFile{Data: []byte("hello le monde")},
				"root/en/world":          &fstest.MapFile{Data: []byte("hello world")},
				"root/world":             &fstest.MapFile{Data: []byte("hello world")},
			},
			want: &SourceDir{
				Name: "root",
				Mode: 2147484013,
				Dirs: []*SourceDir{
					{
						Name: "en",
						Mode: 2147484013,
						Files: []*SourceFile{
							{Name: "world", Mode: 2147484013},
						},
					},
					{
						Name: "hello",
						Mode: 2147484013,
						Dirs: []*SourceDir{
							{
								Name: "fr",
								Mode: 2147484013,
								Files: []*SourceFile{
									{Name: "le_monde", Mode: 2147484013},
								},
							},
						},
						Files: []*SourceFile{
							{Name: "le_monde", Mode: 2147484013},
							{Name: "world", Mode: 2147484013},
						},
					},
				},
				Files: []*SourceFile{
					{Name: "world", Mode: 2147484013},
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
