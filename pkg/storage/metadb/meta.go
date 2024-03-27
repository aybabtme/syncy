package metadb

import "errors"

var (
	ErrAccountDoesntExist = errors.New("account doesn't exist, create one")
	ErrProjectDoesntExist = errors.New("project doesn't exist, create one")
)
