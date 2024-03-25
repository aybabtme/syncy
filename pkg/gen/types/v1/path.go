package typesv1

import "path/filepath"

func PathFromString(path string) *Path {
	return &Path{Elements: filepath.SplitList(path)}
}

func PathJoin(parent *Path, name string) *Path {
	if parent == nil {
		return &Path{Elements: []string{name}}
	}
	return &Path{Elements: append(parent.Elements, name)}
}

func StringFromPath(path *Path) string {
	return filepath.Join(path.Elements...)
}
