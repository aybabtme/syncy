package typesv1

import (
	"path/filepath"
	"strings"
)

func PathFromString(path string) *Path {
	var out []string
	for _, el := range strings.Split(path, "/") {
		if el == "" {
			continue
		}
		if el[0] == '/' {
			el = el[1:]
		}
		if el[len(el)-1] == '/' {
			el = el[:len(el)-1]
		}
		out = append(out, el)
	}
	return &Path{Elements: out}
}

func DirOf(path *Path) *Path {
	n := len(path.Elements)
	if n < 1 {
		return &Path{}
	}
	return &Path{Elements: path.Elements[:n-1]}
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
