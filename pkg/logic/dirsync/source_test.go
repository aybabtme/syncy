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
				Dirs: []*SourceDir{
					{
						Name: "en",
						Files: []*SourceFile{
							{Name: "world"},
						},
					},
					{
						Name: "hello",
						Dirs: []*SourceDir{
							{
								Name: "fr",
								Files: []*SourceFile{
									{Name: "le_monde"},
								},
							},
						},
						Files: []*SourceFile{
							{Name: "le_monde"},
							{Name: "world"},
						},
					},
				},
				Files: []*SourceFile{
					{Name: "world"},
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
