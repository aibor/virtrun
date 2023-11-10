package files

import (
	"errors"
)

var (
	// ErrEntryNotDir is returned if an entry is supposed to be a directory but is
	// not.
	ErrEntryNotDir = errors.New("entry is not a directory")
	// ErrEntryNotExists is returned if an entry that is looked up does not exist.
	ErrEntryNotExists = errors.New("entry does not exist")
	// ErrEntryExists is returned if an entry exists that was not expected.
	ErrEntryExists = errors.New("entry exists")
)
