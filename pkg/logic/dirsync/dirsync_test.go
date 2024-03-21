package dirsync

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestTrace(t *testing.T) {
	tests := []struct {
		name string
		base string
		in   fstest.MapFS
		want *Dir
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
			want: &Dir{
				Name: "root",
				Dirs: []*Dir{
					{
						Name: "en",
						Files: []*File{
							{Name: "world"},
						},
					},
					{
						Name: "hello",
						Dirs: []*Dir{
							{
								Name: "fr",
								Files: []*File{
									{Name: "le_monde"},
								},
							},
						},
						Files: []*File{
							{Name: "le_monde"},
							{Name: "world"},
						},
					},
				},
				Files: []*File{
					{Name: "world"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := Trace(ctx, tt.base, tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
