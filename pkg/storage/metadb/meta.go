package metadb

import "errors"

var (
	ErrAccountDoesntExist   = errors.New("account doesn't exist, create one")
	ErrProjectDoesntExist   = errors.New("project doesn't exist, create one")
	ErrParentDirDoesntExist = errors.New("parent directory doesn't exist, create it first")
)
